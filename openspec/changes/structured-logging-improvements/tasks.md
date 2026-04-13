## 1. Logger Base — Timestamp e Configuração

- [x] 1.1 Configurar `zerolog.TimeFieldFormat = time.RFC3339` e `ConsoleWriter.TimeFormat = "2006-01-02 15:04:05"` em `internal/logger/logger.go`

## 2. Campo `component` — Infraestrutura (db, nats, s3, server)

- [x] 2.1 Adicionar `.Str("component", "db")` em todos os logs de `internal/database/postgres.go`
- [x] 2.2 Adicionar `.Str("component", "nats")` em todos os logs de `internal/broker/nats.go`
- [x] 2.3 Adicionar `.Str("component", "s3")` em todos os logs de `internal/storage/minio.go`
- [x] 2.4 Adicionar `.Str("component", "server")` em todos os logs de `internal/server/server.go` e `internal/server/router.go`

## 3. Campo `component` — Middleware HTTP

- [x] 3.1 Adicionar `.Str("component", "http")` no middleware logger em `internal/middleware/logger.go`
- [x] 3.2 Adicionar `.Str("component", "http")`, `method`, `path` e `ip` no middleware recovery em `internal/middleware/recovery.go`
- [x] 3.3 Adicionar `.Str("component", "http")` no log de auth warning em `internal/middleware/auth.go`

## 4. Campo `component` — WebSocket

- [x] 4.1 Adicionar `.Str("component", "ws")` em todos os logs de `internal/websocket/hub.go`

## 5. Campo `component` e Unificação — Webhook Dispatcher

- [x] 5.1 Adicionar `.Str("component", "webhook")` em todos os logs de `internal/webhook/dispatcher.go`
- [x] 5.2 Unificar os logs "Dispatching webhook" e "Active webhooks found" em um único log com campos `session`, `event`, `webhooks` (count) e `globalURL`

## 6. Campo `component` e Padronização — Handlers e Services

- [x] 6.1 Corrigir `.Str("sessionID", ...)` para `.Str("session", ...)` em `internal/handler/message.go`
- [x] 6.2 Adicionar `.Str("component", "handler")` nos logs de `internal/handler/message.go` e `internal/handler/websocket.go`
- [x] 6.3 Adicionar `.Str("component", "service")` nos logs de `internal/service/session.go`, `internal/service/message.go`, `internal/service/media.go`, `internal/service/group.go` e `internal/service/history.go`

## 7. Campo `component` — WhatsApp Engine

- [x] 7.1 Adicionar `.Str("component", "wa")` em todos os logs de `internal/wa/events.go`
- [x] 7.2 Adicionar `.Str("component", "wa")` em todos os logs de `internal/wa/connect.go` e `internal/wa/qr.go`

## 8. Enriquecimento de Eventos WA

- [x] 8.1 Adicionar `msgType` e `mediaType` no log "Message received" usando `extractMessageContent` em `internal/wa/events.go`
- [x] 8.2 Adicionar `mid` (message ID) no log de `MediaRetry`
- [x] 8.3 Adicionar `from` e `callID` nos logs de eventos de chamada (`CallOffer`, `CallAccept`, `CallTerminate`, `CallOfferNotice`, `CallPreAccept`, `CallReject`, `CallTransport`)
- [x] 8.4 Adicionar `jid` nos logs de eventos de newsletter (`NewsletterJoin`, `NewsletterLeave`, `NewsletterMuteChange`, `NewsletterLiveUpdate`)
- [x] 8.5 Adicionar `action` (muted/unmuted, pinned/unpinned, archived/unarchived) nos logs de `Mute`, `Pin` e `Archive`
- [x] 8.6 Adicionar `chat` e `count` no log de `Receipt` recebido
- [x] 8.7 Corrigir nível de `AppStateSyncError` de DEBUG para WARN

## 9. Campo `component` — Chatwoot (remover prefixo [CW])

- [x] 9.1 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/consumer.go`
- [x] 9.2 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/service.go`
- [x] 9.3 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/handler.go`
- [x] 9.4 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/inbound_message.go`
- [x] 9.5 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/inbound_events.go`
- [x] 9.6 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/outbound.go`
- [x] 9.7 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/conversation.go`
- [x] 9.8 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/client.go`
- [x] 9.9 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/cache.go`
- [x] 9.10 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/bot_commands.go`
- [x] 9.11 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/jid.go`
- [x] 9.12 Substituir `[CW]` por `.Str("component", "chatwoot")` em `internal/integrations/chatwoot/labels.go`

## 10. Verificação Final

- [x] 10.1 Executar `go build ./...` para garantir compilação sem erros
- [x] 10.2 Executar `go test -race ./...` para garantir que todos os testes passam
- [x] 10.3 Verificar que não existe mais nenhum `[CW]` nos arquivos de log via grep
- [x] 10.4 Verificar que não existe mais nenhum `.Str("sessionID"` nos arquivos de log via grep
