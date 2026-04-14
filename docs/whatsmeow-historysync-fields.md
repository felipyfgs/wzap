# WhatsApp HistorySync Proto Field Mapping

**whatsmeow version**: `v0.0.0-20260327181659-02ec817e7cf4`

---

## 1. Event Wrapper: `events.HistorySync`

```go
// go.mau.fi/whatsmeow/types/events
type HistorySync struct {
    Data *waHistorySync.HistorySync
}
```

Simple wrapper — the real data is in `Data`.

---

## 2. Root: `waHistorySync.HistorySync`

| Field | Type | Proto # | Description |
|---|---|---|---|
| `syncType` | `HistorySyncType` (enum, **required**) | 1 | Type of sync event |
| `conversations` | `[]Conversation` | 2 | Chat conversations with messages |
| `statusV3Messages` | `[]WebMessageInfo` | 3 | Status/stories messages |
| `chunkOrder` | `uint32` | 5 | Order of this chunk in multi-chunk sync |
| `progress` | `uint32` | 6 | Sync progress percentage |
| `pushnames` | `[]Pushname` | 7 | Contact push names |
| `globalSettings` | `GlobalSettings` | 8 | Global app settings |
| `threadIDUserSecret` | `bytes` | 9 | Thread encryption secret |
| `threadDsTimeframeOffset` | `uint32` | 10 | DS timeframe offset |
| `recentStickers` | `[]StickerMetadata` | 11 | Recently used stickers |
| `pastParticipants` | `[]PastParticipants` | 12 | Past group participants |
| `callLogRecords` | `[]CallLogRecord` | 13 | Call log history |
| `aiWaitListState` | `BotAIWaitListState` | 14 | AI waitlist state |
| `phoneNumberToLidMappings` | `[]PhoneNumberToLIDMapping` | 15 | Phone → LID mappings |
| `companionMetaNonce` | `string` | 16 | Companion meta nonce |
| `shareableChatIdentifierEncryptionKey` | `bytes` | 17 | Chat identifier encryption key |
| `accounts` | `[]Account` | 18 | Multi-account info |
| `nctSalt` | `bytes` | 19 | NCT salt |

### HistorySyncType (enum)

| Value | Name | Description |
|---|---|---|
| 0 | `INITIAL_BOOTSTRAP` | Initial sync on device pairing — most messages |
| 1 | `INITIAL_STATUS_V3` | Initial status/story sync |
| 2 | `FULL` | Full history sync |
| 3 | `RECENT` | Recent messages sync |
| 4 | `PUSH_NAME` | Push name (contact name) sync only |
| 5 | `NON_BLOCKING_DATA` | Non-blocking data sync |
| 6 | `ON_DEMAND` | On-demand history request |

### Pushname

| Field | Type | Proto # |
|---|---|---|
| `ID` | `string` | 1 |
| `pushname` | `string` | 2 |

### Account

| Field | Type | Proto # |
|---|---|---|
| `lid` | `string` | 1 |
| `username` | `string` | 2 |
| `countryCode` | `string` | 3 |
| `isUsernameDeleted` | `bool` | 4 |

### PhoneNumberToLIDMapping

| Field | Type | Proto # |
|---|---|---|
| `pnJID` | `string` | 1 |
| `lidJID` | `string` | 2 |

---

## 3. `waHistorySync.Conversation`

