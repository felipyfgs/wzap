# Testing Rules

## Test Package Convention

- External test packages for handlers, DTOs, middleware: `package handler_test`, `package dto_test`, `package middleware_test`.
- Internal test packages for services and wa: `package service`, `package wa` — needed to test unexported helpers.

## Test Naming

`Test<Unit>_<Scenario>` — e.g., `TestSendText_MissingPhone`, `TestBuildContextInfo_Full`, `TestParseJID_Phone`.

## No Assertion Libraries

Use standard `testing.T` only:
- `t.Errorf("expected X, got %d", val)` — non-fatal, test continues.
- `t.Fatalf("test error: %v", err)` — fatal, test stops.
- `t.Error("expected error")` — non-fatal boolean check.

## Test App Factory

Each handler test file defines a private factory function:

```go
func newXxxApp(svc *service.XxxService) *fiber.App {
    app := fiber.New(fiber.Config{DisableStartupMessage: true})
    app.Use(recover.New())
    h := handler.NewXxxHandler(svc)
    sess := app.Group("/sessions/:sessionId")
    sess.Use(func(c *fiber.Ctx) error {
        c.Locals("sessionID", c.Params("sessionId"))
        return c.Next()
    })
    sess.Post("/xxx", h.Method)
    return app
}
```

- `DisableStartupMessage: true` always set.
- `recover.New()` always installed.
- Fake middleware via inline `c.Locals()` setters (bypass real auth/session middleware).
- Admin routes inject `c.Locals("authRole", "admin")` via inline middleware.

## Nil-Dependency Injection (No Mocks)

No mocking frameworks. Tests pass `nil` to constructors:

```go
service.NewMessageService(nil)           // engine = nil
service.NewSessionService(nil, nil, nil) // all deps nil
handler.NewHealthHandler(nil, nil, nil)  // all deps nil
```

Tests focus on input validation and HTTP-layer behavior — they validate that bad requests return 400 before hitting nil dependencies.

## HTTP Test Request Pattern

```go
body, _ := json.Marshal(dto.SendTextReq{Body: "hello"})
req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/text", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
resp, err := app.Test(req, -1)
if err != nil {
    t.Fatalf("test error: %v", err)
}
if resp.StatusCode != http.StatusBadRequest {
    t.Errorf("expected 400, got %d", resp.StatusCode)
}
```

- Always `app.Test(req, -1)` (no timeout).
- Always set `Content-Type: application/json` for POST/PUT.
- Bad JSON: `bytes.NewBufferString("{bad")`.

## Assertions

Most handler tests check **only the status code** — they do not decode the response body.

```go
if resp.StatusCode != http.StatusBadRequest {
    t.Errorf("expected 400, got %d", resp.StatusCode)
}
```

## Validator Initialization

When testing handlers, ensure the validator is initialized:

```go
var _ = middleware.Validate  // at package level
```

## shared Test Utilities (`internal/testutil/fiber.go`)

- `NewApp()` — creates test Fiber app.
- `DoRequest(app, method, path, body)` — executes HTTP request.
- `ParseResp(body)` — parses response into `map[string]interface{}`.

These exist but current tests inline the patterns instead.

## What is NOT Tested

- No integration tests (no database, NATS, or MinIO).
- No tests for successful handler responses with correct data.
- No table-driven tests (`t.Run()` subtests).
- No benchmark tests.
- No mocking frameworks or generated mocks.

## Protobuf in Tests

Construct protobuf messages with `proto.String()` wrapper:

```go
msg := &waE2E.Message{
    ImageMessage: &waE2E.ImageMessage{
        Caption: proto.String("caption"),
    },
}
```
