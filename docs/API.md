# wzap API Reference

wzap is a multi-session WhatsApp REST API built on [whatsmeow](https://github.com/tulir/whatsmeow).

---

## Table of Contents

- [Base URL](#base-url)
- [Authentication](#authentication)
- [Response Format](#response-format)
- [Health](#health)
  - [Health Check](#health-check)
- [Sessions](#sessions)
  - [Create Session](#create-session-admin-only)
  - [List Sessions](#list-sessions-admin-only)
  - [Get Session](#get-session)
  - [Delete Session](#delete-session)
  - [Connect Session](#connect-session)
  - [Disconnect Session](#disconnect-session)
  - [Get QR Code](#get-qr-code)
- [Messages](#messages)
  - [Send Text](#send-text)
  - [Send Image / Video / Document / Audio](#send-image--video--document--audio)
  - [Send Sticker](#send-sticker)
  - [Send Contact](#send-contact)
  - [Send Location](#send-location)
  - [Send Poll](#send-poll)
  - [Send Link Preview](#send-link-preview)
  - [Edit Message](#edit-message)
  - [Delete Message](#delete-message)
  - [React to Message](#react-to-message)
  - [Mark Message as Read](#mark-message-as-read)
  - [Set Typing / Recording Presence](#set-typing--recording-presence)
- [Contacts](#contacts)
  - [List Contacts](#list-contacts)
  - [Check Contacts on WhatsApp](#check-contacts-on-whatsapp)
  - [Get Contact Avatar](#get-contact-avatar)
  - [Block Contact](#block-contact)
  - [Unblock Contact](#unblock-contact)
  - [Get Blocklist](#get-blocklist)
  - [Get User Info](#get-user-info)
  - [Get Privacy Settings](#get-privacy-settings)
  - [Set Profile Picture](#set-profile-picture)
- [Groups](#groups)
  - [List Groups](#list-groups)
  - [Create Group](#create-group)
  - [Get Group Info](#get-group-info)
  - [Get Group Info from Invite Link](#get-group-info-from-invite-link)
  - [Join Group with Invite Link](#join-group-with-invite-link)
  - [Get Invite Link](#get-invite-link)
  - [Leave Group](#leave-group)
  - [Update Participants](#update-participants)
  - [Get Join Requests](#get-join-requests)
  - [Approve / Reject Join Requests](#approve--reject-join-requests)
  - [Update Group Name](#update-group-name)
  - [Update Group Description](#update-group-description)
  - [Update Group Photo](#update-group-photo)
  - [Set Announce Mode](#set-announce-mode)
  - [Set Locked Mode](#set-locked-mode)
  - [Set Join Approval](#set-join-approval)
- [Chat](#chat)
  - [Archive Chat](#archive-chat)
  - [Mute Chat](#mute-chat)
  - [Pin Chat](#pin-chat)
  - [Unpin Chat](#unpin-chat)
- [Labels](#labels)
  - [Add Label to Chat](#add-label-to-chat)
  - [Remove Label from Chat](#remove-label-from-chat)
  - [Add Label to Message](#add-label-to-message)
  - [Remove Label from Message](#remove-label-from-message)
  - [Edit Label](#edit-label)
- [Newsletter (WhatsApp Channels)](#newsletter-whatsapp-channels)
  - [Create Newsletter](#create-newsletter)
  - [Get Newsletter Info](#get-newsletter-info)
  - [Get Newsletter Info from Invite](#get-newsletter-info-from-invite)
  - [List Subscribed Newsletters](#list-subscribed-newsletters)
  - [Get Newsletter Messages](#get-newsletter-messages)
  - [Subscribe to Newsletter](#subscribe-to-newsletter)
- [Community](#community)
  - [Create Community](#create-community)
  - [Add Subgroup to Community](#add-subgroup-to-community)
  - [Remove Subgroup from Community](#remove-subgroup-from-community)
- [Webhooks](#webhooks)
  - [Create Webhook](#create-webhook)
  - [List Webhooks](#list-webhooks)
  - [Delete Webhook](#delete-webhook)
- [Webhook Payload](#webhook-payload)
- [Supported Event Types](#supported-event-types)

---

## Base URL

```
http://<host>:<port>
```

Default port: `8080` (configurable via `PORT` env var).

---

## Authentication

All endpoints (except `/health`) require a `Token` header:

```
Token: <token>
```

Two token types are accepted:

| Type | Value | Access |
|---|---|---|
| **Admin** | `API_TOKEN` env var | Full access — session management + all session routes |
| **Session** | Session `token` | Scoped to that session only |

---

## Response Format

HTTP status codes are the source of truth (`2xx` = success, `4xx`/`5xx` = error). The `success` boolean mirrors the status code.

### Success
```json
{
  "success": true,
  "data": { ... },
  "message": "success"
}
```

### Error
```json
{
  "success": false,
  "error": "Error type",
  "message": "Details about the error"
}
```

---

## Health

### Health Check

`GET /health` — No authentication required.

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "data": {
    "status": "UP",
    "services": {
      "database": true,
      "nats": true,
      "minio": true
    }
  },
  "message": "success"
}
```

---

## Sessions

Session-scoped routes use the pattern `/sessions/:sessionId/...` where `:sessionId` is the unique name given at creation.

### Create Session *(Admin only)*

`POST /sessions`

```bash
curl -X POST http://localhost:8080/sessions \
  -H 'Token: ADMIN_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"name": "my-session"}'
```

**Body:**
```json
{
  "name": "my-session",
  "token": "optional-custom-token",
  "proxy": {
    "host": "proxy.example.com",
    "port": 3128,
    "protocol": "http",
    "username": "",
    "password": ""
  },
  "webhook": {
    "url": "https://my-server.com/hook",
    "events": ["Message", "Connected", "Disconnected"]
  },
  "settings": {
    "alwaysOnline": false,
    "rejectCall": false,
    "msgRejectCall": "",
    "readMessages": false,
    "ignoreGroups": false,
    "ignoreStatus": false
  }
}
```

> All fields except `name` are optional. `token` is auto-generated as `sk_<uuid>` if omitted. If `webhook.url` is provided, a webhook entry is automatically created for the session.

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "my-session",
    "token": "sk_generated-or-custom-token",
    "proxy": { "host": "", "port": 0, "protocol": "", "username": "", "password": "" },
    "status": "disconnected",
    "connected": 0,
    "settings": {
      "alwaysOnline": false,
      "rejectCall": false,
      "msgRejectCall": "",
      "readMessages": false,
      "ignoreGroups": false,
      "ignoreStatus": false
    },
    "createdAt": "2026-03-29T00:00:00Z",
    "updatedAt": "2026-03-29T00:00:00Z"
  },
  "message": "success"
}
```

---

### List Sessions *(Admin only)*

`GET /sessions`

```bash
curl http://localhost:8080/sessions \
  -H 'Token: ADMIN_TOKEN'
```

---

### Get Session

`GET /sessions/:sessionId`

> `:sessionId` accepts either the session **UUID** or the session **name**.

```bash
curl http://localhost:8080/sessions/my-session \
  -H 'Token: SESSION_KEY'
# or by UUID:
curl http://localhost:8080/sessions/550e8400-e29b-41d4-a716-446655440000 \
  -H 'Token: SESSION_KEY'
```

---

### Delete Session

`DELETE /sessions/:sessionId`

```bash
curl -X DELETE http://localhost:8080/sessions/my-session \
  -H 'Token: SESSION_KEY'
```

---

### Connect Session

`POST /sessions/:sessionId/connect`

Connects the session. If the device is not yet paired, returns status `PAIRING` and begins generating QR codes (poll `/qr` to retrieve them).

```bash
curl -X POST http://localhost:8080/sessions/my-session/connect \
  -H 'Token: SESSION_KEY'
```

**Response:**
```json
{
  "data": { "status": "PAIRING" },
  "message": "success"
}
```

Status values: `PAIRING` | `CONNECTING` | `CONNECTED`

---

### Disconnect Session

`POST /sessions/:sessionId/disconnect`

```bash
curl -X POST http://localhost:8080/sessions/my-session/disconnect \
  -H 'Token: SESSION_KEY'
```

---

### Get QR Code

`GET /sessions/:sessionId/qr`

Call `/connect` first, then poll this endpoint until the QR is available. Returns the raw QR string and a base64 PNG image.

```bash
curl http://localhost:8080/sessions/my-session/qr \
  -H 'Token: SESSION_KEY'
```

**Response:**
```json
{
  "data": {
    "qr": "2@raw-qr-string...",
    "image": "data:image/png;base64,iVBORw0K..."
  },
  "message": "success"
}
```

---

## Messages

All message endpoints: `POST /sessions/:sessionId/messages/<type>`

The `jid` field accepts either a **phone number** (e.g. `5511999999999`) or a **full JID** (e.g. `5511999999999@s.whatsapp.net` or `120362023605733675@g.us`).

---

### Send Text

`POST /sessions/:sessionId/messages/text`

```bash
curl -X POST http://localhost:8080/sessions/my-session/messages/text \
  -H 'Token: SESSION_KEY' \
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
  "data": { "messageId": "ABCDEF123456" },
  "message": "success"
}
```

---

### Send Image / Video / Document / Audio

`POST /sessions/:sessionId/messages/image`
`POST /sessions/:sessionId/messages/video`
`POST /sessions/:sessionId/messages/document`
`POST /sessions/:sessionId/messages/audio`

```bash
curl -X POST http://localhost:8080/sessions/my-session/messages/image \
  -H 'Token: SESSION_KEY' \
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

`POST /sessions/:sessionId/messages/sticker`

```json
{
  "jid": "5511999999999",
  "mimeType": "image/webp",
  "base64": "<base64-encoded-webp>"
}
```

---

### Send Contact

`POST /sessions/:sessionId/messages/contact`

```json
{
  "jid": "5511999999999",
  "name": "João Silva",
  "vcard": "BEGIN:VCARD\nVERSION:3.0\nFN:João Silva\nTEL:+5511999999999\nEND:VCARD"
}
```

---

### Send Location

`POST /sessions/:sessionId/messages/location`

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

`POST /sessions/:sessionId/messages/poll`

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

`POST /sessions/:sessionId/messages/link`

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

`POST /sessions/:sessionId/messages/edit`

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456",
  "text": "Updated text content"
}
```

---

### Delete Message

`POST /sessions/:sessionId/messages/delete`

Revokes the message for all recipients.

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456"
}
```

---

### React to Message

`POST /sessions/:sessionId/messages/reaction`

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

`POST /sessions/:sessionId/messages/read`

```json
{
  "jid": "5511999999999",
  "messageId": "ABCDEF123456"
}
```

---

### Set Typing / Recording Presence

`POST /sessions/:sessionId/messages/presence`

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

`GET /sessions/:sessionId/contacts`

```bash
curl http://localhost:8080/sessions/my-session/contacts \
  -H 'Token: SESSION_KEY'
```

---

### Check Contacts on WhatsApp

`POST /sessions/:sessionId/contacts/check`

```json
{
  "phones": ["5511999999999", "5511888888888"]
}
```

**Response:**
```json
{
  "data": [
    { "exists": true, "jid": "5511999999999@s.whatsapp.net", "phoneNumber": "5511999999999" },
    { "exists": false, "phoneNumber": "5511888888888" }
  ]
}
```

---

### Get Contact Avatar

`POST /sessions/:sessionId/contacts/avatar`

```json
{
  "jid": "5511999999999@s.whatsapp.net"
}
```

**Response:**
```json
{
  "data": {
    "url": "https://pps.whatsapp.net/v/...",
    "id": "123456789"
  }
}
```

---

### Block Contact

`POST /sessions/:sessionId/contacts/block`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Unblock Contact

`POST /sessions/:sessionId/contacts/unblock`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Get Blocklist

`GET /sessions/:sessionId/contacts/blocklist`

---

### Get User Info

`POST /sessions/:sessionId/contacts/info`

```json
{
  "jids": ["5511999999999@s.whatsapp.net"]
}
```

---

### Get Privacy Settings

`GET /sessions/:sessionId/contacts/privacy`

---

### Set Profile Picture

`POST /sessions/:sessionId/contacts/profile-picture`

```json
{
  "base64": "<base64-encoded-jpeg>"
}
```

---

## Groups

### List Groups

`GET /sessions/:sessionId/groups`

---

### Create Group

`POST /sessions/:sessionId/groups/create`

```json
{
  "name": "My Group",
  "participants": ["5511999999999", "5511888888888"]
}
```

---

### Get Group Info

`POST /sessions/:sessionId/groups/info`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Get Group Info from Invite Link

`POST /sessions/:sessionId/groups/invite-info`

```json
{ "inviteCode": "HffXhYmzzyJGec61oqMXiz" }
```

---

### Join Group with Invite Link

`POST /sessions/:sessionId/groups/join`

```json
{ "inviteCode": "HffXhYmzzyJGec61oqMXiz" }
```

**Response:**
```json
{
  "data": { "jid": "120362023605733675@g.us" }
}
```

---

### Get Invite Link

`POST /sessions/:sessionId/groups/invite-link`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Leave Group

`POST /sessions/:sessionId/groups/leave`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Update Participants

`POST /sessions/:sessionId/groups/participants`

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

`POST /sessions/:sessionId/groups/requests`

```json
{ "groupJid": "120362023605733675@g.us" }
```

---

### Approve / Reject Join Requests

`POST /sessions/:sessionId/groups/requests/action`

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

`POST /sessions/:sessionId/groups/name`

```json
{
  "groupJid": "120362023605733675@g.us",
  "text": "New Group Name"
}
```

---

### Update Group Description

`POST /sessions/:sessionId/groups/description`

```json
{
  "groupJid": "120362023605733675@g.us",
  "text": "New description"
}
```

---

### Update Group Photo

`POST /sessions/:sessionId/groups/photo`

```json
{
  "groupJid": "120362023605733675@g.us",
  "photoBase64": "<base64-encoded-jpeg>"
}
```

---

### Set Announce Mode

`POST /sessions/:sessionId/groups/announce`

Only admins can send messages when enabled.

```json
{
  "groupJid": "120362023605733675@g.us",
  "enabled": true
}
```

---

### Set Locked Mode

`POST /sessions/:sessionId/groups/locked`

Only admins can edit group info when enabled.

```json
{
  "groupJid": "120362023605733675@g.us",
  "enabled": true
}
```

---

### Set Join Approval

`POST /sessions/:sessionId/groups/join-approval`

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

`POST /sessions/:sessionId/chat/archive`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Mute Chat

`POST /sessions/:sessionId/chat/mute`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Pin Chat

`POST /sessions/:sessionId/chat/pin`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

### Unpin Chat

`POST /sessions/:sessionId/chat/unpin`

```json
{ "jid": "5511999999999@s.whatsapp.net" }
```

---

## Labels

Labels are a WhatsApp Business feature for organizing chats.

### Add Label to Chat

`POST /sessions/:sessionId/label/chat`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1"
}
```

---

### Remove Label from Chat

`POST /sessions/:sessionId/unlabel/chat`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1"
}
```

---

### Add Label to Message

`POST /sessions/:sessionId/label/message`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1",
  "messageId": "ABCDEF123456"
}
```

---

### Remove Label from Message

`POST /sessions/:sessionId/unlabel/message`

```json
{
  "jid": "5511999999999@s.whatsapp.net",
  "labelId": "1",
  "messageId": "ABCDEF123456"
}
```

---

### Edit Label

`POST /sessions/:sessionId/label/edit`

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

`POST /sessions/:sessionId/newsletter/create`

```json
{
  "name": "My Channel",
  "description": "Channel description",
  "picture": "<base64-encoded-image>"
}
```

---

### Get Newsletter Info

`POST /sessions/:sessionId/newsletter/info?jid=<newsletterJid>`

```bash
curl "http://localhost:8080/sessions/my-session/newsletter/info?jid=120363166361227321@newsletter" \
  -H 'Token: SESSION_TOKEN'
```

---

### Get Newsletter Info from Invite

`POST /sessions/:sessionId/newsletter/invite?code=<inviteCode>`

---

### List Subscribed Newsletters

`GET /sessions/:sessionId/newsletter/list`

---

### Get Newsletter Messages

`POST /sessions/:sessionId/newsletter/messages`

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

`POST /sessions/:sessionId/newsletter/subscribe`

```json
{ "newsletterJid": "120363166361227321@newsletter" }
```

---

## Community

Communities are groups of groups in WhatsApp.

### Create Community

`POST /sessions/:sessionId/community/create`

```json
{
  "name": "My Community",
  "description": "Community description"
}
```

---

### Add Subgroup to Community

`POST /sessions/:sessionId/community/participant/add`

```json
{
  "communityJid": "120363166361227321@g.us",
  "participants": ["120362023605733675@g.us"]
}
```

---

### Remove Subgroup from Community

`POST /sessions/:sessionId/community/participant/remove`

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

`POST /sessions/:sessionId/webhooks`

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

`GET /sessions/:sessionId/webhooks`

---

### Delete Webhook

`DELETE /sessions/:sessionId/webhooks/:wid`

```bash
curl -X DELETE http://localhost:8080/sessions/my-session/webhooks/webhook-uuid \
  -H 'Token: SESSION_KEY'
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
