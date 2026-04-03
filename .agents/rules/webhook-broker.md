# Webhook, Broker, and WebSocket Rules

## NATS Broker (internal/broker/nats.go)

```go
type NATS struct {
    Conn *nats.Conn
    JS   jetstream.JetStream
}
```

Two JetStream streams created eagerly at startup:

| Stream | Subject | Storage | Max Age |
|---|---|---|---|
| `WZAP_EVENTS` | `wzap.events.>` | File | 7 days |
| `WZAP_WEBHOOKS` | `wzap.webhook.>` | File | 24 hours |

- Connection options: `RetryOnFailedConnect(true)`, `MaxReconnects(10)`, `ReconnectWait(1s)`.
- Shutdown: `Drain()` then `Close()`.
- Publishing: `n.JS.Publish(ctx, subject, data)`.

## Webhook Dispatcher (internal/webhook/dispatcher.go)

### Three-Channel Fan-Out

When `Dispatch(sessionID, eventType, payload)` is called:

1. **Per-session webhooks** — looked up from DB via `FindActiveBySessionAndEvent`. Each webhook goes to NATS queue (if `NATSEnabled`) or direct HTTP.
2. **Global webhook** — if `globalWebhookURL` is set and circuit breaker allows.
3. **WebSocket broadcast** — always, if WS hub is set.

### Payload Size Guard

Events exceeding 512KB are not dispatched to webhooks — only broadcast via WebSocket.

### HMAC-SHA256 Signing

When webhook has a secret:
```go
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(payload)
req.Header.Set("X-Wzap-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
req.Header.Set("X-Wzap-Event", eventType)
```

### HTTP Delivery with Retry

- Max 3 attempts with exponential backoff: 2s, 4s, 8s.
- 4xx responses (except 429) are permanent errors — NOT retried.
- NATS publish failure falls back to direct HTTP.

### Global Webhook Circuit Breaker

- Atomic counters (`globalFailures`, `globalLastAttempt`) for lock-free access.
- Backoff: 5s → 10s → 20s → ... → max 30 minutes (exponential, capped).
- On success: reset failure counter to 0.

### NATS Consumer (`StartConsumer`)

JetStream ephemeral consumer on `WZAP_WEBHOOKS`:
- Ack policy: Explicit (manual `Ack`/`Nak`/`Term`).
- Max deliveries: 5.
- Backoff: `[10s, 30s, 1m, 5m]`.
- Ack wait: 15s.
- Permanent error (4xx) → `msg.Term()`. Transient error → `msg.Nak()`.

### WSBroadcaster Interface

```go
type WSBroadcaster interface {
    Broadcast(sessionID string, payload []byte)
}
```

Set via `SetWSBroadcaster(hub)` to avoid circular dependencies.

## WebSocket Hub (internal/websocket/hub.go)

```go
type Hub struct {
    mu          sync.RWMutex
    connections map[string]map[*ws.Conn]struct{}  // sessionID → set of conns
}
```

- `Broadcast(sessionID, payload)` — read lock to copy conn set, write without lock.
- `BroadcastAll(payload)` — flattens all sessions' connections.
- Auto-cleanup: write failure → `Unregister` + `Close`.
- Unregister cleans up empty session sets.
- Wildcard session `"*"` for global listeners (connected via `/ws` without `:sessionId`).

## WebSocket Handler (internal/handler/websocket.go)

Two-phase setup:
1. **`Upgrade()`** — validates token from query param `?token=` or `Authorization` header against `cfg.AdminToken`.
2. **`Handle()`** — reads `c.Params("sessionId", "*")`, registers with hub, blocks on `ReadMessage` loop.

## Constants

| Name | Value | Location |
|---|---|---|
| `maxNATSPayloadSize` | 512 KB | `wa/events.go` |
| `maxWebhookPayload` | 512 KB | `webhook/dispatcher.go` |
| `maxDeliverAttempts` | 5 | `webhook/dispatcher.go` |
| `maxHTTPRetries` | 3 | `webhook/dispatcher.go` |
| `httpRetryBaseDelay` | 2s | `webhook/dispatcher.go` |
| `globalBackoffBase` | 5s | `webhook/dispatcher.go` |
| `globalBackoffMax` | 30m | `webhook/dispatcher.go` |
| `httpTimeout` | 10s | `webhook/dispatcher.go` |
