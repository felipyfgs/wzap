## Why

Durante o `history_sync`, as URLs de CDN do WhatsApp para download de mídias expiram em minutos. O sistema atualmente descarta silenciosamente qualquer mídia que retorne 403/404/410, resultando em perda irreversível de anexos históricos. O whatsmeow oferece `SendMediaRetryReceipt` — um mecanismo oficial para pedir ao celular do usuário que re-faça o upload ao CDN — que nunca foi aproveitado.

## What Changes

- **Novo**: cache em memória (`sync.Map` com TTL) para rastrear mídias com retry pendente
- **Novo**: callback `MediaRetryFunc` no `wa.Manager` para processar `*events.MediaRetry`
- **Novo**: método `Manager.RequestMediaRetry` que chama `client.SendMediaRetryReceipt`
- **Novo**: método `MediaService.RetryMediaUpload` para re-download e persistência após retry bem-sucedido
- **Modificado**: `HistoryService.PersistHistorySync` — ao falhar com 403/404/410, envia retry em vez de descartar
- **Modificado**: `wa/events.go` — handler `*events.MediaRetry` passa a chamar o callback em vez de apenas logar

## Capabilities

### New Capabilities
- `media-retry`: Recuperação automática de mídias expiradas via `SendMediaRetryReceipt` + processamento assíncrono do `events.MediaRetry`

### Modified Capabilities

## Impact

- `internal/wa/manager.go` — novos campos e método
- `internal/wa/events.go` — handler real para `*events.MediaRetry`
- `internal/service/history.go` — lógica de fallback ao falhar download
- `internal/service/media.go` — novo método de retry
- `internal/server/router.go` — wiring do callback

## Não-objetivos

- Não persistir cache de retry em banco de dados
- Não implementar retry para mídias de mensagens ao vivo (apenas history sync)
- Não reenviar retries após restart do servidor

## Riscos e mitigações

| Risco | Mitigação |
|---|---|
| Celular offline — sem resposta | TTL de 10min; entradas expiram automaticamente |
| `ErrMediaNotAvailableOnPhone` | Logar e remover do cache sem retry |
| Race condition no cache | `sync.Map` é thread-safe por definição |
| Volume alto de retries | Re-download usa o pool async existente |