| Field | Type | Proto # | Description |
|---|---|---|---|
| `ID` | `string` (**required**) | 1 | Chat JID (e.g. `5511999999999@s.whatsapp.net`) |
| `messages` | `[]HistorySyncMsg` | 2 | Messages in conversation |
| `newJID` | `string` | 3 | New JID (after number change) |
| `oldJID` | `string` | 4 | Old JID |
| `lastMsgTimestamp` | `uint64` | 5 | Last message timestamp |
| `unreadCount` | `uint32` | 6 | Unread message count |
| `readOnly` | `bool` | 7 | Read-only flag |
| `endOfHistoryTransfer` | `bool` | 8 | End of history transfer flag |
| `ephemeralExpiration` | `uint32` | 9 | Disappearing message duration (seconds) |
| `ephemeralSettingTimestamp` | `int64` | 10 | When ephemeral was set |
| `endOfHistoryTransferType` | `EndOfHistoryTransferType` | 11 | Transfer completion type |
| `conversationTimestamp` | `uint64` | 12 | Conversation timestamp |
| `name` | `string` | 13 | Contact/group name |
| `pHash` | `string` | 14 | Participant hash |
| `notSpam` | `bool` | 15 | Not spam flag |
| `archived` | `bool` | 16 | Archived flag |
| `disappearingMode` | `DisappearingMode` | 17 | Disappearing mode settings |
| `unreadMentionCount` | `uint32` | 18 | Unread mention count |
| `markedAsUnread` | `bool` | 19 | Marked as unread |
| `participant` | `[]GroupParticipant` | 20 | Group participants |
| `tcToken` | `bytes` | 21 | TC token |
| `tcTokenTimestamp` | `uint64` | 22 | TC token timestamp |
| `contactPrimaryIdentityKey` | `bytes` | 23 | Contact identity key |
| `pinned` | `uint32` | 24 | Pin position (0 = unpinned) |
| `muteEndTime` | `uint64` | 25 | Mute end timestamp |
| `wallpaper` | `WallpaperSettings` | 26 | Chat wallpaper |
| `mediaVisibility` | `MediaVisibility` | 27 | Media visibility setting |
| `tcTokenSenderTimestamp` | `uint64` | 28 | TC token sender timestamp |
| `suspended` | `bool` | 29 | Suspended flag |
| `terminated` | `bool` | 30 | Terminated flag |
| `createdAt` | `uint64` | 31 | Group creation timestamp |
| `createdBy` | `string` | 32 | Group creator JID |
| `description` | `string` | 33 | Group description |
| `support` | `bool` | 34 | Support chat flag |
| `isParentGroup` | `bool` | 35 | Community parent group |
| `isDefaultSubgroup` | `bool` | 36 | Default subgroup |
| `parentGroupID` | `string` | 37 | Parent group JID |
| `displayName` | `string` | 38 | Display name |
| `pnJID` | `string` | 39 | Phone number JID |
| `shareOwnPn` | `bool` | 40 | Share own phone number |
| `pnhDuplicateLidThread` | `bool` | 41 | PNH duplicate LID thread |
| `lidJID` | `string` | 42 | LID JID |
| `username` | `string` | 43 | Username |
| `lidOriginType` | `string` | 44 | LID origin type |
| `commentsCount` | `uint32` | 45 | Comments count |
| `locked` | `bool` | 46 | Locked flag |
| `systemMessageToInsert` | `PrivacySystemMessage` | 47 | System message |
| `capiCreatedGroup` | `bool` | 48 | Cloud API created group |
| `accountLid` | `string` | 49 | Account LID |
| `limitSharing` | `bool` | 50 | Limit sharing |
| `limitSharingSettingTimestamp` | `int64` | 51 | Limit sharing timestamp |
| `limitSharingTrigger` | `LimitSharing.Trigger` | 52 | Limit sharing trigger |
| `limitSharingInitiatedByMe` | `bool` | 53 | Initiated by me |
| `maibaAiThreadEnabled` | `bool` | 54 | AI thread enabled |
| `isMarketingMessageThread` | `bool` | 55 | Marketing thread |
| `isSenderNewAccount` | `bool` | 56 | Sender is new account |
| `afterReadDuration` | `uint32` | 57 | After read duration |

### EndOfHistoryTransferType

| Value | Name |
|---|---|
| 0 | `COMPLETE_BUT_MORE_MESSAGES_REMAIN_ON_PRIMARY` |
| 1 | `COMPLETE_AND_NO_MORE_MESSAGE_REMAIN_ON_PRIMARY` |
| 2 | `COMPLETE_ON_DEMAND_SYNC_BUT_MORE_MSG_REMAIN_ON_PRIMARY` |
| 3 | `COMPLETE_ON_DEMAND_SYNC_WITH_MORE_MSG_ON_PRIMARY_BUT_NO_ACCESS` |

### GroupParticipant

| Field | Type | Proto # |
|---|---|---|
| `userJID` | `string` (**required**) | 1 |
| `rank` | `Rank` (REGULAR=0, ADMIN=1, SUPERADMIN=2) | 2 |
| `memberLabel` | `MemberLabel` | 3 |

---

## 4. `waHistorySync.HistorySyncMsg`

