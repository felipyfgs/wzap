# Code Patterns — wzap Service Integration

Copy-paste-ready snippets from the real codebase.

## DTO

```go
type SendTextReq struct {
    Phone string `json:"phone" validate:"required"`
    Body  string `json:"body" validate:"required"`
}

type MidResp struct {
    Mid string `json:"messageId"`
}
```

- Request: `<Action><Resource>Req`. Response: `<Resource>Resp`.
- `json:"camelCase"`. `omitempty` on optional/sensitive fields.
- Update DTOs use pointer fields: `*string`, `*bool` (nil = not provided).
- Slice fields: `validate:"required,min=1"`.

## Handler

```go
type MessageHandler struct {
    msgSvc *service.MessageService
}

func NewMessageHandler(msgSvc *service.MessageService) *MessageHandler {
    return &MessageHandler{msgSvc: msgSvc}
}

// SendText godoc
// @Summary     Send a text message
// @Description Sends a plain text message to a WhatsApp JID (user or group)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body     dto.SendTextReq true "Message payload"
// @Success     200  {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400  {object} dto.APIError
// @Failure     500  {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
    id, err := getSessionID(c)
    if err != nil {
        return err
    }
    var req dto.SendTextReq
    if err := parseAndValidate(c, &req); err != nil {
        return err
    }

    msgID, err := h.msgSvc.SendText(c.Context(), id, req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
    }

    return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}
```

### Admin-only guard (top of handler body)

```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

### Response helpers

```go
c.JSON(dto.SuccessResp(data))                                    // 200
c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(data))        // 201
c.JSON(dto.SuccessResp(nil))                                     // 200 ack
c.Status(400).JSON(dto.ErrorResp("Bad Request", msg))            // 400
c.Status(404).JSON(dto.ErrorResp("Not Found", msg))              // 404
c.Status(500).JSON(dto.ErrorResp("Internal Server Error", msg))  // 500
```

### Error titles by context

`"Bad Request"`, `"Validation Error"`, `"Forbidden"`, `"Not Found"`, `"Create Error"`, `"Update Error"`, `"Delete Error"`, `"List Error"`, `"Send Error"`, `"Connection Error"`, `"Disconnect Error"`, `"Pair Error"`.

## Service

```go
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

    jid, err := parseJID(req.Phone)
    if err != nil {
        return "", err
    }

    msg := &waE2E.Message{
        Conversation: proto.String(req.Body),
    }

    resp, err := client.SendMessage(ctx, jid, msg)
    if err != nil {
        return "", fmt.Errorf("failed to send text message: %w", err)
    }

    return resp.ID, nil
}
```

Key rules:
- Always `engine.GetClient(sessionID)` first, then `client.IsConnected()`.
- Wrap errors with `%w`. Return WhatsApp message ID (`resp.ID`) from send ops.
- `parseJID` handles bare phone numbers and full JIDs.

## Repo

### QueryRow — single result

```go
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
    query := `SELECT id, name, COALESCE(token, ''), COALESCE(jid, ''), COALESCE(qr_code, ''),
        connected, status, proxy, settings, created_at, updated_at
        FROM wz_sessions WHERE id = $1`

    var s model.Session
    err := r.db.QueryRow(ctx, query, id).Scan(
        &s.ID, &s.Name, &s.Token, &s.JID, &s.QRCode,
        &s.Connected, &s.Status, &s.Proxy, &s.Settings,
        &s.CreatedAt, &s.UpdatedAt,
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
    query := `SELECT id, name, COALESCE(token, ''), COALESCE(jid, ''), COALESCE(qr_code, ''),
        connected, status, proxy, settings, created_at, updated_at
        FROM wz_sessions ORDER BY created_at DESC`

    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query sessions: %w", err)
    }
    defer rows.Close()

    var sessions []model.Session
    for rows.Next() {
        var s model.Session
        if err := rows.Scan(&s.ID, &s.Name, &s.Token, &s.JID, &s.QRCode,
            &s.Connected, &s.Status, &s.Proxy, &s.Settings,
            &s.CreatedAt, &s.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan session: %w", err)
        }
        sessions = append(sessions, s)
    }
    return sessions, rows.Err()
}
```

### Exec — insert / update / delete

```go
func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
    query := `INSERT INTO wz_sessions (id, name, token, status, proxy, settings, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := r.db.Exec(ctx, query,
        session.ID, session.Name, session.Token, session.Status,
        session.Proxy, session.Settings, session.CreatedAt, session.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("failed to insert session: %w", err)
    }
    return nil
}
```

### JSONB array containment

```go
func (r *WebhookRepository) FindActiveBySessionAndEvent(ctx context.Context, sessionID, eventType string) ([]model.Webhook, error) {
    query := `SELECT id, session_id, url, COALESCE(secret, ''),
        events, enabled, nats_enabled, created_at, updated_at
        FROM wz_webhooks
        WHERE session_id = $1 AND enabled = true
          AND (events @> $2::jsonb OR events @> '["All"]'::jsonb)`

    eventJSON, _ := json.Marshal([]string{eventType})
    rows, err := r.db.Query(ctx, query, sessionID, eventJSON)
    // ... rows.Close(), loop, rows.Err()
}
```

## Route registration

```go
// Admin-only (under grp):
grp.Post("/sessions", sessionHandler.Create)
grp.Get("/sessions", sessionHandler.List)

// Session-scoped (under sess):
sess.Post("/messages/text", messageHandler.SendText)
sess.Get("/contacts", contactHandler.List)
```

## Dependency wiring

```go
// In SetupRoutes — order: repos → hub/dispatcher → engine → services → callbacks → handlers
mySvc := service.NewMyService(engine)
myHandler := handler.NewMyHandler(mySvc)
sess.Post("/my-domain/action", myHandler.DoAction)
```
