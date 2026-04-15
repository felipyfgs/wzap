# Spec: wa/events.go Decomposition

## Capability: Split Monolithic Event Handler

### Purpose

Decompose the ~530-line `Manager.handleEvent` method into three focused functions and eliminate the two-pass type switch.

### Interface

```go
// classifiedEvent holds the result of classifying a whatsmeow event.
type classifiedEvent struct {
    Type model.EventType
    Data map[string]any  // pre-built data for special events; nil for default serialization
}

// classifyEvent determines the event type and optionally pre-builds data.
// Returns a zero-value classifiedEvent for events that should be skipped.
func classifyEvent(evt any) classifiedEvent

// serializeEventData builds the data map for NATS/webhook dispatch.
// If classifiedEvent.Data is non-nil, it is returned directly.
// Otherwise, the event is JSON-encoded/decoded to a map.
func serializeEventData(evt any, ce classifiedEvent) map[string]any

// dispatchEvent publishes to NATS and dispatches via webhook.
func (m *Manager) dispatchEvent(sessionID, sessionName string, eventType model.EventType, data map[string]any)
```

### Refactored `handleEvent`

```go
func (m *Manager) handleEvent(sessionID string, evt any) {
    ce := classifyEvent(evt)
    if ce.Type == "" {
        return
    }
    data := serializeEventData(evt, ce)
    nameCtx, nameCancel := context.WithTimeout(context.Background(), 3*time.Second)
    sessionName := m.getSessionName(nameCtx, sessionID)
    nameCancel()
    m.dispatchEvent(sessionID, sessionName, ce.Type, data)
}
```

### Single-Pass Optimization

For `HistorySync` and `AppState` events, `classifyEvent` populates `Data` inline, avoiding the second type switch in `serializeEventData`. For all other events, `serializeEventData` uses the existing JSON encode/decode path with the special-case cleanup (deleting `RawMessage`, `SourceWebMsg`, adding `Error` fields).

### Migration

1. Create `classifyEvent` as a standalone function (move type switch from `handleEvent`)
2. Create `serializeEventData` (move second type switch + JSON path)
3. Create `dispatchEvent` (move NATS publish + webhook dispatch)
4. Rewrite `handleEvent` as the 3-call composition
5. Verify `go build ./...` and `go test ./internal/wa/...`

### Constraints

- Event dispatch order must remain sequential (NATS first, then webhook)
- No behavioral changes — same events, same data, same logging
