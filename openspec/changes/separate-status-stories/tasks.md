## 1. Migration e Modelo

- [x] 1.1 Criar `migrations/008_statuses.down.sql` com DROP TABLE e recriacao dos dados em `wz_messages`
- [x] 1.2 Revisar `migrations/008_statuses.up.sql` (ja criado parcialmente) — garantir integridade da migracao de dados
- [x] 1.3 Criar `internal/model/status.go` com struct `Status` e tags JSON/DB

## 2. Repositorio

- [x] 2.1 Criar `internal/repo/status.go` com interface `StatusRepo` (Save, FindBySession, FindBySender, DeleteBySender)
- [x] 2.2 Implementar `StatusRepository` com SQL usando pgx, seguindo padrao de `MessageRepository`
- [x] 2.3 Adicionar `AND chat_jid NOT LIKE 'status@%'` em `MessageRepository.FindBySession` em `internal/repo/message.go`

## 3. DTOs

- [x] 3.1 Criar `internal/dto/status.go` com `SendStatusTextReq` (text obrigatorio), `SendStatusMediaReq` (base64/url, caption, mimeType)

## 4. Service

- [x] 4.1 Criar `internal/service/status.go` com `StatusService` (SendText, SendMedia, ListReceived, ListBySender)
- [x] 4.2 Refatorar `internal/service/message_status.go` para usar `StatusRepo` em vez de `persistSent` para `wz_messages`
- [x] 4.3 Adicionar callback `ShouldIgnoreStatus(sessionID string) bool` no `wa.Manager` e injetar logica de leitura do setting no `server.go`

## 5. Event Routing (whatsmeow)

- [x] 5.1 Adicionar callback `OnStatusReceived` em `internal/wa/manager.go` com tipo `StatusReceivedFunc`
- [x] 5.2 Modificar `internal/wa/events.go` no handler de `events.Message`: se `Chat.Server == BroadcastServer`, rotear para `OnStatusReceived` e respeitar `ShouldIgnoreStatus`
- [x] 5.3 Modificar `internal/service/history.go` para pular conversas com `status@` no `PersistHistorySync`

## 6. Handler e Rotas

- [x] 6.1 Criar `internal/handler/status.go` com `StatusHandler` (SendText, SendImage, SendVideo, ListStatus, ListContactStatus)
- [x] 6.2 Registrar `StatusRepo`, `StatusService` e `StatusHandler` em `internal/server/router.go`
- [x] 6.3 Registrar rotas: `POST /status/text`, `POST /status/image`, `POST /status/video`, `GET /status`, `GET /status/:senderJid`
- [x] 6.4 Conectar `OnStatusReceived` do Manager ao `StatusService` no `server.go`

## 7. Integracao Chatwoot

- [x] 7.1 Verificar filtro generico para `status@` em `internal/integrations/chatwoot/inbound_message.go` e `jid.go`

## 8. Frontend: Composable

- [x] 8.1 Criar `web/app/composables/useStatus.ts` com fetchStatuses, fetchContactStatuses, sendStatusText, sendStatusImage, sendStatusVideo

## 9. Frontend: Componentes

- [x] 9.1 Criar `web/app/components/sessions/StatusStoryCard.vue` — card de contato com indicador de stories
- [x] 9.2 Criar `web/app/components/sessions/StatusViewModal.vue` — visualizacao fullscreen com navegacao
- [x] 9.3 Criar `web/app/components/sessions/StatusSendModal.vue` — modal dedicado para envio de status (texto/imagem/video)

## 10. Frontend: Pagina e Navegacao

- [x] 10.1 Criar `web/app/pages/sessions/[id]/status.vue` — pagina dedicada de status com lista de contatos
- [x] 10.2 Adicionar link "Status" no `sessionNavLinks` em `web/app/layouts/default.vue` com icon `i-lucide-circle-dot`

## 11. Frontend: Limpeza

- [x] 11.1 Remover opcoes `status-text`, `status-image`, `status-video` de `MessageType`, `MESSAGE_TYPE_OPTIONS` e `TYPE_TO_ENDPOINT` em `web/app/composables/useMessageSender.ts`
- [x] 11.2 Remover status forms e logica relacionada de `web/app/components/sessions/SendMessageModal.vue`
- [x] 11.3 Remover forms de status de `web/app/components/sessions/message-forms/` (StatusTextForm, StatusImageForm, StatusVideoForm)

## 12. Testes e Lint

- [x] 12.1 Rodar `go vet ./...` e `golangci-lint run ./...` para verificar backend
- [ ] 12.2 Rodar `cd web && nuxt typecheck` para verificar frontend
- [ ] 12.3 Rodar `go test -v -race ./internal/repo/...` para testes de repositorio