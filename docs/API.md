# wzap API Reference

wzap is a multi-session WhatsApp REST API built on [whatsmeow](https://github.com/tulir/whatsmeow).

---

## Base URL

```
http://<host>:<port>
```

Default port: `3000` (configurable via `PORT` env var).

---

## Authentication

All endpoints (except `/health`) require an `Authorization` header:

```
Authorization: Bearer <token>
```

Two token types are accepted:

| Type | Value | Access |
|---|---|---|
| **Admin** | `API_KEY` env var | Full access — session management + all session routes |
| **Session** | Session `apiKey` | Scoped to that session only |

---

## Response Format

### Success
```json
{
  "success": true,
  "data": { ... },
  "message": "Human-readable message"
}
```

### Error
```json
{
  "success": false,
  "error": "Error type",
  "details": "Details about the error"
}
```

---

## Health

### Health Check

`GET /health` — No authentication required.

```bash
curl http://localhost:3000/health
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "UP",
    "services": {
      "database": true,
      "nats": true,
      "minio": true
    }
  },
  "message": "wzap is running"
}
```

---

## Sessions

Session-scoped routes use the pattern `/sessions/:sessionName/...` where `:sessionName` is the unique name given at creation.

### Create Session *(Admin only)*

`POST /sessions`

```bash
curl -X POST http://localhost:3000/sessions \
  -H 'Authorization: Bearer ADMIN_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"name": "my-session", "metadata": {"owner": "john"}}'
```

**Body:**
```json
{
  "name": "my-session",
  "apiKey": "optional-custom-key",
  "metadata": {}
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "my-session",
    "apiKey": "generated-or-custom-key",
    "status": "disconnected",
    "connected": 0,
    "createdAt": "2026-03-29T00:00:00Z",
    "updatedAt": "2026-03-29T00:00:00Z"
  },
  "message": "Session created"
}
```

---

### List Sessions *(Admin only)*

`GET /sessions`

```bash
curl http://localhost:3000/sessions \
  -H 'Authorization: Bearer ADMIN_TOKEN'
```

---

### Get Session

`GET /sessions/:sessionName`

```bash
curl http://localhost:3000/sessions/my-session \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

### Delete Session

`DELETE /sessions/:sessionName`

```bash
curl -X DELETE http://localhost:3000/sessions/my-session \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

### Connect Session

`POST /sessions/:sessionName/connect`

Connects the session. If the device is not yet paired, returns status `PAIRING` and begins generating QR codes (poll `/qr` to retrieve them).

```bash
curl -X POST http://localhost:3000/sessions/my-session/connect \
  -H 'Authorization: Bearer SESSION_KEY'
```

**Response:**
```json
{
  "success": true,
  "data": { "status": "PAIRING" },
  "message": "Connection initiated"
}
```

Status values: `PAIRING` | `CONNECTING` | `CONNECTED`

---

### Disconnect Session

`POST /sessions/:sessionName/disconnect`

```bash
curl -X POST http://localhost:3000/sessions/my-session/disconnect \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

### Get QR Code

`GET /sessions/:sessionName/qr`

Call `/connect` first, then poll this endpoint until the QR is available. Returns the raw QR string and a base64 PNG image.

```bash
curl http://localhost:3000/sessions/my-session/qr \
  -H 'Authorization: Bearer SESSION_KEY'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "qr": "2@raw-qr-string...",
    "image": "data:image/png;base64,iVBORw0K..."
  },
  "message": "QR Code retrieved"
}
```

---

## Messages

All message endpoints: `POST /sessions/:sessionName/messages/<type>`

The `jid` field accepts either a **phone number** (e.g. `5511999999999`) or a **full JID** (e.g. `5511999999999@s.whatsapp.net` or `120362023605733675@g.us`).

---

### Send Text

`POST /sessions/:sessionName/messages/text`

```bash
curl -X POST http://localhost:3000/sessions/my-session/messages/text \
  -H 'Authorization: Bearer SESSION_KEY' \
  -H 'Content-Type: application/json' \
  -d '{"jid": "5511999999999", "text": "Hello!"}'
```

**Body:**
```json
{
  "jid": "5511999999999",
  "text": "Hello!"
}
```

**Response:**
```json
{
  "success": true,
  "data": { "messageId": "ABCDEF123456" },
  "message": "Text message sent"
}
```

---

### Send Image / Video / Document / Audio

`POST /sessions/:sessionName/messages/image`
`POST /sessions/:sessionName/messages/video`
`POST /sessions/:sessionName/messages/document`
`POST /sessions/:sessionName/messages/audio`

```bash
curl -X POST http://localhost:3000/sessions/my-session/messages/image \
  -H 'Authorization: Bearer SESSION_KEY' \
  -H 'Content-Type: application/json' \
  -d '{"jid":"5511999999999","mimeType":"image/jpeg","base64":"<base64>","caption":"Look at this"}'
