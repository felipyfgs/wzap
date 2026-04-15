# Spec: Error Handling & Runtime Cleanup

## Capability: Improve Error Visibility and Remove Dead Patterns

### Purpose

Replace silently discarded errors with logged warnings on critical paths, and remove unidiomatic nil-receiver checks.

### Changes

#### 1. Critical `_ =` Replacements

**DB operations** — replace `_ = s.msgRepo.XXX(...)` with logged error:

```go
if err := s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID); err != nil {
    logger.Warn().Str("component", "chatwoot").Err(err).Str("mid", msgID).Msg("failed to update chatwoot ref")
}
```

**Files**: `inbound_message.go` (8 occurrences), `outbound.go` (2), `service.go` (2)

**NATS operations** — replace `_ = msg.Nak()` / `_ = msg.Ack()` with logged error:

```go
if err := msg.Nak(); err != nil {
    logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to NAK message")
}
```

**File**: `consumer.go` (16 occurrences)

**Chatwoot API calls** — replace `_ = client.UpdateContact(...)` / `_ = client.DeleteMessage(...)`:

```go
if err := client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: name}); err != nil {
    logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to update contact")
}
```

**File**: `inbound_events.go` (3 occurrences)

**Session repo** — replace `_ = m.sessionRepo.UpdateStatus(...)` / `_ = m.sessionRepo.UpdateQRCode(...)`:

```go
if err := m.sessionRepo.UpdateQRCode(opCtx, sessionID, ""); err != nil {
    logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to clear QR code")
}
```

**File**: `connect.go` (4 occurrences)

**Excluded** (acceptable `_ =`):
- `defer res.Body.Close()` — cleanup
- `_, _ = io.Copy(io.Discard, res.Body)` — drain
- `_, _ = client.CreateMessage(...)` in bot replies — best-effort

#### 2. Remove Nil-Receiver Checks

Remove `if r == nil { return ... }` from these methods on `SessionRuntime`:
- `Session()`, `Engine()`, `Provider()`, `WithContext()`, `CloudConfig()`, `Client()`, `ConnectedClient()`, `RequireCapability()`

Remove from `RuntimeResolver`:
- `SetProvider()`

Keep nil checks on:
- `Resolve()` — guards against misconfiguration
- `resolveCapability()` — delegates to `Resolve()`

#### 3. Deduplicate `Support()` Methods

Four identical methods on `MessageRuntime`, `MediaRuntime`, `StatusRuntime`, `ProfileRuntime`. Since they all embed `*SessionRuntime` and have a `support` field, the simplest approach is to keep the methods but remove the nil checks (reducing each from 4 lines to 2 lines).

### Migration

1. Replace `_ =` patterns file by file, starting with `consumer.go`
2. Remove nil-receiver checks from `session_runtime.go`
3. Verify `go build ./...` and `go test ./...`
