# Tasks: wzap Competitive Gap Features

**Input**: Design documents from `/specs/001-competitive-gap-analysis/`
**Prerequisites**: plan.md (required), spec.md (required)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Already Implemented (no tasks needed)

| Feature | Status | Details |
|---------|--------|---------|
| FR-005 Revogar Link de Convite | DONE | `GetInviteLink` already accepts `?reset=true` query param — `internal/handler/group.go:126`, `internal/service/group.go:258` |
| FR-003 Menções (Reply) | PARTIAL | `buildContextInfo` already wires `MentionedJID` via `ReplyContext` — `internal/service/message.go:490`. Missing: standalone mentions without reply (US3 below). |

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Dependency additions and shared infrastructure

- [X] T001 Adicionar `github.com/prometheus/client_golang/prometheus` e `promhttp` ao go.mod e rodar `go mod tidy`
- [X] T002 [P] Adicionar `ffmpeg` ao Dockerfile em `Dockerfile` — incluir `apk add --no-cache ffmpeg` no runtime stage

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before user stories

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Criar migration `internal/database/migrations/003_webhook_event_urls.up.sql` — `ALTER TABLE wz_webhooks ADD COLUMN event_urls JSONB NOT NULL DEFAULT '{}'`
- [X] T004 [P] Criar `internal/database/migrations/003_webhook_event_urls.down.sql` — `ALTER TABLE wz_webhooks DROP COLUMN event_urls`
- [X] T005 Criar pacote `internal/metrics/metrics.go` — definir counters, gauges e histograms Prometheus: `SessionsTotal`, `SessionsConnected`, `MessagesSent`, `MessagesReceived`, `WebhooksDelivered`, `WebhooksFailed`, `WebhooksDuration`

**Checkpoint**: Foundation ready — user story implementation can now begin in parallel

---

## Phase 3: User Story 1 — Status/Stories (Priority: P1)

**Goal**: Enviar mensagens de status (texto, imagem, video) via WhatsApp Stories

**Independent Test**: `POST /sessions/:sessionId/messages/status/text` (ou /image, /video) — retorna ID da mensagem de status

### Implementation for User Story 1

- [X] T006 [US1] Adicionar DTOs `SendStatusTextReq` e `SendStatusMediaReq` em `internal/dto/message.go`
- [X] T007 [US1] Adicionar `SendStatusText()` e `SendStatusMedia()` em `internal/service/message.go` — enviar para `types.StatusBroadcastJID` usando protobuf `ExtendedTextMessage` / `ImageMessage` / `VideoMessage`
- [X] T008 [US1] Adicionar handlers `SendStatusText()`, `SendStatusImage()`, `SendStatusVideo()` em `internal/handler/message.go` — seguir padrão existente (getSessionID + parseAndValidate + service call)
- [X] T009 [US1] Registrar 3 rotas em `internal/server/router.go`: `sess.Post("/messages/status/text", ...)`, `sess.Post("/messages/status/image", ...)`, `sess.Post("/messages/status/video", ...)`

**Checkpoint**: Status/Stories funcional — pode ser testado independentemente com sessao conectada

---

## Phase 4: User Story 2 — Encaminhar Mensagem (Priority: P1)

**Goal**: Encaminhar uma mensagem existente para outro chat ou grupo com indicador de encaminhada

**Independent Test**: `POST /sessions/:sessionId/messages/forward` com messageId + fromJid + phone — mensagem aparece no destino

### Implementation for User Story 2

- [X] T010 [US2] Adicionar DTO `ForwardMessageReq` em `internal/dto/message.go` — campos: `MessageID`, `FromJID`, `Phone` (todos required)
- [X] T011 [US2] Adicionar `ForwardMessage()` em `internal/service/message.go` — montar `ExtendedTextMessage` com `ContextInfo{IsForwarded: true, ForwardingScore: 1, StanzaId, RemoteJID}`
- [X] T012 [US2] Adicionar handler `ForwardMessage()` em `internal/handler/message.go`
- [X] T013 [US2] Registrar rota em `internal/server/router.go`: `sess.Post("/messages/forward", messageHandler.ForwardMessage)`

**Checkpoint**: Forward funcional — pode ser testado independentemente

---

## Phase 5: User Story 3 — Menções @user Standalone (Priority: P2)

**Goal**: Suportar menções (@user) em mensagens de texto sem exigir ReplyTo

**Independent Test**: `POST /sessions/:sessionId/messages/text` com campo `mentionedJids` no body — mencões aparecem na mensagem

**Nota**: `buildContextInfo` ja suporta `MentionedJID` via `ReplyContext`. Esta story adiciona suporte standalone (sem reply).

### Implementation for User Story 3

