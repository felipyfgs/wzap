## Context

O whatsmeow expõe dois métodos para recuperação de mídia expirada:
- `client.SendMediaRetryReceipt(ctx, *types.MessageInfo, mediaKey []byte)` — envia receipt ao WA solicitando re-upload pelo celular
- `whatsmeow.DecryptMediaRetryNotification(*events.MediaRetry, mediaKey []byte)` — decripta a resposta com o novo `DirectPath`

O fluxo é assíncrono: a solicitação é enviada e a resposta chega como evento `*events.MediaRetry` em tempo indeterminado (geralmente segundos). Para correlacionar a resposta com a solicitação original, precisamos de um cache keyed por `messageID`.

Atualmente o código em `history.go` descarta qualquer mídia que falhe no download sem tentar recuperação.

## Goals / Non-Goals

**Goals:**
- Reduzir perda de mídia no history sync para casos onde o celular está online
- Reutilizar infraestrutura existente (pool async, MinIO, `UpdateMediaURL`)
- Manter a lógica desacoplada: o `wa.Manager` não conhece MinIO, o `MediaService` não conhece whatsmeow events

**Non-Goals:**
- Persistência de retry entre restarts
- Retry de mídias ao vivo (mensagens recebidas em tempo real)

## Decisions

### Cache em memória com TTL implícito

Estrutura armazenada em `wa.Manager` como `sync.Map`:
```
messageID → mediaRetryCacheEntry{
    sessionID  string
    chatJID    string
    senderJID  string
    fromMe     bool
    mimeType   string
    timestamp  time.Time
    mediaKey   []byte
    expiresAt  time.Time   // agora + 10min
}
```

Limpeza: verificação de TTL no momento do lookup (`Get`). Sem goroutine de GC separada — entradas expiradas são detectadas e removidas na hora da leitura.

### Callback `MediaRetryFunc`

Mesmo padrão do `MediaAutoUploadFunc` já existente:
```go
type MediaRetryFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int)
```

`wa.Manager` chama o callback quando `*events.MediaRetry` é bem-sucedido e decriptado. O `MediaService` implementa essa função e executa o download + upload MinIO + `UpdateMediaURL`.

### `Manager.RequestMediaRetry`

Novo método público no Manager:
```go
func (m *Manager) RequestMediaRetry(ctx context.Context, sessionID string, info *types.MessageInfo, mediaKey []byte) error
```

Chamado pelo `HistoryService` quando o download falha com erro 403/404/410. Internamente:
1. Chama `client.SendMediaRetryReceipt`
2. Guarda entrada no `mediaRetryCache`

### Detecção de erros 403/404/410

Em `history.go`, comparar o erro retornado por `DownloadMediaByPath` com `whatsmeow.ErrMediaDownloadFailedWith403`, `ErrMediaDownloadFailedWith404`, `ErrMediaDownloadFailedWith410`.

### Interface `MediaRetryRequester` no HistoryService

Para evitar dependência direta em `*wa.Manager`:
```go
type MediaRetryRequester interface {
    RequestMediaRetry(ctx context.Context, sessionID string, info *types.MessageInfo, mediaKey []byte) error
}
```

`HistoryService` recebe essa interface via `SetMediaRetryRequester(r MediaRetryRequester)`.

### `ParseWebMessage` para obter `types.MessageInfo`

Para chamar `SendMediaRetryReceipt`, precisamos de `*types.MessageInfo`. O `wa.Manager` expõe `ParseWebMessage(chatJID, *waProto.WebMessageInfo) (*events.Message, error)` do whatsmeow.

## Risks / Trade-offs

| Risco | Impacto | Mitigação |
|---|---|---|
| Celular offline durante history sync | Retry nunca responde | TTL 10min; media fica sem URL (comportamento atual) |
| `ErrMediaNotAvailableOnPhone` (código 2) | Mídia deletada do celular | Logar warn e descartar |
| Histórico muito antigo | Celular pode não ter mais a mídia | Idem acima |
| Cache grande em memória | Baixo — cada entrada ~200 bytes | Max ~50k mensagens simultâneas = ~10MB |