```

**Body:**
```json
{
  "jid": "5511999999999",
  "mimeType": "image/jpeg",
  "base64": "<base64-encoded-file>",
  "caption": "Optional caption",
  "filename": "optional-filename.jpg"
}
```

---

### Send Sticker

`POST /sessions/:sessionName/messages/sticker`

```json
{
  "jid": "5511999999999",
  "mimeType": "image/webp",
  "base64": "<base64-encoded-webp>"
}
```

---

### Send Contact

`POST /sessions/:sessionName/messages/contact`

```json
{
  "jid": "5511999999999",
  "name": "João Silva",
  "vcard": "BEGIN:VCARD\nVERSION:3.0\nFN:João Silva\nTEL:+5511999999999\nEND:VCARD"
}
```

---

### Send Location

`POST /sessions/:sessionName/messages/location`

```json
{
  "jid": "5511999999999",
  "lat": -23.5505,
  "lng": -46.6333,
  "name": "São Paulo",
  "address": "Av. Paulista, 1000"
}
```

---

### Send Poll

`POST /sessions/:sessionName/messages/poll`

```json
{
  "jid": "5511999999999",
  "name": "Favorite color?",
  "options": ["Red", "Green", "Blue"],
  "selectableCount": 1
}
```

`selectableCount`: how many options a recipient may choose. `0` = unlimited.

---

### Send Link Preview

`POST /sessions/:sessionName/messages/link`

```json
{
  "jid": "5511999999999",
  "url": "https://example.com",
  "title": "Example Site",
  "description": "An example website"
}
```

---

### Edit Message

`POST /sessions/:sessionName/messages/edit`

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456",
  "text": "Updated text content"
}
```

---

### Delete Message

`POST /sessions/:sessionName/messages/delete`

Revokes the message for all recipients.

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456"
}
```

---

### React to Message

`POST /sessions/:sessionName/messages/reaction`

Pass an empty `reaction` string to remove an existing reaction.

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456",
  "reaction": "👍"
}
```

---

### Mark Message as Read

`POST /sessions/:sessionName/messages/read`

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456"
}
```

---

### Set Typing / Recording Presence

`POST /sessions/:sessionName/messages/presence`

```json
{
  "jid": "5511999999999",
  "presence": "typing"
}
```

`presence` values: `typing` | `recording` | `paused`

---

## Contacts

### List Contacts

`GET /sessions/:sessionName/contacts`

```bash
curl http://localhost:3000/sessions/my-session/contacts \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

### Check Contacts on WhatsApp

`POST /sessions/:sessionName/contacts/check`

```json
{
  "phones": ["5511999999999", "5511888888888"]
}
```

**Response:**
```json
{
  "success": true,
  "data": [
    { "exists": true, "jid": "5511999999999@s.whatsapp.net", "phoneNumber": "5511999999999" },
    { "exists": false, "phoneNumber": "5511888888888" }
  ]
}
```

---

### Get Contact Avatar

`POST /sessions/:sessionName/contacts/avatar`

```json
{
  "jid": "5511999999999@s.whatsapp.net"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "url": "https://pps.whatsapp.net/v/...",
    "id": "123456789"
  }
}
```

---

### Block Contact

`POST /sessions/:sessionName/contacts/block`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Unblock Contact

`POST /sessions/:sessionName/contacts/unblock`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Get Blocklist

`GET /sessions/:sessionName/contacts/blocklist`

---

### Get User Info

`POST /sessions/:sessionName/contacts/info`

```json
{
  "jids": ["5511999999999@s.whatsapp.net"]
}
```

---

### Get Privacy Settings

`GET /sessions/:sessionName/contacts/privacy`

---

### Set Profile Picture

`POST /sessions/:sessionName/contacts/profile-picture`

```json
{
  "base64": "<base64-encoded-jpeg>"
}
```

---

## Groups

### List Groups

`GET /sessions/:sessionName/groups`

---

### Create Group

`POST /sessions/:sessionName/groups/create`

```json
{
  "name": "My Group",
  "participants": ["5511999999999", "5511888888888"]
}
```

---

### Get Group Info

