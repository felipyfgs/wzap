# Implementation Plan: wzap Competitive Gap Features

**Branch**: `001-competitive-gap-analysis` | **Date**: 2026-04-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-competitive-gap-analysis/spec.md`

## Summary

Implementar 10 features gap identificadas na analise competitiva wzap vs Evolution API/CodeChat/WPPConnect. Features priorizadas por impacto competitivo (P1: Status e Forward, P2: Mentions, Profile, Revoke, Webhook por evento, P3: Audio, Prometheus, Unread, Restart).

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: Fiber, whatsmeow, pgx, NATS JetStream, MinIO
**Storage**: PostgreSQL (pgx)
**Testing**: testing.T (padrao Go), fiber.Test
**Target Platform**: Linux server (Docker)
**Project Type**: Web service (REST API)
**Performance Goals**: Mesmo nivel atual (120 req/min rate limit)
**Constraints**: Sem breaking changes na API existente
**Scale/Scope**: 10 features, ~30 arquivos modificados

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Arquitetura em Camadas | ✅ | Toda feature: handler -> service -> (repo se necessario) |
| II. Testes Obrigatorios | ✅ | Testes com testing.T para cada handler/service novo |
| III. Convencoes Go | ✅ | Imports 3 grupos, PascalCase, tags json/validate |
| IV. Context e Erros | ✅ | ctx primeiro, erros wrapados |
| V. Simplicidade | ✅ | Sem abstracao nova, seguir padroes existentes |

No violations.

## Feasibility Matrix (whatsmeow)

| # | Feature | Viavel? | API whatsmeow |
|---|---------|---------|---------------|
| FR-001 | Status/Stories | SIM | `client.SendMessage(ctx, types.StatusBroadcastJID, msg)` |
| FR-002 | Encaminhar mensagem | SIM | `ContextInfo.IsForwarded` + `ForwardingScore` no protobuf |
| FR-003 | Mencoes @user | SIM | `ContextInfo.MentionedJID []string` no protobuf |
| FR-004 | Atualizar nome perfil | SIM | `appstate.BuildSettingPushName(name)` + `client.SendAppState()` |
| FR-005 | Revogar link convite | SIM | `client.GetGroupInviteLink(ctx, jid, true)` (reset=true) |
| FR-006 | Webhook por evento | SIM | wzap interno — estender modelo Webhook |
| FR-007 | Conversao audio ffmpeg | SIM | `os/exec` do Go + ffmpeg no container |
| FR-008 | Prometheus metrics | SIM | wzap interno — novo endpoint `/metrics` |
| FR-009 | Marcar chat nao lido | PARCIAL | Investigar app state ou workaround |
| FR-010 | Restart instancia | SIM | wzap interno — disconnect + connect |

## Project Structure

### Documentation (this feature)

```text
specs/001-competitive-gap-analysis/
├── plan.md              # This file
├── spec.md              # Feature specification
├── checklists/
│   └── requirements.md  # Quality checklist
└── tasks.md             # Tasks (created by /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── dto/
│   ├── message.go          # +3 DTOs (status text/image/video)
│   ├── session.go          # +1 DTO (update profile name)
│   ├── group.go            # modificado (invite link reset)
│   ├── chat.go             # +1 DTO (mark unread)
│   └── webhook.go          # modificado (eventUrls)
├── handler/
│   ├── message.go          # +4 handlers (status text/image/video, forward)
│   ├── session.go          # +1 handler (restart)
│   ├── contact.go          # +1 handler (update profile name)
│   ├── group.go            # modificado (revoke invite)
│   ├── chat.go             # +1 handler (mark unread)
│   └── metrics.go          # NOVO (prometheus handler)
├── service/
│   ├── message.go          # +4 metodos (status, forward) + mentions + audio convert
│   ├── session.go          # +1 metodo (restart)
│   ├── contact.go          # +1 metodo (update profile name)
│   ├── group.go            # modificado (revoke invite)
│   ├── chat.go             # +1 metodo (mark unread)
│   └── media.go            # modificado (audio conversion)
├── metrics/
│   └── metrics.go          # NOVO (prometheus definitions)
├── model/
│   └── webhook.go          # modificado (+EventURLs)
├── repo/
│   └── webhook.go          # modificado (scan EventURLs)
├── webhook/
│   └── dispatcher.go       # modificado (event-based URL routing)
├── database/
│   └── migrations/
│       └── 003_webhook_event_urls.up.sql  # NOVO
├── server/
│   └── router.go           # +8 rotas
├── wa/
│   └── connect.go          # sem mudancas
└── testutil/
    └── fiber.go            # sem mudancas

