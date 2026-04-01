# Code Patterns — wzap Service Integration

Concrete, copy-paste-ready snippets from the actual codebase. All examples follow the established conventions.

---

## 1. DTO (internal/dto/)

```go
// internal/dto/message.go

type SendTextReq struct {
    JID  string `json:"jid"`
    Text string `json:"text"`
}

type SendTextResp struct {
    MessageID string `json:"messageId"`
}
```

- Use `json:"camelCase"` tags.
- Add `omitempty` only when nil/zero is meaningful.
- Keep request DTOs separate from response DTOs.

---

## 2. Handler (internal/handler/)

```go
// internal/handler/message.go

type MessageHandler struct {
    messageSvc *service.MessageService
}

func NewMessageHandler(messageSvc *service.MessageService) *MessageHandler {
    return &MessageHandler{messageSvc: messageSvc}
}

// SendText godoc
// @Summary     Send text message
// @Description Sends a plain text message to a WhatsApp JID
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path     string           true "Session name or ID"
// @Param       body      body     dto.SendTextReq  true "Message payload"
// @Success     200       {object} dto.APIResponse
// @Failure     400       {object} dto.APIResponse
// @Security    ApiKey
// @Router      /sessions/{sessionId}/messages/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
    id := c.Locals("sessionId").(string)

    var req dto.SendTextReq
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
    }

    msgID, err := h.messageSvc.SendText(c.Context(), id, req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
    }

    return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}))
}
```

**Admin-only guard** (place at the top of the handler body):
```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

---

## 3. Service (internal/service/)

```go
// internal/service/message.go

type MessageService struct {
    engine *wa.Manager
}

func NewMessageService(engine *wa.Manager) *MessageService {
    return &MessageService{engine: engine}
}

func (s *MessageService) SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error) {
    client, err := s.engine.GetClient(sessionID)
    if err != nil {
        return "", err
    }
    if !client.IsConnected() {
        return "", fmt.Errorf("client not connected")
    }

    jid, err := parseJID(req.JID)
    if err != nil {
        return "", err
    }

    msg := &waProto.Message{
        Conversation: proto.String(req.Text),
    }

    resp, err := client.SendMessage(ctx, jid, msg)
    if err != nil {
        return "", fmt.Errorf("failed to send text message: %w", err)
    }

    return resp.ID, nil
}
```

Key rules:
- Always call `engine.GetClient(sessionID)` first.
- Always check `client.IsConnected()` before sending.
- Wrap errors: `fmt.Errorf("failed to <action>: %w", err)`.
- The `parseJID` helper lives in `internal/service/message.go`; reuse it or copy it.

---

## 4. Repo (internal/repo/)

### QueryRow — single result

```go
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
    query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
        "connected", "status", "proxy", "settings", "createdAt", "updatedAt"
        FROM "wzSessions" WHERE "id" = $1`

    var s model.Session
    err := r.db.QueryRow(ctx, query, id).Scan(
        &s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected,
        &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("session not found: %w", err)
    }
    return &s, nil
}
```

### Query — multiple results

```go
func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
    query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
        "connected", "status", "proxy", "settings", "createdAt", "updatedAt"
        FROM "wzSessions" ORDER BY "createdAt" DESC`

    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query sessions: %w", err)
    }
    defer rows.Close()

    var sessions []model.Session
    for rows.Next() {
        var s model.Session
        if err := rows.Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected,
            &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan session: %w", err)
        }
        sessions = append(sessions, s)
    }
    return sessions, rows.Err()
}
```

### Exec — insert / update / delete

```go
func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
    _, err := r.db.Exec(ctx,
        `UPDATE "wzSessions" SET "status" = $1 WHERE "id" = $2`,
        status, id,
    )
    if err != nil {
        return fmt.Errorf("failed to update status for session %s: %w", id, err)
    }
    return nil
}
```

---

## 5. Route registration (internal/server/router.go)

```go
// Session-scoped (requires valid session apiKey or admin apiKey + :sessionId):
sess.Post("/messages/text", messageHandler.SendText)
sess.Get("/contacts", contactHandler.List)

// Admin-only (requires global API_KEY):
grp.Post("/sessions", sessionHandler.Create)
grp.Get("/sessions", sessionHandler.List)
```

- `grp` = `s.App.Group("/", middleware.Auth(...))` — auth required, no session resolved.
- `sess` = `grp.Group("/sessions/:sessionId", reqSession)` — auth + session ID resolved into `c.Locals("sessionId")`.

---

## 6. Dependency wiring (internal/server/router.go)

When adding a new handler that needs a new service:

```go
// 1. Instantiate the service (in SetupRoutes)
myNewSvc := service.NewMyNewService(engine)

// 2. Instantiate the handler
myNewHandler := handler.NewMyNewHandler(myNewSvc)

// 3. Register routes
sess.Post("/my-domain/action", myNewHandler.DoAction)
```

---

## 7. Response helpers

```go
// Success
return c.JSON(dto.SuccessResp(data))
return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(session))

// Error
return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "session not found"))
return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
```