`POST /sessions/:sessionName/groups/info`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Get Group Info from Invite Link

`POST /sessions/:sessionName/groups/invite-info`

```json
{ "inviteCode": "HffXhYmzzyJGec61oqMXiz" }
```

---

### Join Group with Invite Link

`POST /sessions/:sessionName/groups/join`

```json
{ "inviteCode": "HffXhYmzzyJGec61oqMXiz" }
```

**Response:**
```json
{
  "success": true,
  "data": { "jid": "120362023605733675@g.us" }
}
```

---

### Get Invite Link

`POST /sessions/:sessionName/groups/invite-link`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Leave Group

`POST /sessions/:sessionName/groups/leave`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Update Participants

`POST /sessions/:sessionName/groups/participants`

```json
{
  "groupJid": "120362023605733675@g.us",
  "participants": ["5511999999999"],
  "action": "add"
}
```

`action` values: `add` | `remove` | `promote` | `demote`

---

### Get Join Requests

`POST /sessions/:sessionName/groups/requests`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Approve / Reject Join Requests

`POST /sessions/:sessionName/groups/requests/action`

```json
{
  "groupJid": "120362023605733675@g.us",
  "participants": ["5511999999999"],
  "action": "approve"
}
```

`action` values: `approve` | `reject`

---

### Update Group Name

`POST /sessions/:sessionName/groups/name`

```json
{
  "groupJid": "120362023605733675@g.us",
  "text": "New Group Name"
}
```

---

### Update Group Description

`POST /sessions/:sessionName/groups/description`

```json
{
  "groupJid": "120362023605733675@g.us",
  "text": "New description"
}
```

---

### Update Group Photo

`POST /sessions/:sessionName/groups/photo`

```json
{
  "groupJid": "120362023605733675@g.us",
  "photoBase64": "<base64-encoded-jpeg>"
}
```

---

### Set Announce Mode

`POST /sessions/:sessionName/groups/announce`

Only admins can send messages when enabled.

```json
{
  "groupJid": "120362023605733675@g.us",
  "enabled": true
}
```

---

### Set Locked Mode

`POST /sessions/:sessionName/groups/locked`

Only admins can edit group info when enabled.

```json
{
  "groupJid": "120362023605733675@g.us",
  "enabled": true
}
```

---

### Set Join Approval

`POST /sessions/:sessionName/groups/join-approval`

Requires admin approval for new members.

```json
{
  "groupJid": "120362023605733675@g.us",
  "enabled": true
}
```

---

## Chat

### Archive Chat

`POST /sessions/:sessionName/chat/archive`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Mute Chat

`POST /sessions/:sessionName/chat/mute`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Pin Chat

`POST /sessions/:sessionName/chat/pin`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Unpin Chat

`POST /sessions/:sessionName/chat/unpin`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

## Labels

Labels are a WhatsApp Business feature for organizing chats.

### Add Label to Chat

`POST /sessions/:sessionName/label/chat`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1"
}
```

---

### Remove Label from Chat

`POST /sessions/:sessionName/unlabel/chat`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1"
}
```

---

### Add Label to Message

`POST /sessions/:sessionName/label/message`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1",
  "messageId": "ABCDEF123456"
}
```

---

### Remove Label from Message

`POST /sessions/:sessionName/unlabel/message`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1",
  "messageId": "ABCDEF123456"
}
```

---

### Edit Label

`POST /sessions/:sessionName/label/edit`

```json
{
  "labelId": "1",
  "name": "New Label Name",
  "color": 1
}
```

---

## Newsletter (WhatsApp Channels)

### Create Newsletter

`POST /sessions/:sessionName/newsletter/create`

```json
{
  "name": "My Channel",
  "description": "Channel description",
  "picture": "<base64-encoded-image>"
}
```

---

### Get Newsletter Info

`POST /sessions/:sessionName/newsletter/info?jid=<newsletterJid>`

