## 1. Corrigir duplo AddEventHandler em wa/connect.go

- [ ] 1.1 Investigar `Connect()` e confirmar que linha 165-167 é o handler redundante (adicionar `default: m.handleEvent(sessionID, evt)` ao switch inline da linha 169 antes de remover)
- [ ] 1.2 Adicionar `default: m.handleEvent(sessionID, evt)` ao switch inline de `Connect()` (linhas 169-204) em `internal/wa/connect.go`
- [ ] 1.3 Remover o `AddEventHandler` redundante das linhas 165-167 de `internal/wa/connect.go`
- [ ] 1.4 Verificar que eventos de lifecycle (`Connected`, `PairSuccess`, `Disconnected`, `LoggedOut`) continuam sendo tratados corretamente e não caem no `default`

## 2. Enriquecer extractMessageContent com tipos ausentes

- [ ] 2.1 Adicionar cases para `ViewOnceMessage`, `TemplateMessage`, `InteractiveMessage`, `LiveLocationMessage`, `ProductMessage` em `extractMessageContent` de `internal/wa/events.go`
- [ ] 2.2 Alterar o `default` de `extractMessageContent` de `return "unknown", "", ""` para `return "unsupported", "", ""`

## 3. Corrigir log handleMessage no Chatwoot

- [ ] 3.1 Adicionar campo `Str("session", cfg.SessionID)` ao log `handleMessage` em `internal/integrations/chatwoot/inbound_message.go`

## 4. Padronizar component=async em async/pool.go

- [ ] 4.1 Adicionar `.Str("component", "async")` em todos os logs de `internal/async/pool.go`

## 5. Verificação

- [ ] 5.1 Executar `go build ./...` sem erros
- [ ] 5.2 Executar `go test -race ./internal/wa/... ./internal/async/... ./internal/integrations/chatwoot/...`
- [ ] 5.3 Verificar nos logs do Docker que "Message received" aparece exatamente uma vez por mensagem recebida
