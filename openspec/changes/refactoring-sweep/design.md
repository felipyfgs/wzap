# Design — Refactoring Sweep

## Architecture Overview

```
                    ┌─────────────────────────────────────────┐
                    │         NEW UTILITY PACKAGES            │
                    ├─────────────────────────────────────────┤
                    │  internal/wautil/                       │
                    │    ├─ message.go  (extractMessageContent,│
                    │    │               extractMediaDownloadInfo,│
                    │    │               inferChatType)         │
                    │    └─ helpers.go (stringPtr, intPtr,     │
                    │                    unixTimePtr, etc.)     │
                    │                                         │
                    │  internal/imgutil/                      │
                    │    └─ convert.go (WebP→PNG, WebP→GIF,   │
                    │                   palette, paletted)     │
                    └─────────────────────────────────────────┘
                                     ▲
                                     │ imports
                    ┌────────────────┼────────────────────────┐
                    │                │                         │
       ┌────────────┴──┐    ┌───────┴───────┐    ┌────────────┴──┐
       │ internal/wa/  │    │ internal/     │    │ internal/     │
       │               │    │ service/      │    │ integrations/ │
       │ events.go     │    │ history.go    │    │ chatwoot/     │
       │ (refactored)  │    │ (slimmed)     │    │ (DRY-ed)      │
       └───────────────┘    └───────────────┘    └───────────────┘
```

## Phase 1: Quick Wins

### 1.1 Remove `service/message_status.go`

Simple deletion. No code references this file.

### 1.2 Create `internal/wautil/message.go`

Move these functions from their current locations:

| Function | Current Location | New Location |
|----------|-----------------|--------------|
| `extractMessageContent` | `wa/events.go:532` AND `service/history.go:540` | `wautil/message.go` |
| `extractMediaDownloadInfo` | `service/history.go:515` | `wautil/message.go` |
| `inferChatType` | `service/history.go:463` | `wautil/message.go` |

Also move small helper functions used by these:
| `stringPtr`, `intPtr`, `unixTimePtr`, `uint64ToInt64`, `firstNonEmpty` | `service/history.go:480-513` | `wautil/helpers.go` |

**Import changes**:
- `wa/events.go`: add `"wzap/internal/wautil"`, remove local `extractMessageContent`
- `service/history.go`: add `"wzap/internal/wautil"`, remove local copies of all moved functions

**Package**: `wautil` depends on `go.mau.fi/whatsmeow/proto/waE2E` for the message type. No internal dependencies.

### 1.3 Unify runtime dispatch functions

**File**: `internal/service/session_runtime.go`

Replace `runSessionRuntime` and `runConnectedRuntime` with:

```go
type clientResolver func() (*whatsmeow.Client, error)

func runClientRuntime[T any](
    ctx context.Context,
    runtime *SessionRuntime,
    resolveClient clientResolver,
    cloud func(context.Context, *model.Session, *cloudWA.Client) (T, error),
    whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error),
) (T, error) {
    var zero T
    if runtime == nil || runtime.session == nil {
        return zero, fmt.Errorf("session runtime is nil")
    }
    ctx = runtime.WithContext(ctx)
    if runtime.IsCloudAPI() {
        if cloud == nil {
            return zero, fmt.Errorf("cloud runtime handler is nil")
        }
        if runtime.provider == nil {
            return zero, fmt.Errorf("cloud provider unavailable")
        }
        return cloud(ctx, runtime.session, runtime.provider)
    }
    if whatsmeowFn == nil {
        return zero, fmt.Errorf("whatsmeow runtime handler is nil")
    }
    client, err := resolveClient()
    if err != nil {
        return zero, err
    }
    return whatsmeowFn(ctx, runtime.session, client)
}

func runSessionRuntime[T any](...) (T, error) {
    return runClientRuntime(ctx, runtime, runtime.Client, cloud, whatsmeowFn)
}

func runConnectedRuntime[T any](...) (T, error) {
    return runClientRuntime(ctx, runtime, runtime.ConnectedClient, cloud, whatsmeowFn)
}
```

`runRuntimeErr` simplifies to delegate to `runSessionRuntime`.

---

## Phase 2: Chatwoot Package DRY