```bash
curl "http://localhost:3000/sessions/my-session/newsletter/info?jid=120363166361227321@newsletter" \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

### Get Newsletter Info from Invite

`POST /sessions/:sessionName/newsletter/invite?code=<inviteCode>`

---

### List Subscribed Newsletters

`GET /sessions/:sessionName/newsletter/list`

---

### Get Newsletter Messages

`POST /sessions/:sessionName/newsletter/messages`

```json
{
  "newsletterJid": "120363166361227321@newsletter",
  "count": 20,
  "beforeId": 0
}
```

`beforeId`: server message ID for pagination cursor. `0` = latest messages.

---

### Subscribe to Newsletter

`POST /sessions/:sessionName/newsletter/subscribe`

```json
{ "newsletterJid": "120363166361227321@newsletter" }
```

---

## Community

Communities are groups of groups in WhatsApp.

### Create Community

`POST /sessions/:sessionName/community/create`

```json
{
  "name": "My Community",
  "description": "Community description"
}
```

---

### Add Subgroup to Community

`POST /sessions/:sessionName/community/participant/add`

```json
{
  "communityJid": "120363166361227321@g.us",
  "participants": ["120362023605733675@g.us"]
}
```

---

### Remove Subgroup from Community

`POST /sessions/:sessionName/community/participant/remove`

```json
{
  "communityJid": "120363166361227321@g.us",
  "participants": ["120362023605733675@g.us"]
}
```

---

## Webhooks

Webhooks receive real-time event notifications for a session.

### Create Webhook

`POST /sessions/:sessionName/webhooks`

```json
{
  "url": "https://your-server.com/webhook",
  "secret": "optional-signing-secret",
  "events": ["Message", "Connected", "Disconnected"]
}
```

See [Supported Event Types](#supported-event-types) for all valid `events` values.

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "sessionId": "session-uuid",
    "url": "https://your-server.com/webhook",
    "events": ["Message", "Connected", "Disconnected"],
    "enabled": true,
    "createdAt": "2026-03-29T00:00:00Z"
  }
}
```

---

### List Webhooks

`GET /sessions/:sessionName/webhooks`

---

### Delete Webhook

`DELETE /sessions/:sessionName/webhooks/:wid`

```bash
curl -X DELETE http://localhost:3000/sessions/my-session/webhooks/webhook-uuid \
  -H 'Authorization: Bearer SESSION_KEY'
```

---

## Webhook Payload

Events are published to NATS (`wzap.events.<sessionId>`) and delivered to registered webhook URLs.

Payload fields at the top level:

| Field | Type | Description |
|---|---|---|
| `eventId` | string | Unique UUID for this event |
| `sessionId` | string | Session that emitted the event |
| `event` | string | Event type name (see below) |
| `timestamp` | string | ISO 8601 timestamp |
| _...native fields_ | any | All whatsmeow event fields merged at top level |

**Example — Message event:**
```json
{
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "sessionId": "my-session-uuid",
  "event": "Message",
  "timestamp": "2026-03-29T16:00:00+02:00",
  "id": "ABCDEF123456",
  "pushName": "João",
  "fromMe": false,
  "message": { "conversation": "Hello!" }
}
```

---

## Supported Event Types

Use these values in the `events` field when creating webhooks. Use `"All"` to subscribe to every event.

| Category | Event |
|---|---|
| **Special** | `All` |
| **Messages** | `Message`, `UndecryptableMessage`, `MediaRetry`, `Receipt`, `DeleteForMe` |
| **Connection** | `Connected`, `Disconnected`, `ConnectFailure`, `LoggedOut`, `PairSuccess`, `PairError`, `QR`, `QRScannedWithoutMultidevice`, `StreamError`, `StreamReplaced`, `KeepAliveTimeout`, `KeepAliveRestored`, `ClientOutdated`, `TemporaryBan`, `CATRefreshError`, `ManualLoginReconnect` |
| **Contacts** | `Contact`, `Picture`, `IdentityChange`, `UserAbout`, `PushName`, `BusinessName` |
| **Groups** | `GroupInfo`, `JoinedGroup` |
| **Presence** | `Presence`, `ChatPresence` |
| **Chat State** | `Archive`, `Mute`, `Pin`, `Star`, `ClearChat`, `DeleteChat`, `MarkChatAsRead`, `UnarchiveChatsSetting` |
| **Labels** | `LabelEdit`, `LabelAssociationChat`, `LabelAssociationMessage` |
| **Calls** | `CallOffer`, `CallAccept`, `CallTerminate`, `CallOfferNotice`, `CallRelayLatency`, `CallPreAccept`, `CallReject`, `CallTransport`, `UnknownCallEvent` |
| **Newsletter** | `NewsletterJoin`, `NewsletterLeave`, `NewsletterMuteChange`, `NewsletterLiveUpdate` |
| **Sync** | `HistorySync`, `AppState`, `AppStateSyncComplete`, `AppStateSyncError`, `OfflineSyncCompleted`, `OfflineSyncPreview` |
| **Privacy** | `PrivacySettings`, `PushNameSetting`, `UserStatusMute`, `BlocklistChange`, `Blocklist` |