Dockerfile                  # modificado (adicionar ffmpeg)
go.mod                      # modificado (+prometheus client)
```

## Implementation Details — Por Feature

---

### FR-003: Mencoes @user (P2 — Baixa complexidade)

**Mudanca:** Estender DTOs existentes com campo `mentionedJids`.

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/dto/message.go` | Adicionar `MentionedJIDs []string` em `SendTextReq` e outros DTOs de mensagem |
| `internal/service/message.go` | Incluir `MentionedJID` no `ContextInfo` ao montar mensagens |

**DTO:**
```go
type SendTextReq struct {
    Phone         string   `json:"phone" validate:"required"`
    Text          string   `json:"text" validate:"required"`
    ReplyTo       string   `json:"replyTo,omitempty"`
    MentionedJIDs []string `json:"mentionedJids,omitempty"`
}
```

**Service:**
```go
contextInfo := &waE2E.ContextInfo{}
if len(req.MentionedJIDs) > 0 {
    contextInfo.MentionedJID = req.MentionedJIDs
}
```

---

### FR-005: Revogar Link de Convite (P2 — Baixa complexidade)

**Mudanca:** Estender endpoint existente.

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/handler/group.go` | Modificar `GetInviteLink` para aceitar query param `?reset=true` |
| `internal/service/group.go` | Passar flag reset para whatsmeow |

**Handler:**
```go
reset := c.Query("reset") == "true"
link, err := h.groupSvc.GetInviteLink(c.Context(), sessionID, groupJID, reset)
```

**Service:**
```go
func (s *GroupService) GetInviteLink(ctx context.Context, sessionID, groupJID string, reset bool) (string, error) {
    client, _ := s.engine.GetClient(sessionID)
    return client.GetGroupInviteLink(ctx, parsedJID, reset)
}
```

---

### FR-004: Atualizar Nome de Perfil (P2 — Baixa complexidade)

**Endpoint:** `POST /sessions/:sessionId/profile/name`

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/dto/session.go` | Adicionar `UpdateProfileNameReq` |
| `internal/service/contact.go` | Adicionar `UpdateProfileName()` |
| `internal/handler/contact.go` | Adicionar handler |
| `internal/server/router.go` | Registrar rota |

**DTO:**
```go
type UpdateProfileNameReq struct {
    Name string `json:"name" validate:"required"`
}
```

**Service:**
```go
func (s *ContactService) UpdateProfileName(ctx context.Context, sessionID, name string) error {
    client, _ := s.engine.GetClient(sessionID)
    patch := appstate.BuildSettingPushName(name)
    return client.SendAppState(ctx, patch)
}
```

---

### FR-010: Restart Instancia (P3 — Baixa complexidade)

**Endpoint:** `POST /sessions/:sessionId/restart`

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/handler/session.go` | Adicionar `Restart()` |
| `internal/service/session.go` | Adicionar `Restart()` |
| `internal/server/router.go` | Registrar rota |

**Service:**
```go
func (s *SessionService) Restart(ctx context.Context, id string) (*dto.SessionResp, error) {
    s.engine.Disconnect(id)
    time.Sleep(1 * time.Second)
    if err := s.engine.Connect(id); err != nil {
        return nil, fmt.Errorf("failed to restart session: %w", err)
    }
    return s.Get(ctx, id)
}
```

---

### FR-002: Encaminhar Mensagem (P1 — Media complexidade)

**Endpoint:** `POST /sessions/:sessionId/messages/forward`

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/dto/message.go` | Adicionar `ForwardMessageReq` |
| `internal/service/message.go` | Adicionar `ForwardMessage()` |
| `internal/handler/message.go` | Adicionar `ForwardMessage()` |
| `internal/server/router.go` | Registrar rota |

**DTO:**
```go
type ForwardMessageReq struct {
    MessageID string `json:"messageId" validate:"required"`
    FromJID   string `json:"fromJid" validate:"required"`
    Phone     string `json:"phone" validate:"required"`
}
```

**Service:**
```go
msg := &waE2E.Message{
    ExtendedTextMessage: &waE2E.ExtendedTextMessage{
        ContextInfo: &waE2E.ContextInfo{
            IsForwarded:     proto.Bool(true),
            ForwardingScore: proto.Uint32(1),
            StanzaId:        proto.String(req.MessageID),
            RemoteJID:       proto.String(req.FromJID),
        },
    },
}
resp, err := client.SendMessage(ctx, destJID, msg, whatsmeow.SendRequestExtra{ID: msgID})
```

---

### FR-001: Status/Stories (P1 — Media complexidade)

