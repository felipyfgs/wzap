## 1. wa/manager.go — Cache e infraestrutura de retry

- [x] 1.1 Adicionar struct `mediaRetryCacheEntry` com campos: `sessionID`, `chatJID`, `senderJID`, `fromMe`, `mimeType`, `timestamp`, `mediaKey`, `expiresAt`
- [x] 1.2 Adicionar campo `mediaRetryCache sync.Map` ao `Manager`
- [x] 1.3 Declarar tipo `MediaRetryFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int)`
- [x] 1.4 Adicionar campo `OnMediaRetry MediaRetryFunc` ao `Manager`
- [x] 1.5 Adicionar `SetMediaRetry(fn MediaRetryFunc)` ao Manager
- [x] 1.6 Implementar `Manager.RequestMediaRetry(ctx, sessionID, *types.MessageInfo, mediaKey)` — chama `SendMediaRetryReceipt` e guarda no cache

## 2. wa/events.go — Handler de *events.MediaRetry

- [x] 2.1 No `case *events.MediaRetry`: buscar entrada no cache, verificar TTL
- [x] 2.2 Chamar `whatsmeow.DecryptMediaRetryNotification(evt, entry.mediaKey)`
- [x] 2.3 Se `SUCCESS`: extrair `DirectPath` e demais campos, chamar `m.OnMediaRetry` callback
- [x] 2.4 Se `ErrMediaNotAvailableOnPhone`: logar warn e remover do cache
- [x] 2.5 Remover entrada do cache após processar (sucesso ou erro conhecido)

## 3. service/history.go — Interface e fallback no download

- [x] 3.1 Declarar interface `MediaRetryRequester` com método `RequestMediaRetry`
- [x] 3.2 Adicionar campo `retryRequester MediaRetryRequester` ao `HistoryService`
- [x] 3.3 Adicionar `SetMediaRetryRequester(r MediaRetryRequester)` ao HistoryService
- [x] 3.4 Em `PersistHistorySync`, após falha de download, verificar se erro é 403/404/410
- [x] 3.5 Se sim e `retryRequester != nil`: chamar `RequestMediaRetry` com o `MessageInfo` e `mediaKey`

## 4. service/media.go — RetryMediaUpload

- [x] 4.1 Implementar `MediaService.RetryMediaUpload(sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int)`
- [x] 4.2 Usar pool async para não bloquear
- [x] 4.3 Baixar com `DownloadMediaByPath` usando o novo `directPath`
- [x] 4.4 Gerar `MediaObjectKey` com os metadados
- [x] 4.5 Fazer upload para MinIO
- [x] 4.6 Chamar `msgRepo.UpdateMediaURL` para persistir a chave S3

## 5. server/router.go — Wiring

- [x] 5.1 Chamar `engine.SetMediaRetry(mediaSvc.RetryMediaUpload)`
- [x] 5.2 Chamar `historySvc.SetMediaRetryRequester(engine)`