| Field | Type | Proto # | Description |
|---|---|---|---|
| `message` | `WebMessageInfo` | 1 | Full message info |
| `msgOrderID` | `uint64` | 2 | Message ordering ID |

---

## 5. `waWeb.WebMessageInfo`

The complete message envelope with metadata.

| Field | Type | Proto # | Description |
|---|---|---|---|
| `key` | `MessageKey` (**required**) | 1 | Message key (remoteJID, fromMe, ID, participant) |
| `message` | `Message` | 2 | **The actual message content** |
| `messageTimestamp` | `uint64` | 3 | Server timestamp |
| `status` | `Status` | 4 | Delivery status |
| `participant` | `string` | 5 | Sender JID in groups |
| `messageC2STimestamp` | `uint64` | 6 | Client-to-server timestamp |
| `ignore` | `bool` | 16 | Ignore flag |
| `starred` | `bool` | 17 | Starred/bookmarked |
| `broadcast` | `bool` | 18 | Broadcast message |
| `pushName` | `string` | 19 | Sender's push name |
| `mediaCiphertextSHA256` | `bytes` | 20 | Media ciphertext hash |
| `multicast` | `bool` | 21 | Multicast flag |
| `urlText` | `bool` | 22 | URL in text |
| `urlNumber` | `bool` | 23 | URL number |
| `messageStubType` | `StubType` | 24 | System message type (group events etc.) |
| `clearMedia` | `bool` | 25 | Clear media flag |
| `messageStubParameters` | `[]string` | 26 | Stub message parameters |
| `duration` | `uint32` | 27 | Duration |
| `labels` | `[]string` | 28 | Business labels |
| `paymentInfo` | `PaymentInfo` | 29 | Payment info |
| `finalLiveLocation` | `LiveLocationMessage` | 30 | Final live location |
| `quotedPaymentInfo` | `PaymentInfo` | 31 | Quoted payment info |
| `ephemeralStartTimestamp` | `uint64` | 32 | Ephemeral start |
| `ephemeralDuration` | `uint32` | 33 | Ephemeral duration |
| `ephemeralOffToOn` | `bool` | 34 | Ephemeral off→on |
| `ephemeralOutOfSync` | `bool` | 35 | Ephemeral out of sync |
| `bizPrivacyStatus` | `BizPrivacyStatus` | 36 | Business privacy |
| `verifiedBizName` | `string` | 37 | Verified business name |
| `mediaData` | `MediaData` | 38 | Local media data |
| `photoChange` | `PhotoChange` | 39 | Photo change |
| `userReceipt` | `[]UserReceipt` | 40 | Read/delivery receipts |
| `reactions` | `[]Reaction` | 41 | Message reactions |
| `quotedStickerData` | `MediaData` | 42 | Quoted sticker data |
| `futureproofData` | `bytes` | 43 | Future-proof data |
| `statusPsa` | `StatusPSA` | 44 | Status PSA |
| `pollUpdates` | `[]PollUpdate` | 45 | Poll votes |
| `pollAdditionalMetadata` | `PollAdditionalMetadata` | 46 | Poll metadata |
| `agentID` | `string` | 47 | Agent ID |
| `statusAlreadyViewed` | `bool` | 48 | Status already viewed |
| `messageSecret` | `bytes` | 49 | Message secret |
| `keepInChat` | `KeepInChat` | 50 | Keep in chat info |
| `originalSelfAuthorUserJIDString` | `string` | 51 | Original author |
| `revokeMessageTimestamp` | `uint64` | 52 | Revoke timestamp |
| `pinInChat` | `PinInChat` | 54 | Pin in chat |
| `premiumMessageInfo` | `PremiumMessageInfo` | 55 | Premium message |
| `is1PBizBotMessage` | `bool` | 56 | 1P biz bot message |
| `isGroupHistoryMessage` | `bool` | 57 | Group history message |
| `botMessageInvokerJID` | `string` | 58 | Bot invoker JID |
| `commentMetadata` | `CommentMetadata` | 59 | Comment metadata |
| `eventResponses` | `[]EventResponse` | 61 | Event responses |
| `reportingTokenInfo` | `ReportingTokenInfo` | 62 | Reporting token |
| `newsletterServerID` | `uint64` | 63 | Newsletter server ID |
| `eventAdditionalMetadata` | `EventAdditionalMetadata` | 64 | Event metadata |