### 2.1 `@lid` resolution helper

Add to `internal/integrations/chatwoot/jid.go`:

```go
// resolveLID resolves a @lid JID to a phone-number JID.
// It tries altJIDs first (in order), then falls back to the JID resolver.
// Returns the original jid unchanged if it's not a @lid.
func (s *Service) resolveLID(ctx context.Context, sessionID, jid string, altJIDs ...string) string {
    if !strings.HasSuffix(jid, "@lid") {
        return jid
    }
    for _, alt := range altJIDs {
        if alt != "" && !strings.HasSuffix(alt, "@lid") {
            if !strings.Contains(alt, "@") {
                return alt + "@s.whatsapp.net"
            }
            return alt
        }
    }
    return s.resolveJID(ctx, sessionID, jid)
}
```

**Call site migration** (10+ locations in `inbound_events.go` and `inbound_message.go`):

Before:
```go
if strings.HasSuffix(jid, "@lid") {
    pJID = s.resolveJID(ctx, cfg.SessionID, pJID)
}
```

After:
```go
pJID = s.resolveLID(ctx, cfg.SessionID, pJID)
```

The `handleMessage` LID resolution block (lines 36-65) becomes:
```go
chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
if strings.HasSuffix(chatJID, "@lid") {
    // still unresolvable
    logger.Warn()...
    return nil
}
```

### 2.2 Message params builder

Add to `internal/integrations/chatwoot/inbound_message.go`:

```go
type cwMsgParams struct {
    MessageType    string
    SourceID       string
    ContentAttrs   map[string]any
    SourceReplyID  int
}

func newCWMsgParams(fromMe bool, msgID, stanzaID string, cwReplyID int) cwMsgParams {
    p := cwMsgParams{
        MessageType: "incoming",
    }
    if fromMe {
        p.MessageType = "outgoing"
    }
    if msgID != "" {
        p.SourceID = "WAID:" + msgID
    }
    if stanzaID != "" {
        p.ContentAttrs = map[string]any{"reply_source_id": "WAID:" + stanzaID}
        if cwReplyID > 0 {
            p.ContentAttrs["in_reply_to"] = cwReplyID
            p.SourceReplyID = cwReplyID
        }
    }
    return p
}
```

Used by: `handleMediaMessage`, `handleStickerMessage`, `handlePollCreation`, `handleReaction`, `handleButtonResponse`, `handleListResponse`, `handleTemplateReply`, `handleViewOnce`.

### 2.3 Image conversion to `internal/imgutil`

Create `internal/imgutil/convert.go` with:
- `ConvertWebPToPNG(data []byte) ([]byte, error)` — exported
- `ConvertWebPToGIF(data []byte) ([]byte, error)` — exported
- `imageToPaletted(img image.Image) *image.Paletted` — unexported
- `buildPalette(img image.Image, bounds image.Rectangle) color.Palette` — unexported
- `rgbaKey` struct — unexported

Imports: `bytes`, `image`, `image/color`, `image/gif`, `image/png`, `_ golang.org/x/image/webp`

`inbound_message.go` removes all image-related imports and calls `imgutil.ConvertWebPToPNG(data)` / `imgutil.ConvertWebPToGIF(data)`.

### 2.4 Idempotency extraction

Extract from `HandleIncomingWebhook`:

```go
func (s *Service) isOutboundDuplicate(ctx context.Context, sessionID string, msg *dto.ChatwootWebhookMessage) bool {
    sourceID := msg.SourceID
    if sourceID != "" {
        if exists, err := s.msgRepo.ExistsBySourceID(ctx, sessionID, sourceID); err == nil && exists {
            logger.Debug()...
            metrics.CWIdempotentDrops...
            return true
        }
    }
    if msg.ID > 0 {
        cwIdemKey := fmt.Sprintf("cw-out:%d", msg.ID)
        if s.cache.GetIdempotent(ctx, sessionID, cwIdemKey) {
            logger.Debug()...
            metrics.CWIdempotentDrops...
            return true
        }
        s.cache.SetIdempotent(ctx, sessionID, cwIdemKey)
    }
    return false
}
```

---

## Phase 3: wa/events.go Decomposition

