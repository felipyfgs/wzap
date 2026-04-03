# DTO and Model Rules

## Naming Conventions

- Request DTOs: `<Action><Resource>Req` — e.g., `SendTextReq`, `CreateWebhookReq`, `SessionUpdateReq`.
- Response DTOs: `<Resource>Resp` — e.g., `SessionResp`, `MidResp`, `GroupInviteLinkResp`.
- Small single-purpose response DTOs: `MidResp`, `ConnectResp`, `QRResp`, `PairPhoneResp`.
- Domain models in `internal/model/`, DTOs in `internal/dto/`.

## JSON Tags

- Always `camelCase`: `json:"messageId"`, `json:"groupJid"`, `json:"baseUrl"`.
- `omitempty` on optional fields, sensitive fields (`token`, `secret`), and nullable columns.
- No `omitempty` on always-populated fields (`id`, `name`, `status`).
- Acronyms in JSON: `JID` → `jid`/`groupJid`, `URL` → `url`/`pictureUrl`, `NATS` → `natsEnabled`.

## Validation Tags (Request DTOs only)

```go
validate:"required"           // mandatory fields
validate:"required,url"       // URL fields (only webhook URL)
validate:"required,min=1"     // non-empty slices (buttons, sections, rows)
validate:"required,min=2"     // poll options (minimum 2)
```

- No validation tags on models or response DTOs.
- No validation tags on update DTOs (all fields are optional pointers).

## Create vs Update DTOs

**Create** — value types:
```go
type SessionCreateReq struct {
    Name     string          `json:"name" validate:"required"`
    Proxy    SessionProxy    `json:"proxy,omitempty"`
}
```

**Update** — pointer types (nil = not provided):
```go
type SessionUpdateReq struct {
    Name     *string         `json:"name,omitempty"`
    Proxy    *SessionProxy   `json:"proxy,omitempty"`
}
```

## API Response Envelope (`dto/response.go`)

```go
// Success
func SuccessResp(data interface{}) APIResponse

// Error
func ErrorResp(title string, message string) APIError
```

Never construct `APIResponse` or `APIError` directly — always use factory functions.

## Model Conventions

- No `validate` tags.
- `CreatedAt` and `UpdatedAt` on all persisted entities.
- `SessionID` as foreign key on child entities.
- `Raw any` for unstructured data (`model.Message.Raw`).

## Shared Structs Between Model and DTO

`SessionProxy` and `SessionSettings` are duplicated identically in both packages. Convert via direct type cast:

```go
dto.SessionProxy(s.Proxy)      // model → dto
model.SessionProxy(req.Proxy)   // dto → model
```

## Model-to-DTO Conversion

Use helper function `dto.SessionToResp(s model.Session) SessionResp` — `Token` is intentionally excluded from `SessionResp` (only in `SessionCreatedResp`).

## Enums (model/events.go)

```go
type EventType string
const EventMessage EventType = "Message"

var ValidEventTypes = func() map[EventType]bool {
    types := []EventType{EventMessage, ...}
    m := make(map[EventType]bool, len(types))
    for _, t := range types { m[t] = true }
    return m
}()

func IsValidEventType(e EventType) bool { return ValidEventTypes[e] }
```

- `EventType` is a `string` type alias (not `int` iota).
- String values are PascalCase: `"Message"`, `"PairSuccess"`.
- `EventAll = "All"` is the wildcard subscription.

## Nested Sub-Structs

Complex DTOs decompose into reusable sub-structs defined in `internal/dto/`:
- `ReplyContext` — used by send message DTOs.
- `ButtonItem`, `ListSection`, `ListRow` — used by template DTOs.
- `SessionProxy`, `SessionSettings` — embedded in session DTOs and models.