### MessageKey (`waCommon.MessageKey`)

| Field | Type | Proto # | Description |
|---|---|---|---|
| `remoteJID` | `string` | 1 | Chat JID |
| `fromMe` | `bool` | 2 | Sent by us |
| `ID` | `string` | 3 | Unique message ID |
| `participant` | `string` | 4 | Sender in group |

### Status (enum)

| Value | Name |
|---|---|
| 0 | `ERROR` |
| 1 | `PENDING` |
| 2 | `SERVER_ACK` |
| 3 | `DELIVERY_ACK` |
| 4 | `READ` |
| 5 | `PLAYED` |

### UserReceipt

| Field | Type | Proto # |
|---|---|---|
| `userJID` | `string` (**required**) | 1 |
| `receiptTimestamp` | `int64` | 2 |
| `readTimestamp` | `int64` | 3 |
| `playedTimestamp` | `int64` | 4 |
| `pendingDeviceJID` | `[]string` | 5 |
| `deliveredDeviceJID` | `[]string` | 6 |

### Reaction

| Field | Type | Proto # |
|---|---|---|
| `key` | `MessageKey` | 1 |
| `text` | `string` | 2 |
| `groupingKey` | `string` | 3 |
| `senderTimestampMS` | `int64` | 4 |
| `unread` | `bool` | 5 |

---

## 6. `waE2E.Message` — All Message Types

The `Message` struct uses oneof-style optional fields. Each message type is a separate field:

### Core Message Types

| Field | Type | Proto # | Description |
|---|---|---|---|
| `conversation` | `string` | 1 | **Plain text message** |
| `imageMessage` | `ImageMessage` | 3 | Image |
| `contactMessage` | `ContactMessage` | 4 | Contact card |
| `locationMessage` | `LocationMessage` | 5 | Location |
| `extendedTextMessage` | `ExtendedTextMessage` | 6 | Rich text with link preview |
| `documentMessage` | `DocumentMessage` | 7 | Document/file |
| `audioMessage` | `AudioMessage` | 8 | Audio/voice note |
| `videoMessage` | `VideoMessage` | 9 | Video |
| `stickerMessage` | `StickerMessage` | 26 | Sticker |
| `contactsArrayMessage` | `ContactsArrayMessage` | 13 | Multiple contacts |
| `liveLocationMessage` | `LiveLocationMessage` | 18 | Live location |
| `reactionMessage` | `ReactionMessage` | 46 | Reaction emoji |
| `pollCreationMessage` | `PollCreationMessage` | 49 | Poll (v1) |
| `pollCreationMessageV2` | `PollCreationMessage` | 60 | Poll (v2) |
| `pollCreationMessageV3` | `PollCreationMessage` | 64 | Poll (v3) |
| `pollUpdateMessage` | `PollUpdateMessage` | 50 | Poll vote |
| `eventMessage` | `EventMessage` | 75 | Event/calendar |
| `commentMessage` | `CommentMessage` | 77 | Comment |

### Wrapped/FutureProof Messages

| Field | Type | Proto # | Description |
|---|---|---|---|
| `viewOnceMessage` | `FutureProofMessage` | 37 | View once (v1) — unwrap `.message` |
| `viewOnceMessageV2` | `FutureProofMessage` | 55 | View once (v2) — unwrap `.message` |
| `viewOnceMessageV2Extension` | `FutureProofMessage` | 59 | View once extension |
| `ephemeralMessage` | `FutureProofMessage` | 40 | Ephemeral wrapper |
| `documentWithCaptionMessage` | `FutureProofMessage` | 53 | Document with caption |
| `editedMessage` | `FutureProofMessage` | 58 | Edited message |

### Protocol/System Messages

| Field | Type | Proto # | Description |
|---|---|---|---|
| `protocolMessage` | `ProtocolMessage` | 12 | Protocol message (revoke, edit, history sync notification) |
| `senderKeyDistributionMessage` | `SenderKeyDistributionMessage` | 2 | Key distribution |
| `callLogMesssage` | `CallLogMessage` | 69 | Call log |
| `groupInviteMessage` | `GroupInviteMessage` | 28 | Group invite |
| `ptvMessage` | `VideoMessage` | 66 | Push-to-talk video |
| `albumMessage` | `AlbumMessage` | 83 | Album container |