### 3.1 Split `handleEvent`

Current single method → three functions:

```go
// classifyEvent returns the event type for a whatsmeow event.
// Returns ("", false) for events that should be skipped entirely.
func classifyEvent(evt any) (model.EventType, bool)

// serializeEventData builds the data map for NATS/webhook dispatch.
func serializeEventData(evt any, eventType model.EventType) map[string]any

// dispatchEvent publishes to NATS and dispatches via webhook.
func (m *Manager) dispatchEvent(sessionID, sessionName string, eventType model.EventType, data map[string]any)
```

`handleEvent` becomes:
```go
func (m *Manager) handleEvent(sessionID string, evt any) {
    eventType, ok := classifyEvent(evt)
    if !ok || eventType == "" {
        return
    }
    data := serializeEventData(evt, eventType)
    nameCtx, nameCancel := context.WithTimeout(context.Background(), 3*time.Second)
    sessionName := m.getSessionName(nameCtx, sessionID)
    nameCancel()
    m.dispatchEvent(sessionID, sessionName, eventType, data)
}
```

### 3.2 Single-pass classify + serialize

Instead of two type switches, `classifyEvent` returns both the type and a pre-built partial data map:

```go
type classifiedEvent struct {
    Type model.EventType
    Data map[string]any  // nil if default serialization should be used
}

func classifyEvent(evt any) classifiedEvent
```

For most event types, `Data` is nil and `serializeEventData` uses the default JSON encode/decode path. For `HistorySync` and `AppState`, `Data` is populated inline.

---

## Phase 4: Error Handling & Runtime Cleanup

### 4.1 Log silently discarded errors

**Strategy**: Replace `_ =` with `if err := ...; err != nil { logger.Warn()... }` for these categories:

| Category | Files | Pattern |
|----------|-------|---------|
| DB updates (UpdateChatwootRef, Save, MarkImported) | inbound_message.go, outbound.go, service.go | `_ = s.msgRepo.XXX(...)` |
| NATS ack/nak | consumer.go | `_ = msg.Nak()` / `_ = msg.Ack()` |
| Chatwoot API calls | inbound_events.go | `_ = client.UpdateContact(...)` / `_ = client.DeleteMessage(...)` |
| Session repo updates | connect.go | `_ = m.sessionRepo.UpdateStatus(...)` / `_ = m.sessionRepo.UpdateQRCode(...)` |

**Excluded** (acceptable `_ =`):
- `defer res.Body.Close()` / `_, _ = io.Copy(io.Discard, res.Body)` — cleanup, error doesn't matter
- `_, _ = client.CreateMessage(...)` in bot replies — best-effort notifications

### 4.2 Remove nil-receiver checks

Remove `if r == nil { return ... }` from internal methods on:
- `SessionRuntime`: `Session()`, `Engine()`, `Provider()`, `WithContext()`, `CloudConfig()`, `Client()`, `ConnectedClient()`, `RequireCapability()`
- `MessageRuntime`: `Support()`
- `MediaRuntime`: `Support()`
- `StatusRuntime`: `Support()`
- `ProfileRuntime`: `Support()`
- `RuntimeResolver`: `SetProvider()`

Keep nil checks only on `Resolve()` and `resolveCapability()` where they guard against misconfiguration.

### 4.3 Deduplicate `Support()` methods

Since `MessageRuntime`, `MediaRuntime`, `StatusRuntime`, `ProfileRuntime` all embed `*SessionRuntime` and have a `support` field, add a single method:

```go
// On the embedded support field directly — no wrapper method needed.
// Callers access runtime.support directly, or we add one method on a shared interface.
```

Alternatively, since these are small structs with identical layout, consider a generic:

```go
type CapabilityRuntime[S any] struct {
    *SessionRuntime
    support model.CapabilitySupport
}
```

But this may be over-engineering. Simplest: just keep the 4 methods but remove the nil checks (they're 3 lines each, acceptable).

---

## Migration Strategy

Each phase is a single PR with:
1. Create new files/packages first (additive, no breakage)
2. Update call sites to use new packages
3. Remove old code
4. Run `go build ./...` and `go test -v -race ./...`

No phase should touch more than ~5-8 files.
