# WA Engine Rules (internal/wa/)

## Manager Struct

```go
type Manager struct {
    clients      map[string]*whatsmeow.Client  // sessionID → client
    sessionNames map[string]string             // sessionID → name cache
    mu           sync.RWMutex

    ctx           context.Context
    sessionRepo   *repo.SessionRepository
    container     *sqlstore.Container
    nats          *broker.NATS
    dispatcher    *webhook.Dispatcher
    cfg           *config.Config
    waLog         waLog.Logger
    OnMediaReceived   MediaAutoUploadFunc
    OnMessageReceived MessagePersistFunc
}
```

- `clients` and `sessionNames` are the only mutable state, protected by a single `sync.RWMutex`.
- Callbacks (`OnMediaReceived`, `OnMessageReceived`) set post-construction via setters.

## Concurrency Pattern: Phase-Based Locking

The lock is **never held during I/O** (network calls, DB queries). Four-phase pattern in `Connect()`:

1. **Read lock** — check if client exists in map.
2. **No lock** — load device, create whatsmeow client (no map access).
3. **Write lock** — insert into map (map operation only).
4. **No lock** — call `client.Connect()` (network I/O).

## Dual Event Handlers

When a client is created, **two** event handlers are registered:

1. **Generic handler** (`m.handleEvent`) — logs, serializes, publishes to NATS, dispatches webhooks.
2. **Lifecycle handler** — manages DB state (JID updates, status changes, device cleanup on logout).

```go
client.AddEventHandler(func(evt interface{}) { m.handleEvent(sessionID, evt) })
client.AddEventHandler(func(evt interface{}) {
    switch v := evt.(type) {
    case *events.Connected:    m.sessionRepo.UpdateJID(...)
    case *events.PairSuccess:  m.sessionRepo.UpdateJID(...) + clear QR
    case *events.Disconnected: m.sessionRepo.UpdateStatus(..., "disconnected")
    case *events.LoggedOut:    client.Store.Delete(ctx); delete from map; clear device
    }
})
```

## Event Handler Pipeline (`handleEvent`)

1. Type switch → map to `model.EventType`.
2. Event-specific processing (media upload, message persistence).
3. Filter irrelevant events (e.g., non-read/played receipts → early return).
4. Serialize event data to JSON map.
5. Remove large/unsafe fields (`RawMessage`, `SourceWebMsg`).
6. Wrap in standardized envelope.
7. Publish to NATS (`wzap.events.<sessionID>`) with 512KB size guard.
8. Dispatch webhooks via goroutine.

## Event Envelope Format

```go
payload := map[string]interface{}{
    "event":     eventType,
    "eventId":   uuid.NewString(),
    "session":   map[string]interface{}{"id": sessionID, "name": m.getSessionName(sessionID)},
    "timestamp": time.Now().Format(time.RFC3339),
    "data":      data,
}
```

## Message Content Extraction

`extractMessageContent(msg)` — pure function returning `(msgType, body, mediaType)` via priority-based type switch:
- Text → Image → Video → Audio → Document → Sticker → Contact → Location → List → Buttons → Poll → unknown.

## Callback Pattern

Callbacks are set from services via setter methods (breaks dependency cycle):

```go
type MediaAutoUploadFunc func(sessionID, messageID, mimeType string, downloadable whatsmeow.DownloadableMessage)
type MessagePersistFunc func(sessionID, messageID, ...)

engine.SetMediaAutoUpload(mediaSvc.AutoUploadMedia)
engine.SetMessagePersist(historySvc.PersistMessage)
```

Called synchronously inside `handleEvent`. The actual work runs in a goroutine inside the service.

## Session Name Cache

`getSessionName(sessionID)` checks in-memory cache first, falls back to DB, caches result. Populated during `Connect`, `ReconnectAll`, and `UpdateSessionName`.

## QR Code Flow

Background goroutine consumes QR channel:
- `"code"` → persist QR code to DB, render to terminal.
- `"timeout"` → clear QR code, set status "disconnected", remove client from map.
- `"success"` → clear QR code, set status "connected".

## NATS Publishing

- Subject: `wzap.events.<sessionID>`.
- Max payload: 512KB — events exceeding this are skipped.
- Errors logged with `logger.Error()`.

## JID Helper (wa/jid.go)

```go
func EnsureJIDSuffix(jid string) string {
    if jid == "" || strings.Contains(jid, "@") { return jid }
    return jid + "@" + types.DefaultUserServer
}
```

Public function used across services for phone number → JID conversion.

## Error Handling in WA Layer

- Status updates to DB are fire-and-forget: `_ = m.sessionRepo.UpdateStatus(...)`.
- Logout failure: log warning, fall back to `Disconnect()` + `Store.Delete()`.
- Logging by severity: `Info` (connections), `Warn` (disconnections), `Error` (failures), `Debug` (receipts, contacts).

## Testing

Same-package tests (`package wa`) targeting unexported `extractMessageContent`. Standard `testing.T`, protobuf construction with `proto.String()`.