### Business/Interactive Messages

| Field | Type | Proto # | Description |
|---|---|---|---|
| `templateMessage` | `TemplateMessage` | 25 | Template message |
| `listMessage` | `ListMessage` | 36 | List message |
| `listResponseMessage` | `ListResponseMessage` | 39 | List response |
| `buttonsMessage` | `ButtonsMessage` | 42 | Buttons |
| `buttonsResponseMessage` | `ButtonsResponseMessage` | 43 | Buttons response |
| `interactiveMessage` | `InteractiveMessage` | 45 | Interactive (native flow) |
| `interactiveResponseMessage` | `InteractiveResponseMessage` | 48 | Interactive response |
| `productMessage` | `ProductMessage` | 30 | Product |
| `orderMessage` | `OrderMessage` | 38 | Order |

---

## 7. Media Message Types — Download Fields

### ImageMessage

| Field | Type | Proto # | For Download? |
|---|---|---|---|
| `URL` | `string` | 1 | ✅ CDN URL |
| `mimetype` | `string` | 2 | ✅ MIME type |
| `fileSHA256` | `bytes` | 4 | ✅ Plaintext file hash |
| `fileLength` | `uint64` | 5 | ✅ File size |
| `mediaKey` | `bytes` | 8 | ✅ **Decryption key** |
| `fileEncSHA256` | `bytes` | 9 | ✅ Encrypted file hash |
| `directPath` | `string` | 11 | ✅ **Direct download path** |
| `mediaKeyTimestamp` | `int64` | 12 | ✅ Key timestamp |
| `caption` | `string` | 3 | Caption text |
| `height` | `uint32` | 6 | Dimensions |
| `width` | `uint32` | 7 | Dimensions |
| `JPEGThumbnail` | `bytes` | 16 | Inline thumbnail |
| `viewOnce` | `bool` | 25 | View once flag |
| `imageSourceType` | `ImageSourceType` | 31 | AI generated etc. |

### VideoMessage

| Field | Type | Proto # | For Download? |
|---|---|---|---|
| `URL` | `string` | 1 | ✅ CDN URL |
| `mimetype` | `string` | 2 | ✅ MIME type |
| `fileSHA256` | `bytes` | 3 | ✅ Plaintext file hash |
| `fileLength` | `uint64` | 4 | ✅ File size |
| `mediaKey` | `bytes` | 6 | ✅ **Decryption key** |
| `fileEncSHA256` | `bytes` | 11 | ✅ Encrypted file hash |
| `directPath` | `string` | 13 | ✅ **Direct download path** |
| `mediaKeyTimestamp` | `int64` | 14 | ✅ Key timestamp |
| `caption` | `string` | 7 | Caption text |
| `gifPlayback` | `bool` | 8 | GIF flag |
| `height` | `uint32` | 9 | Dimensions |
| `width` | `uint32` | 10 | Dimensions |
| `seconds` | `uint32` | 5 | Duration |
| `JPEGThumbnail` | `bytes` | 16 | Inline thumbnail |
| `viewOnce` | `bool` | 20 | View once flag |
| `streamingSidecar` | `bytes` | 18 | Streaming sidecar |

### AudioMessage

| Field | Type | Proto # | For Download? |
|---|---|---|---|
| `URL` | `string` | 1 | ✅ CDN URL |
| `mimetype` | `string` | 2 | ✅ MIME type |
| `fileSHA256` | `bytes` | 3 | ✅ Plaintext file hash |
| `fileLength` | `uint64` | 4 | ✅ File size |
| `mediaKey` | `bytes` | 7 | ✅ **Decryption key** |
| `fileEncSHA256` | `bytes` | 8 | ✅ Encrypted file hash |
| `directPath` | `string` | 9 | ✅ **Direct download path** |
| `mediaKeyTimestamp` | `int64` | 10 | ✅ Key timestamp |
| `seconds` | `uint32` | 5 | Duration |
| `PTT` | `bool` | 6 | Push-to-talk (voice note) flag |
| `waveform` | `bytes` | 19 | Audio waveform |
| `viewOnce` | `bool` | 21 | View once flag |

### DocumentMessage