**Endpoints:**
```
POST /sessions/:sessionId/messages/status/text
POST /sessions/:sessionId/messages/status/image
POST /sessions/:sessionId/messages/status/video
```

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/dto/message.go` | Adicionar `SendStatusTextReq`, `SendStatusMediaReq` |
| `internal/service/message.go` | Adicionar `SendStatusText()`, `SendStatusMedia()` |
| `internal/handler/message.go` | Adicionar 3 handlers |
| `internal/server/router.go` | Registrar 3 rotas |

**Service Pattern:**
```go
msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
    Text: proto.String(req.Text),
}}
resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, msg, whatsmeow.SendRequestExtra{ID: msgID})
```

---

### FR-009: Marcar Chat como Nao Lido (P3 — Media complexidade)

**Endpoint:** `POST /sessions/:sessionId/chat/unread`

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/dto/chat.go` | Adicionar DTO |
| `internal/handler/chat.go` | Adicionar handler |
| `internal/service/chat.go` | Adicionar metodo |
| `internal/server/router.go` | Registrar rota |

**NOTA:** whatsmeow nao tem API direta para isso. Vamos investigar na implementacao se e viavel via app state ou workaround.

---

### FR-006: Webhook por Tipo de Evento (P2 — Alta complexidade)

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/model/webhook.go` | Adicionar `EventURLs map[string]string` |
| `internal/dto/webhook.go` | Adicionar campo em Create/Update |
| `internal/repo/webhook.go` | Scan do novo campo JSONB |
| `internal/webhook/dispatcher.go` | Routing por evento |
| `internal/database/migrations/003_webhook_event_urls.up.sql` | NOVO |

**Migration:**
```sql
ALTER TABLE wz_webhooks ADD COLUMN event_urls JSONB NOT NULL DEFAULT '{}';
```

**Dispatcher Logic:**
```go
url := wh.URL
if specific, ok := wh.EventURLs[string(eventType)]; ok {
    url = specific
}
go d.deliverHTTPWithRetry(url, wh.Secret, payload)
```

---

### FR-007: Conversao de Audio ffmpeg (P3 — Media complexidade)

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/service/media.go` | Adicionar `convertToOGG()` |
| `internal/service/message.go` | Chamar conversao antes do upload |
| `Dockerfile` | Adicionar `ffmpeg` |

**Dockerfile:**
```dockerfile
RUN apk add --no-cache ffmpeg
```

**Service:**
```go
func convertToOGG(input []byte) ([]byte, error) {
    cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-c:a", "libopus", "-f", "ogg", "pipe:1")
    cmd.Stdin = bytes.NewReader(input)
    var out bytes.Buffer
    cmd.Stdout = &out
    return out.Bytes(), cmd.Run()
}
```

---

### FR-008: Prometheus Metrics (P3 — Alta complexidade)

**Nova dependencia:** `github.com/prometheus/client_golang/prometheus` e `promhttp`

**Arquivos:**

| Arquivo | Mudanca |
|---------|---------|
| `internal/metrics/metrics.go` | NOVO — definir counters, gauges, histograms |
| `internal/handler/metrics.go` | NOVO — handler `/metrics` |
| `internal/server/router.go` | Registrar rota publica `/metrics` |
| `internal/service/session.go` | Incrementar gauges em connect/disconnect |
| `internal/service/message.go` | Incrementar counters em send |
| `internal/webhook/dispatcher.go` | Incrementar counters em deliver |
| `go.mod` | Adicionar dependencia |

**Metrics:**
```
wzap_sessions_total (gauge)            — total de sessoes criadas
wzap_sessions_connected (gauge)        — sessoes conectadas agora
wzap_messages_sent_total (counter)     — mensagens enviadas
wzap_messages_received_total (counter) — mensagens recebidas
wzap_webhooks_delivered_total (counter) — webhooks entregues
wzap_webhooks_failed_total (counter)   — webhooks falharam
wzap_webhooks_duration_seconds (histogram)
```

## Ordem de Implementacao

| Ordem | Feature | Prioridade | Complexidade | Novos Arquivos |
|-------|---------|-----------|-------------|---------------|
| 1 | FR-003 Mencoes @user | P2 | Baixa | 0 (modifica 2) |
| 2 | FR-005 Revogar link convite | P2 | Baixa | 0 (modifica 3) |
| 3 | FR-004 Atualizar nome perfil | P2 | Baixa | 0 (modifica 4) |
| 4 | FR-010 Restart instancia | P3 | Baixa | 0 (modifica 3) |
| 5 | FR-002 Encaminhar mensagem | P1 | Media | 0 (modifica 4) |
| 6 | FR-001 Status/Stories | P1 | Media | 0 (modifica 4) |
| 7 | FR-009 Marcar nao lido | P3 | Media | 0 (modifica 4) |
| 8 | FR-006 Webhook por evento | P2 | Alta | 1 (migration) |
| 9 | FR-007 Conversao audio | P3 | Media | 0 (+ Dockerfile) |
| 10 | FR-008 Prometheus metrics | P3 | Alta | 2 (metrics + handler) |

## Complexity Tracking

Nenhuma violacao da constituicao. Todas as features seguem o padrao handler -> service -> repo existente.
