# Handler Rules

## Struct and Constructor

```go
type XxxHandler struct {
    xxxSvc *service.XxxService
}

func NewXxxHandler(xxxSvc *service.XxxService) *XxxHandler {
    return &XxxHandler{xxxSvc: xxxSvc}
}
```

- Unexported fields, concrete types (no interfaces).
- Constructor returns `*XxxHandler`.
- Dependency field naming: `<camelCase>Svc` for services, or type name for infra (`db`, `nats`, `minio`, `cfg`).

## Method Template

```go
// MethodName godoc
// @Summary     <one-line summary>
// @Description <detailed description>
// @Tags        <Tag>
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.XxxReq true "Description"
// @Success     200 {object} dto.APIResponse{Data=dto.XxxResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/xxx [post]
func (h *XxxHandler) MethodName(c *fiber.Ctx) error {
    id, err := getSessionID(c)
    if err != nil {
        return err
    }
    var req dto.XxxReq
    if err := parseAndValidate(c, &req); err != nil {
        return err
    }
    result, err := h.xxxSvc.Method(c.Context(), id, req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error Title", err.Error()))
    }
    return c.JSON(dto.SuccessResp(result))
}
```

## Key Helpers (`internal/handler/helpers.go`)

- `parseAndValidate(c, &req)` — combines `BodyParser` + `validator.Struct`. Already writes error response on failure, returns `fiber.ErrBadRequest`.
- `mustGetSessionID(c)` — for routes behind `RequiredSession` middleware (safe, returns empty string if unset).
- `getSessionID(c)` — for routes that validate session ID at handler level (returns error if unset).

## Response Patterns

```go
c.JSON(dto.SuccessResp(data))                                        // 200
c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(data))            // 201
c.JSON(dto.SuccessResp(nil))                                         // 200 ack
c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", msg))  // 400
c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", msg))      // 404
c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Title", msg)) // 500
```

## Error Title Conventions

| Context | Title |
|---|---|
| Parse/validation | `"Bad Request"`, `"Validation Error"` |
| Auth | `"Forbidden"`, `"Unauthorized"` |
| Not found | `"Not Found"` |
| CRUD | `"Create Error"`, `"Update Error"`, `"Delete Error"`, `"List Error"` |
| Session lifecycle | `"Connection Error"`, `"Disconnect Error"`, `"Reconnect Error"`, `"Logout Error"`, `"Pair Error"` |
| Messaging | `"Send Error"` |

## Swagger Annotations

- Every handler method has a full godoc block: `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Security`, `@Router`.
- `@Accept json` is omitted for GET routes and routes without a body.
- `@Security Authorization` is present on all authenticated routes.
- `@Router` path uses `{sessionId}` with curly braces.
- `@Success` uses `dto.APIResponse{Data=dto.XxxResp}` for typed data, `dto.APIResponse` for nil data.
- `@Failure` always uses `dto.APIError`.

## Private Helper Methods

When multiple handlers share logic (e.g., media sending), use an unexported method that accepts a function parameter:

```go
func (h *MessageHandler) sendMedia(c *fiber.Ctx, sendFunc func(...) (string, error)) error { ... }
```

## Admin-Only Check

```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

## Query Parameters

```go
filter := c.Query("filter", "saved")
limit, _ := strconv.Atoi(c.Query("limit", "50"))
offset, _ := strconv.Atoi(c.Query("offset", "0"))
reset := c.QueryBool("reset", false)
```