| Field | Type | Proto # | For Download? |
|---|---|---|---|
| `URL` | `string` | 1 | ✅ CDN URL |
| `mimetype` | `string` | 2 | ✅ MIME type |
| `fileSHA256` | `bytes` | 4 | ✅ Plaintext file hash |
| `fileLength` | `uint64` | 5 | ✅ File size |
| `mediaKey` | `bytes` | 7 | ✅ **Decryption key** |
| `fileEncSHA256` | `bytes` | 9 | ✅ Encrypted file hash |
| `directPath` | `string` | 10 | ✅ **Direct download path** |
| `mediaKeyTimestamp` | `int64` | 11 | ✅ Key timestamp |
| `title` | `string` | 3 | Document title |
| `fileName` | `string` | 8 | Original filename |
| `pageCount` | `uint32` | 6 | PDF page count |
| `caption` | `string` | 20 | Caption text |
| `JPEGThumbnail` | `bytes` | 16 | Inline thumbnail |

### StickerMessage

| Field | Type | Proto # | For Download? |
|---|---|---|---|
| `URL` | `string` | 1 | ✅ CDN URL |
| `mimetype` | `string` | 5 | ✅ MIME type |
| `fileSHA256` | `bytes` | 2 | ✅ Plaintext file hash |
| `fileLength` | `uint64` | 9 | ✅ File size |
| `mediaKey` | `bytes` | 4 | ✅ **Decryption key** |
| `fileEncSHA256` | `bytes` | 3 | ✅ Encrypted file hash |
| `directPath` | `string` | 8 | ✅ **Direct download path** |
| `mediaKeyTimestamp` | `int64` | 10 | ✅ Key timestamp |
| `height` | `uint32` | 6 | Dimensions |
| `width` | `uint32` | 7 | Dimensions |
| `isAnimated` | `bool` | 13 | Animated sticker |
| `isLottie` | `bool` | 21 | Lottie format |
| `pngThumbnail` | `bytes` | 16 | PNG thumbnail |

---

## 8. Non-Media Message Types — Key Fields

### ContactMessage

| Field | Type | Proto # |
|---|---|---|
| `displayName` | `string` | 1 |
| `vcard` | `string` | 16 |

### ContactsArrayMessage

| Field | Type | Proto # |
|---|---|---|
| `displayName` | `string` | 1 |
| `contacts` | `[]ContactMessage` | 2 |

### LocationMessage

| Field | Type | Proto # |
|---|---|---|
| `degreesLatitude` | `double` | 1 |
| `degreesLongitude` | `double` | 2 |
| `name` | `string` | 3 |
| `address` | `string` | 4 |
| `URL` | `string` | 5 |
| `isLive` | `bool` | 6 |
| `accuracyInMeters` | `uint32` | 7 |
| `comment` | `string` | 11 |
| `JPEGThumbnail` | `bytes` | 16 |

### LiveLocationMessage

| Field | Type | Proto # |
|---|---|---|
| `degreesLatitude` | `double` | 1 |
| `degreesLongitude` | `double` | 2 |
| `accuracyInMeters` | `uint32` | 3 |
| `speedInMps` | `float` | 4 |
| `degreesClockwiseFromMagneticNorth` | `uint32` | 5 |
| `caption` | `string` | 6 |
| `sequenceNumber` | `int64` | 7 |

### ExtendedTextMessage (Link Preview)

| Field | Type | Proto # |
|---|---|---|
| `text` | `string` | 1 |
| `matchedText` | `string` | 2 |
| `description` | `string` | 5 |
| `title` | `string` | 6 |
| `previewType` | `PreviewType` | 10 |
| `JPEGThumbnail` | `bytes` | 16 |
| `thumbnailDirectPath` | `string` | 19 |
| `thumbnailSHA256` | `bytes` | 20 |
| `thumbnailEncSHA256` | `bytes` | 21 |
| `mediaKey` | `bytes` | 22 |

### PollCreationMessage

| Field | Type | Proto # |
|---|---|---|
| `encKey` | `bytes` | 1 |
| `name` | `string` | 2 |
| `options` | `[]Option{optionName, optionHash}` | 3 |
| `selectableOptionsCount` | `uint32` | 4 |
| `pollType` | `PollType` (POLL/QUIZ) | 7 |

### ReactionMessage

