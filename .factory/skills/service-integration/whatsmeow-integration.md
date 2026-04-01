# whatsmeow Integration — wzap

Reference for using the `wa.Manager` and the underlying `whatsmeow.Client` from service layer code.

---

## Obtaining a client

Every service method that interacts with WhatsApp starts the same way:

```go
client, err := s.engine.GetClient(sessionID)
if err != nil {
    return "", err  // session not found or not initialised
}
if !client.IsConnected() {
    return "", fmt.Errorf("client not connected")
}
```

`engine` is `*wa.Manager`, injected into the service constructor.

---

## Session lifecycle states

| Status string | Meaning |
|---|---|
| `"disconnected"` | Session exists in DB, no active WA connection |
| `"connecting"` | `engine.Connect()` called, handshake in progress |
| `"pairing"` | QR code generated, waiting for phone scan |
| `"connected"` | Paired and fully connected, JID populated |

The status is stored in `"wzSessions"."status"` and updated by the event handlers in `internal/wa/events.go`.

---

## JID parsing

```go
import "go.mau.fi/whatsmeow/types"

// Full JID (e.g. "5511999999999@s.whatsapp.net")
jid, err := types.ParseJID(target)

// Phone number only — construct a user JID
jid = types.NewJID(phoneNumber, types.DefaultUserServer)

// Group JID suffix
// types.GroupServer = "g.us"
```

The helper `parseJID` in `internal/service/message.go` handles both cases:

```go
func parseJID(target string) (types.JID, error) {
    jid, err := types.ParseJID(target)
    if err != nil {
        if !strings.Contains(target, "@") {
            return types.NewJID(target, types.DefaultUserServer), nil
        }
        return types.JID{}, fmt.Errorf("invalid JID: %w", err)
    }
    return jid, nil
}
```

---

## Sending messages

```go
resp, err := client.SendMessage(ctx, jid, msg)
// resp.ID is the WhatsApp message ID string
```

**Text:**
```go
msg := &waProto.Message{Conversation: proto.String(text)}
```

**Image / Video / Document / Audio:**
```go
uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage) // or MediaVideo, MediaDocument, MediaAudio
```

**Poll:**
```go
msg := client.BuildPollCreation(name, options, selectableCount)
```

**Edit / Delete / Reaction:**
```go
msg := client.BuildEdit(jid, originalMsgID, newMsg)
msg := client.BuildRevoke(jid, *client.Store.ID, msgID)
msg := client.BuildReaction(jid, *client.Store.ID, msgID, emoji)
```

---

## Media upload

```go
import "github.com/gofiber/fiber/v2"
import "go.mau.fi/whatsmeow"

data, err := base64.StdEncoding.DecodeString(req.Base64)
uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
// uploaded.URL, uploaded.DirectPath, uploaded.MediaKey,
// uploaded.FileEncSHA256, uploaded.FileSHA256 — use all in the proto message
```

---

## Read receipts and presence

```go
// Mark as read
client.MarkRead(ctx, []types.MessageID{msgID}, time.Now(), jid, *client.Store.ID)

// Typing indicator
client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
```

---

## Event types

All event type constants are defined in `internal/model/events.go`. The full set (50+):

| Category | Constants |
|---|---|
| Messages | `EventMessage`, `EventUndecryptableMessage`, `EventMediaRetry`, `EventReceipt`, `EventDeleteForMe` |
| Connection | `EventConnected`, `EventDisconnected`, `EventConnectFailure`, `EventLoggedOut`, `EventPairSuccess`, `EventPairError`, `EventQR`, `EventStreamError`, `EventStreamReplaced`, `EventKeepAliveTimeout`, `EventClientOutdated`, `EventTemporaryBan` |
| Contacts | `EventContact`, `EventPicture`, `EventIdentityChange`, `EventUserAbout`, `EventPushName`, `EventBusinessName` |
| Groups | `EventGroupInfo`, `EventJoinedGroup` |
| Presence | `EventPresence`, `EventChatPresence` |
| Chat state | `EventArchive`, `EventMute`, `EventPin`, `EventStar`, `EventClearChat`, `EventDeleteChat`, `EventMarkChatAsRead` |
| Labels | `EventLabelEdit`, `EventLabelAssociationChat`, `EventLabelAssociationMessage` |
| Calls | `EventCallOffer`, `EventCallAccept`, `EventCallTerminate`, `EventCallOfferNotice`, `EventCallReject` |
| Newsletter | `EventNewsletterJoin`, `EventNewsletterLeave`, `EventNewsletterMuteChange`, `EventNewsletterLiveUpdate` |
| Sync | `EventHistorySync`, `EventAppState`, `EventAppStateSyncComplete`, `EventOfflineSyncCompleted` |
| Privacy | `EventPrivacySettings`, `EventPushNameSetting`, `EventBlocklistChange`, `EventBlocklist` |
| Wildcard | `EventAll` — matches any event in webhook subscriptions |

**Adding a new event type:**
1. Add the constant to `internal/model/events.go`.
2. Add it to the `types` slice inside the `ValidEventTypes` init function.
3. Handle it in `internal/wa/events.go` within the `handleEvent` switch.
4. Dispatch via `s.dispatcher.Dispatch(ctx, sessionID, string(eventType), payload)`.

---

## Dispatcher / webhook fan-out

```go
// internal/dispatcher/dispatcher.go
// Publish an event to all matching webhooks and NATS:
disp.Dispatch(ctx, sessionID, "Message", payloadBytes)
```

The dispatcher reads active webhooks via `WebhookRepository.FindActiveBySessionAndEvent` and delivers HTTP POST + publishes to the NATS subject `wzap.events.<sessionID>.<eventType>`.