- [X] T014 [US3] Adicionar campo `MentionedJIDs []string` em `SendTextReq` e demais DTOs de mensagem em `internal/dto/message.go` (SendMediaReq, SendButtonReq, SendListReq)
- [X] T015 [US3] Modificar `SendText()` em `internal/service/message.go` — se `req.MentionedJIDs` esta presente e nao ha ReplyTo, criar `ContextInfo` com `MentionedJID` direktamente
- [X] T016 [US3] Refatorar `buildContextInfo()` em `internal/service/message.go` para aceitar `mentionedJIDs []string` adicional ou criar helper `buildMentionContextInfo()`
- [X] T017 [US3] Aplicar mesmo padrao em `SendImage()`, `SendVideo()`, `SendButton()`, `SendList()` em `internal/service/message.go`

**Checkpoint**: Menções standalone funcionais — `mentionedJids` funciona com e sem ReplyTo

---

## Phase 6: User Story 4 — Atualizar Nome de Perfil (Priority: P2)

**Goal**: Atualizar push name do perfil via API

**Independent Test**: `POST /sessions/:sessionId/profile/name` com body `{"name": "Novo Nome"}` — nome atualizado

### Implementation for User Story 4

- [X] T018 [US4] Adicionar DTO `UpdateProfileNameReq` em `internal/dto/session.go` — campo: `Name string validate:"required"`
- [X] T019 [US4] Adicionar `UpdateProfileName()` em `internal/service/contact.go` — usar `appstate.BuildSettingPushName(name)` + `client.SendAppState(ctx, patch)`
- [X] T020 [US4] Adicionar handler `UpdateProfileName()` em `internal/handler/contact.go`
- [X] T021 [US4] Registrar rota em `internal/server/router.go`: `sess.Post("/profile/name", contactHandler.UpdateProfileName)`

**Checkpoint**: Profile name atualizavel via API

---

## Phase 7: User Story 5 — Revogar Link de Convite (Priority: P2)

**Goal**: Revogar link de convite de grupo (gerar novo link)

**Independent Test**: `POST /sessions/:sessionId/groups/invite-link` com `?reset=true` — retorna novo link

**NOTA**: Ja implementado! Handler em `internal/handler/group.go:116` aceita `?reset=true`. Service em `internal/service/group.go:258` passa `reset` para whatsmeow. Sem tarefas necessarias.

**Checkpoint**: Ja funcional — nenhuma acao necessaria

---

## Phase 8: User Story 6 — Webhook por Tipo de Evento (Priority: P2)

**Goal**: Configurar URLs de webhook diferentes para cada tipo de evento

**Independent Test**: Criar webhook com `eventUrls: {"Message": "https://a.com", "GroupInfo": "https://b.com"}` — cada evento vai para sua URL

### Implementation for User Story 6

- [X] T022 [US6] Adicionar campo `EventURLs map[string]string` em `model.Webhook` em `internal/model/webhook.go`
- [X] T023 [US6] Adicionar campo `EventURLs map[string]string` em `dto.CreateWebhookReq` e `dto.UpdateWebhookReq` em `internal/dto/webhook.go`
- [X] T024 [US6] Atualizar `FindBySessionID()`, `FindActiveBySessionAndEvent()`, `FindByID()` em `internal/repo/webhook.go` — incluir `event_urls` no SELECT e Scan como JSONB
- [X] T025 [US6] Atualizar `Create()` e `Update()` em `internal/repo/webhook.go` — incluir `event_urls` no INSERT/UPDATE
- [X] T027 [US6] Atualizar `WebhookResp` em `internal/dto/session.go` para incluir `EventURLs`
- [X] T028 [US6] Atualizar handlers de webhook em `internal/handler/webhook.go` para mapear `EventURLs` entre DTO e model

**Checkpoint**: Webhooks com routing por evento funcional — cada tipo de evento pode ter URL dedicada

---

## Phase 9: User Story 7 — Conversao de Audio ffmpeg (Priority: P3)

**Goal**: Converter automaticamente audios (MP3, WAV, M4A) para OGG Opus antes de enviar como mensagem de voz

**Independent Test**: Enviar MP3 via `/messages/audio` — audio reproduzivel como mensagem de voz

### Implementation for User Story 7

- [X] T029 [US7] Adicionar funcao `convertToOGG(input []byte) ([]byte, error)` em `internal/service/media.go` — usar `os/exec` com `ffmpeg -i pipe:0 -c:a libopus -f ogg pipe:1`
- [X] T030 [US7] Modificar `SendAudio()` em `internal/service/message.go` — detectar formato do audio (por mime type ou extensao); se nao for OGG Opus, chamar `convertToOGG()` antes do upload
- [X] T031 [US7] Adicionar fallback gracoso em `convertToOGG()` — se ffmpeg nao disponivel, logar warning e enviar audio sem conversao