| Field | Type | Proto # |
|---|---|---|
| `key` | `MessageKey` | 1 |
| `text` | `string` | 2 |
| `senderTimestampMS` | `int64` | 4 |

### EventMessage

| Field | Type | Proto # |
|---|---|---|
| `isCanceled` | `bool` | 2 |
| `name` | `string` | 3 |
| `description` | `string` | 4 |
| `location` | `LocationMessage` | 5 |
| `joinLink` | `string` | 6 |
| `startTime` | `int64` | 7 |
| `endTime` | `int64` | 8 |

---

## 9. HistorySyncNotification (in ProtocolMessage)

This is how the phone signals a history blob is available for download:

| Field | Type | Proto # | Description |
|---|---|---|---|
| `fileSHA256` | `bytes` | 1 | Blob hash |
| `fileLength` | `uint64` | 2 | Blob size |
| `mediaKey` | `bytes` | 3 | Decryption key |
| `fileEncSHA256` | `bytes` | 4 | Encrypted blob hash |
| `directPath` | `string` | 5 | Download path |
| `syncType` | `HistorySyncType` | 6 | Sync type |
| `chunkOrder` | `uint32` | 7 | Chunk order |
| `originalMessageID` | `string` | 8 | Original message ID |
| `progress` | `uint32` | 9 | Progress |
| `oldestMsgInChunkTimestampSec` | `int64` | 10 | Oldest msg timestamp |
| `initialHistBootstrapInlinePayload` | `bytes` | 11 | Inline payload |
| `peerDataRequestSessionID` | `string` | 12 | Session ID |
| `fullHistorySyncOnDemandRequestMetadata` | `...` | 13 | On-demand metadata |
| `encHandle` | `string` | 14 | Encryption handle |

---

## 10. Summary: Media Download Pattern

All downloadable media types share these common fields needed for `whatsmeow.Client.Download()`:

```go
type DownloadableMessage interface {
    GetURL() string
    GetDirectPath() string
    GetMediaKey() []byte
    GetFileEncSHA256() []byte
    GetFileSHA256() []byte
    GetFileLength() uint64
    GetMimetype() string
    GetMediaKeyTimestamp() int64
}
```

**Media types implementing this pattern:**
- `ImageMessage`
- `VideoMessage`
- `AudioMessage`
- `DocumentMessage`
- `StickerMessage`
- `StickerPackMessage`

**whatsmeow provides `client.Download(msg)` which accepts any downloadable message and handles decryption automatically.**

---

## 11. Data Extraction Hierarchy

```
events.HistorySync
└── Data: *waHistorySync.HistorySync
    ├── syncType (INITIAL_BOOTSTRAP / RECENT / FULL / PUSH_NAME / ...)
    ├── pushnames[] → {ID, pushname}     ← contact names
    ├── conversations[]
    │   ├── ID                             ← chat JID
    │   ├── name                           ← chat name
    │   ├── participant[]                  ← group members
    │   ├── archived, pinned, muteEndTime  ← chat metadata
    │   └── messages[] (HistorySyncMsg)
    │       ├── msgOrderID
    │       └── message (WebMessageInfo)
    │           ├── key {remoteJID, fromMe, ID, participant}
    │           ├── messageTimestamp
    │           ├── status (ERROR/PENDING/SERVER_ACK/DELIVERY_ACK/READ/PLAYED)
    │           ├── pushName                ← sender display name
    │           ├── starred
    │           ├── reactions[]
    │           └── message (waE2E.Message)
    │               ├── conversation        ← plain text
    │               ├── imageMessage        ← image + media fields
    │               ├── videoMessage         ← video + media fields
    │               ├── audioMessage         ← audio + media fields
    │               ├── documentMessage      ← document + media fields
    │               ├── stickerMessage       ← sticker + media fields
    │               ├── contactMessage       ← contact vcard
    │               ├── locationMessage      ← coordinates
    │               ├── extendedTextMessage  ← text + link preview
    │               ├── reactionMessage      ← emoji reaction
    │               ├── pollCreationMessage  ← poll
    │               ├── viewOnceMessage      ← unwrap → Message
    │               ├── ephemeralMessage     ← unwrap → Message
    │               └── protocolMessage      ← system (edit, revoke, etc.)
    └── globalSettings                      ← app settings
```
