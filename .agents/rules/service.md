# Service Rules

## Struct and Constructor

```go
type XxxService struct {
    engine *wa.Manager
    repo   *repo.XxxRepository
}

func NewXxxService(engine *wa.Manager, repo *repo.XxxRepository) *XxxService {
    return &XxxService{engine: engine, repo: repo}
}
```

- `*wa.Manager` is present in 9 of 11 services (all except `WebhookService` and `HistoryService`).
- All dependencies are concrete types (no interfaces).
- Only `SessionService` has two repos (`SessionRepository` + `WebhookRepository`).

## The "GetClient + Guard" Pattern

Almost every service method that interacts with WhatsApp follows:

```go
func (s *XxxService) Method(ctx context.Context, sessionID string, ...) (T, error) {
    client, err := s.engine.GetClient(sessionID)
    if err != nil {
        return zero, err
    }
    if !client.IsConnected() {
        return zero, fmt.Errorf("client not connected")
    }
    jid, err := parseJID(target)
    if err != nil {
        return zero, err
    }
    result, err := client.SomeMethod(ctx, jid, ...)
    if err != nil {
        return zero, fmt.Errorf("failed to X: %w", err)
    }
    return result, nil
}
```

- Some methods also check `client.Store.ID == nil` (not logged in).
- Read-only operations (e.g., `ContactService.List`) skip the `IsConnected()` check.

## Return Type Patterns

| Operation | Return |
|---|---|
| Send message | `(string, error)` — string is WhatsApp message ID |
| Query single | `(*dto.XxxResp, error)` or `(*model.T, error)` |
| Query list | `([]dto.XxxResp, error)` or `([]model.T, error)` |
| Action (no data) | `error` only |
| Action (scalar) | `(string, error)` |

## JID Parsing

- `parseJID(target)` — package-level unexported in `service/message.go`. Smart parser: accepts bare phone numbers OR full JIDs.
- `types.ParseJID(wa.EnsureJIDSuffix(p))` — used in `group.go` and `community.go` for phone number inputs.
- `types.ParseJID(jidStr)` — used when input is already a full JID (groups, newsletters).

## Async / Fire-and-Forget

```go
func (s *HistoryService) PersistMessage(...) {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        // ... save to DB
    }()
}
```

These methods do NOT accept `context.Context` as a parameter — they create their own.

## Error Handling

- Wrap errors: `fmt.Errorf("failed to X: %w", err)`.
- Pass-through (no wrapping) when upstream error is already descriptive.
- Business-rule errors: `fmt.Errorf("name is required")`, `fmt.Errorf("invalid action, must be add, remove, promote or demote")`.
- Non-fatal errors logged with `logger.Warn().Err(err)...Msg(...)`, not returned.

## Logging in Services

Only `logger.Warn()` and `logger.Debug()` are used in the service layer. No `logger.Info()` or `logger.Error()` — those belong in the `wa/` layer.

## Model-to-DTO Conversion

Use `dto.SessionToResp(s)` helper or direct struct literal with type casting:

```go
dto.SessionProxy(modelProxy)  // direct type cast for identical structs
```

## Partial Updates (Update DTOs with Pointers)

```go
if req.Name != nil { session.Name = *req.Name }
if req.Proxy != nil { session.Proxy = model.SessionProxy(*req.Proxy) }
```

## Send Operations Return Message ID

```go
resp, err := client.SendMessage(ctx, jid, msg, opts...)
if err != nil {
    return "", fmt.Errorf("failed to send text message: %w", err)
}
return resp.ID, nil
```

## Slice Initialization

Pre-allocate when capacity is known: `make([]T, 0, len(src))`.
