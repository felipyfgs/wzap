## Context

O wzap captura eventos `HistorySync` via whatsmeow e os persiste em `wz_messages`/`wz_chats` através do `HistoryService` (`internal/service/history.go`). A integração Chatwoot (`internal/integrations/chatwoot/`) recebe esses eventos via `OnEvent` → `handleHistorySync`, mas o handler atual é no-op. O método `importHistory` em `service.go` é um placeholder.

Fluxo atual de mídia ao vivo: `wa.Manager.OnMediaReceived` → `MediaService.AutoUploadMedia` → whatsmeow `Download` → `storage.Minio.Upload`. Este mesmo padrão pode ser reaproveitado para mídias do histórico.

O `MediaDownloader` injetado no Chatwoot Service usa `wa.Manager.DownloadMediaByPath`, que suporta download por `directPath` + chaves de criptografia — exatamente o que as mensagens de histórico fornecem via proto.

## Goals / Non-Goals

**Goals:**
- Download e armazenamento de mídias do histórico no MinIO imediatamente ao receber `HistorySync`
- Implementação completa do `importHistory`: iterar mensagens locais não importadas, criar entidades no Chatwoot via API REST
- Hierarquia de nomes de contato: Nome da Agenda WA > PushName > Telefone
- Endpoint `POST /sessions/{id}/integrations/chatwoot/import` funcional

**Non-Goals:**
- Acesso direto ao banco do Chatwoot
- Importação de mensagens de grupos ou status
- Re-download de mídias já presentes no MinIO
- Nova migration de banco (campos já existem)

## Decisions

### D1: Download de mídia dentro de `PersistHistorySync` (e não lazy)

**Escolha**: No momento em que `PersistHistorySync` processa cada mensagem de histórico, se a mensagem contém mídia (image/video/audio/document/sticker), fazer o download síncrono via `wa.Manager.DownloadMediaByPath` e upload para MinIO, salvando a URL em `media_url`.

**Alternativa considerada**: Download lazy (apenas quando `importHistory` precisar). Rejeitada porque as media keys do histórico expiram em ~2 semanas, e o import pode ocorrer muito depois.

**Fluxo de dados**:
```
HistorySync event → PersistHistorySync()
  → buildHistoryMessage() [existente]
  → se msg tem mídia (Raw.(*waWeb.WebMessageInfo).GetMessage()):
      → manager.DownloadMediaByPath(directPath, encFileHash, fileHash, mediaKey, fileLength, mediaType)
      → minio.Upload(key, data, mimeType)
      → msg.MediaURL = presignedURL
  → messageRepo.Save(msg)
```

**Dependência**: `HistoryService` precisa receber referência ao `wa.Manager` e ao `storage.Minio` (ou interface `MediaStorage`). Hoje recebe apenas `messageRepo` e `chatRepo`.

### D2: Extração de mídia do proto `waE2E.Message`

**Escolha**: Adicionar função `extractMediaDownloadInfo(msg *waE2E.Message) (directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, ok bool)` que retorna os campos de download para cada tipo de mídia suportado. Segue o mesmo padrão de `extractMessageContent` já existente.

**Tipos suportados**: ImageMessage, VideoMessage, AudioMessage, DocumentMessage, StickerMessage. Cada um tem `GetDirectPath()`, `GetFileEncSHA256()`, `GetFileSHA256()`, `GetMediaKey()`, `GetFileLength()`.

### D3: Importação via API REST do Chatwoot (não SQL direto)

**Escolha**: O `importHistory` itera mensagens do `wz_messages` onde `source='history_sync'` e `imported_to_chatwoot_at IS NULL`, ordenadas por `timestamp ASC`. Para cada chatJID distinto, chama `findOrCreateConversation` (já existente) e então `CreateMessage` ou `CreateMessageWithAttachment`.

**Alternativa considerada**: INSERT direto no Postgres do Chatwoot (como Evolution API). Rejeitada: frágil, acoplada ao schema, não dispara webhooks/eventos do Chatwoot.

**Fluxo de dados**:
```
importHistory(sessionID, period)
  → msgRepo.FindUnimportedHistory(sessionID, since) []Message
  → para cada msg (rate-limited 10/s via rateTicker):
      → findOrCreateConversation(cfg, msg.ChatJID, contactName)
      → se msg.MsgType == "text":
          → client.CreateMessage(convID, {content, messageType, sourceID})
      → se msg tem mediaURL:
          → download do MinIO via presignedURL
          → client.CreateMessageWithAttachment(convID, ...)
      → msgRepo.MarkImportedToChatwoot(sessionID, msg.ID, now)
```

### D4: Query de mensagens não importadas

**Escolha**: Novo método `MessageRepo.FindUnimportedHistory(ctx, sessionID, since, limit, offset) ([]Message, error)` que retorna mensagens com `source = 'history_sync'` e `imported_to_chatwoot_at IS NULL` e `timestamp >= since`, ordenadas por `timestamp ASC, id ASC`. Paginação com `limit/offset` para processar em chunks de 100.

### D5: Proteção contra import concorrente

**Escolha**: Usar `singleflight.Group` com chave `"import:" + sessionID` no `Service`. Se um import já está em execução para aquela sessão, a segunda chamada aguarda e recebe o mesmo resultado. Isso evita imports duplicados tanto do `handleConnected` (auto-import) quanto do endpoint manual.

### D6: Hierarquia de nomes de contato

**Escolha**: Interface `ContactNameGetter` com método `GetContactName(ctx, sessionID, jid) string` que consulta `client.Store.Contacts` para obter `FullName`/`FirstName` do contato na agenda do WhatsApp.

**Prioridade**: `FullName (agenda WA)` > `FirstName (agenda WA)` > `PushName` > `phone`

**Alterações**:
1. `handlePushName`: não sobrescrever nome de contato que já tem nome da agenda (não-numérico, diferente do telefone)
2. `upsertConversation`: consultar `ContactNameGetter` antes de usar `pushName`
3. `handleContact` (já existente): já recebe nome da agenda via evento `Contact`, manter como está

### D7: Endpoint de importação

**Escolha**: Implementar `Handler.ImportHistory` que chama `Service.importHistory` em goroutine separada, retornando `202 Accepted` imediatamente com o estado atual do progresso via métrica Prometheus `cw_history_import_progress`.

## Risks / Trade-offs

| Risco | Mitigação |
|---|---|
| Download de mídia bloqueia worker do async.Pool | Usar `s.pool.Submit` com timeout individual de 30s por mídia; erros não impedem persistência do texto |
| Volume alto de mensagens (milhares) no import | Rate limiter de 10 msgs/s + chunks de 100 com offset; métrica Prometheus de progresso |
| Import parcial por erro no meio | Idempotente: `imported_to_chatwoot_at` marca o que foi importado; re-executar retoma |
| Mídia do histórico já expirou no servidor WA | Log Warn + fallback: criar mensagem texto com "[Mídia expirada: {tipo}]" |
| `singleflight` bloqueia import concorrente por até 10 min | Timeout de 10 min no context; chamada simultânea recebe mesmo resultado |
| `CreateMessageWithAttachment` com mídia grande do MinIO | Download via presigned URL (stream), limitar a 50MB por arquivo |