**Checkpoint**: Audio conversion funcional — MP3/WAV/M4A convertidos automaticamente para OGG Opus

---

## Phase 10: User Story 8 — Prometheus Metrics (Priority: P3)

**Goal**: Endpoint `/metrics` exportando metricas no formato Prometheus para monitoramento

**Independent Test**: `GET /metrics` retorna metricas no formato Prometheus text

### Implementation for User Story 8

- [X] T032 [US8] Implementar definicoes de metricas em `internal/metrics/metrics.go` — registrar no prometheus.DefaultRegisterer: gauges (SessionsTotal, SessionsConnected), counters (MessagesSent, MessagesReceived, WebhooksDelivered, WebhooksFailed), histogram (WebhooksDuration)
- [X] T033 [US8] Criar `internal/handler/metrics.go` — handler que expoe `promhttp.Handler()` como endpoint Fiber
- [X] T034 [US8] Registrar rota publica em `internal/server/router.go`: `s.App.Get("/metrics", metricsHandler.Serve)` — sem auth, antes do grupo autenticado
- [X] T035 [US8] Instrumentar `SessionService` em `internal/service/session.go` — incrementar `metrics.SessionsTotal` em Create, `metrics.SessionsConnected` em Connect/Disconnect
- [X] T036 [US8] Instrumentar `MessageService` em `internal/service/message.go` — incrementar `metrics.MessagesSent` em cada metodo Send*
- [X] T037 [US8] Instrumentar dispatcher em `internal/webhook/dispatcher.go` — incrementar `metrics.WebhooksDelivered`/`WebhooksFailed` e observar duracao no histogram

**Checkpoint**: Endpoint /metrics funcional com pelo menos 7 metricas relevantes

---

## Phase 11: User Story 9 — Marcar Chat como Nao Lido (Priority: P3)

**Goal**: Marcar chat como nao lido via API

**Independent Test**: `POST /sessions/:sessionId/chat/unread` com `{"jid": "..."}` — chat marcado como nao lido

**NOTA**: whatsmeow nao tem API direta para isso. Investigar app state ou workaround na implementacao.

### Implementation for User Story 9

- [X] T038 [US9] Adicionar DTO `ChatMarkUnreadReq` em `internal/dto/chat.go` — campo: `JID string validate:"required"`
- [X] T039 [US9] Investigar viabilidade via whatsmeow — verificar se `client.SendAppState()` com `appstate.BuildSetting...` suporta marcar como nao lido, ou se existe workaround via `MarkRead` inverso ou ChatSettings
- [X] T040 [US9] Se viavel: adicionar `MarkUnread()` em `internal/service/chat.go` e handler em `internal/handler/chat.go`; registrar rota `sess.Post("/chat/unread", chatHandler.MarkUnread)` em `internal/server/router.go`
- [X] T041 [US9] Se inviavel: adicionar stub que retorna erro 501 Not Implemented com mensagem explicativa em `internal/handler/chat.go`; registrar rota mesmo assim para documentar a intenção

**Checkpoint**: Endpoint /chat/unread disponivel (funcional ou stub com 501)

---

## Phase 12: FR-010 — Restart Instancia (Priority: P3)

**Goal**: Reiniciar instancia sem perder estado (disconnect + connect)

**Independent Test**: `POST /sessions/:sessionId/restart` — sessao reconecta e retorna status atualizado

### Implementation

- [X] T042 Adicionar `Restart()` em `internal/service/session.go` — chamar `s.engine.Disconnect(id)`, aguardar 1s, `s.engine.Connect(id)`, retornar `s.Get(ctx, id)`
- [X] T043 Adicionar handler `Restart()` em `internal/handler/session.go`
- [X] T044 Registrar rota em `internal/server/router.go`: `sess.Post("/restart", sessionHandler.Restart)`

**Checkpoint**: Restart funcional — sessao reconecta sem perder estado

---

## Phase 13: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T045 Rodar `make docs` para regenerar Swagger docs apos todas as rotas novas
- [X] T046 Rodar `go mod tidy` para garantir dependencias corretas
- [X] T047 [P] Rodar `golangci-lint run ./...` e corrigir warnings
- [X] T048 [P] Rodar `go test -v -race ./...` e garantir que testes existentes passam

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — pode comecar imediatamente
- **Foundational (Phase 2)**: Depende de Phase 1 (Prometheus dep) — BLOCKS US8, US6
- **User Stories (Phase 3–12)**: Maioria pode comecar sem Phase 2, exceto:
  - US6 (Webhook por evento): depende de T003 (migration)
  - US8 (Prometheus): depende de T005 (metrics package)
