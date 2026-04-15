# Spec: Chatwoot Package DRY

## Capability: Reduce Duplication in Chatwoot Integration

### Purpose

Eliminate repetitive patterns in the Chatwoot integration package through targeted helper extraction.

### Changes

#### 1. `@lid` Resolution Helper

**New method** on `Service` in `jid.go`:

```go
func (s *Service) resolveLID(ctx context.Context, sessionID, jid string, altJIDs ...string) string
```

- Returns `jid` unchanged if not a `@lid` suffix
- Tries `altJIDs` in order (skipping empty or `@lid` suffixed ones)
- Appends `@s.whatsapp.net` if resolved JID has no `@`
- Falls back to `s.resolveJID(ctx, sessionID, jid)`

**Call sites replaced**: 10+ in `inbound_events.go` and `inbound_message.go`

#### 2. Message Params Builder

**New struct** in `inbound_message.go`:

```go
type cwMsgParams struct {
    MessageType   string
    SourceID      string
    ContentAttrs  map[string]any
    SourceReplyID int
}

func newCWMsgParams(fromMe bool, msgID, stanzaID string, cwReplyID int) cwMsgParams
```

**Call sites replaced**: 6+ handler methods in `inbound_message.go`

#### 3. Idempotency Check

**New method** on `Service` in `outbound.go`:

```go
func (s *Service) isOutboundDuplicate(ctx context.Context, sessionID string, msg *dto.ChatwootWebhookMessage) bool
```

Extracts the sourceID check + CW msg ID cache check from `HandleIncomingWebhook`.

**Call site**: Single replacement in `HandleIncomingWebhook`.

### Migration

Each extraction is a standalone change:
1. Add the new helper
2. Replace call sites one by one
3. Verify each replacement with `go build ./...`
4. Run `go test ./internal/integrations/chatwoot/...`
