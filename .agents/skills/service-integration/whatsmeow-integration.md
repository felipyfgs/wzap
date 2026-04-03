# whatsmeow Integration — wzap

Reference for using `wa.Manager` and `whatsmeow.Client` from service layer code.

## Obtaining a client

Every service method that interacts with WhatsApp starts the same way:

```go
client, err := s.engine.GetClient(sessionID)
if err != nil {
    return zero, err
}
if !client.IsConnected() {
    return zero, fmt.Errorf("client not connected")
}
```

Some operations also need `client.Store.ID != nil` (logged in check).

## Session lifecycle states

| Status | Meaning |
|---|---|
| `"disconnected"` | Session exists in DB, no active WA connection |
| `"connecting"` | `engine.Connect()` called, handshake/QR in progress |
| `"connected"` | Paired and fully connected, `jid` populated |

## Manager API

| Method | Returns | Purpose |
|---|---|---|
| `GetClient(sessionID)` | `(*whatsmeow.Client, error)` | Get active client from in-memory map |
| `Connect(ctx, sessionID)` | `(*Client, <-chan QRChannelItem, error)` | Create device + connect or start QR pairing |
| `Disconnect(sessionID)` | `error` | Disconnect without removing device |
| `Logout(ctx, sessionID)` | `error` | Unpair + delete device from sqlstore + clear DB |
| `GetQRCode(ctx, sessionID)` | `(string, error)` | Current QR code string |
| `ReconnectAll(ctx)` | `error` | Reconnect all paired devices on startup; cleans orphan devices |

## JID parsing

```go
// Smart parser — handles bare phone numbers and full JIDs
jid, err := parseJID(target)  // unexported, in service/message.go

// For phone number inputs in group/community services:
jid, err := types.ParseJID(wa.EnsureJIDSuffix(phone))

// For known full JIDs:
jid, err := types.ParseJID(groupJID)
```

`wa.EnsureJIDSuffix` appends `@s.whatsapp.net` if no `@` present.

## Sending messages

```go
resp, err := client.SendMessage(ctx, jid, msg)
return resp.ID, nil  // WhatsApp message ID
```

**Text:**
```go
msg := &waE2E.Message{
    ExtendedTextMessage: &waE2E.ExtendedTextMessage{
        Text:        proto.String(req.Body),
        ContextInfo: buildContextInfo(req.ReplyTo),
    },
}
```

**Media (Image/Video/Document/Audio):**
```go
data, _ := base64.StdEncoding.DecodeString(req.Base64)
uploaded, _ := client.Upload(ctx, data, whatsmeow.MediaImage)  // or MediaVideo, MediaDocument, MediaAudio
```

**Poll:**
```go
msg := client.BuildPollCreation(name, options, selectableCount)
```

**Edit / Delete / Reaction:**
```go
msg, _ := client.BuildEdit(jid, originalMsgID, newMsg)
msg, _ := client.BuildRevoke(jid, *client.Store.ID, msgID)
msg, _ := client.BuildReaction(jid, *client.Store.ID, msgID, emoji)
```

**Read receipts and presence:**
```go
client.MarkRead(ctx, []types.MessageID{msgID}, time.Now(), chatJID, *client.Store.ID)
client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
```

## Protobuf construction

Use `proto.String()`, `proto.Bool()`, `proto.Uint64()` for proto fields:
```go
msg := &waE2E.Message{
    Conversation: proto.String(text),
}
```

## App state patches (Chat/Label services)

```go
patch := appstate.BuildArchive(jid, true, time.Now(), nil)
client.SendAppState(ctx, patch)
```

## Event types

All constants in `internal/model/events.go`. Categories:

| Category | Examples |
|---|---|
| Messages | `EventMessage`, `EventUndecryptableMessage`, `EventReceipt`, `EventDeleteForMe` |
| Connection | `EventConnected`, `EventDisconnected`, `EventPairSuccess`, `EventLoggedOut` |
| Contacts | `EventContact`, `EventPicture`, `EventPushName` |
| Groups | `EventGroupInfo`, `EventJoinedGroup` |
| Presence | `EventPresence`, `EventChatPresence` |
| Chat state | `EventArchive`, `EventMute`, `EventPin`, `EventMarkChatAsRead` |
| Labels | `EventLabelEdit`, `EventLabelAssociationChat` |
| Calls | `EventCallOffer`, `EventCallTerminate`, `EventUnknownCallEvent` |
| Newsletter | `EventNewsletterJoin`, `EventNewsletterLiveUpdate` |
| Sync | `EventHistorySync`, `EventAppStateSyncComplete` |
| Privacy | `EventPrivacySettings`, `EventBlocklistChange` |
| Wildcard | `EventAll` — matches any event |

**Adding a new event type:**
1. Add constant to `internal/model/events.go` and the `ValidEventTypes` slice.
2. Handle it in `internal/wa/events.go` in the `handleEvent` type switch.

## Dispatcher / webhook fan-out

```go
m.dispatcher.Dispatch(sessionID, eventType, payloadBytes)
```

Signature: `func (d *Dispatcher) Dispatch(sessionID string, eventType model.EventType, payload []byte)` — no `context.Context`.

The dispatcher reads active webhooks via `WebhookRepository.FindActiveBySessionAndEvent` and delivers HTTP POST + publishes to NATS `wzap.events.<sessionID>`.

## Callback pattern

Set from services via setters (breaks dependency cycle):
```go
engine.SetMediaAutoUpload(mediaSvc.AutoUploadMedia)
engine.SetMessagePersist(historySvc.PersistMessage)
```

Called synchronously in `handleEvent`; actual work runs in a goroutine inside the service.