- **Polish (Phase 13)**: Depende de todas as stories desejadas completas

### User Story Dependencies

- **US1 (Status)**: Independente — pode comecar apos Phase 1
- **US2 (Forward)**: Independente — pode comecar apos Phase 1
- **US3 (Menções)**: Independente — modifica DTOs e service de mensagem (pode conflitar com US1 se em paralelo no mesmo arquivo)
- **US4 (Profile Name)**: Independente — modifica contact handler/service
- **US5 (Revogar Link)**: Ja implementado — sem acao
- **US6 (Webhook por evento)**: Depende de T003 (migration)
- **US7 (Audio conversion)**: Depende de T002 (ffmpeg no Dockerfile)
- **US8 (Prometheus)**: Depende de T005 (metrics package)
- **US9 (Unread)**: Independente — mas pode precisar de pesquisa whatsmeow
- **FR-010 (Restart)**: Independente

### Parallel Opportunities

- US1 + US4 podem ser feitos em paralelo (arquivos diferentes: message vs contact)
- US2 + US4 podem ser feitos em paralelo
- US3 + US7 podem ser feitos em paralelo (menções toca message.go DTO, audio toca media.go)
- FR-010 pode ser feito em paralelo com qualquer story

---

## Parallel Example: Sprint 1 (P1 Stories)

```bash
# Launch in parallel (different handlers/services):
Task T006-T009: "US1 — Status/Stories (handler/message.go + service/message.go)"
Task T010-T013: "US2 — Forward Message (handler/message.go + service/message.go)"
Task T042-T044: "FR-010 — Restart Instance (handler/session.go + service/session.go)"

# Note: US1 and US2 both touch message.go — sequential if same developer
```

## Parallel Example: Sprint 2 (P2 Stories)

```bash
# Launch in parallel (different files):
Task T014-T017: "US3 — Menções (dto/message.go + service/message.go)"
Task T018-T021: "US4 — Profile Name (handler/contact.go + service/contact.go)"
Task T022-T028: "US6 — Webhook por evento (model/webhook.go + repo/webhook.go + webhook/dispatcher.go)"
```

---

## Implementation Strategy

### MVP First (P1 Stories Only)

1. Complete Phase 1: Setup
2. Complete Phase 3: US1 — Status/Stories
3. Complete Phase 4: US2 — Forward Message
4. **STOP and VALIDATE**: Testar US1 e US2 independentemente
5. Deploy/demo se pronto

### Incremental Delivery

1. Setup (Phase 1) → Dependencias prontas
2. US1 + US2 (P1) → Testar → Deploy (MVP!)
3. US3 + US4 + US5 (P2, baixa complexidade) → Testar → Deploy
4. US6 (P2, alta complexidade) → Testar → Deploy
5. US7 + US8 + US9 + FR-010 (P3) → Testar → Deploy
6. Polish (Phase 13) → Lint + testes + docs

### Suggested Execution Order (single developer)

| Ordem | Tasks | Story | Arquivos principais |
|-------|-------|-------|-------------------|
| 1 | T001, T002 | Setup | go.mod, Dockerfile |
| 2 | T003, T004, T005 | Foundational | migrations, metrics |
| 3 | T006–T009 | US1 Status | dto/message, service/message, handler/message, router |
| 4 | T010–T013 | US2 Forward | dto/message, service/message, handler/message, router |
| 5 | T014–T017 | US3 Menções | dto/message, service/message |
| 6 | T018–T021 | US4 Profile | dto/session, service/contact, handler/contact, router |
| 7 | T042–T044 | FR-010 Restart | service/session, handler/session, router |
| 8 | T022–T028 | US6 Webhook | model, dto, repo, dispatcher, handler |
| 9 | T029–T031 | US7 Audio | service/media, service/message |
| 10 | T032–T037 | US8 Prometheus | metrics, handler/metrics, router, service/*, dispatcher |
| 11 | T038–T041 | US9 Unread | dto/chat, service/chat, handler/chat, router |
| 12 | T045–T048 | Polish | docs, go.mod, lint, test |

---

## Notes

- **FR-005 (Revogar Link)** ja implementado — sem tarefas
- **FR-003 (Menções)** parcialmente implementado via ReplyContext — US3 completa o suporte standalone
- [P] tasks = arquivos diferentes, sem dependencias entre si
- [Story] label mapeia task para user story especifica para rastreabilidade
- Cada user story deve ser independentemente completavel e testavel
- Commit apos cada task ou grupo logico
- Parar em qualquer checkpoint para validar story independentemente
- Sem breaking changes na API existente (constitution principle)
