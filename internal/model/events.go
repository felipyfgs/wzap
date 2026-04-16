package model

type EventType string

const EventAll EventType = "All"

// Messages
const (
	EventMessage              EventType = "Message"
	EventMessageRevoke        EventType = "MessageRevoke"
	EventMessageEdit          EventType = "MessageEdit"
	EventUndecryptableMessage EventType = "UndecryptableMessage"
	EventMediaRetry           EventType = "MediaRetry"
	EventReceipt              EventType = "Receipt"
	EventDeleteForMe          EventType = "DeleteForMe"
)

// Connection
const (
	EventConnected            EventType = "Connected"
	EventDisconnected         EventType = "Disconnected"
	EventManualLoginReconnect EventType = "ManualLoginReconnect"
)

// Pairing
const (
	EventQR                          EventType = "QR"
	EventQRScannedWithoutMultidevice EventType = "QRScannedWithoutMultidevice"
	EventPairSuccess                 EventType = "PairSuccess"
	EventPairError                   EventType = "PairError"
)

// Connection Errors
const (
	EventConnectFailure    EventType = "ConnectFailure"
	EventLoggedOut         EventType = "LoggedOut"
	EventStreamError       EventType = "StreamError"
	EventStreamReplaced    EventType = "StreamReplaced"
	EventKeepAliveTimeout  EventType = "KeepAliveTimeout"
	EventKeepAliveRestored EventType = "KeepAliveRestored"
	EventClientOutdated    EventType = "ClientOutdated"
	EventTemporaryBan      EventType = "TemporaryBan"
	EventCATRefreshError   EventType = "CATRefreshError"
)

// Contacts
const (
	EventContact      EventType = "Contact"
	EventPushName     EventType = "PushName"
	EventBusinessName EventType = "BusinessName"
)

// Profile & Identity
const (
	EventPicture        EventType = "Picture"
	EventIdentityChange EventType = "IdentityChange"
	EventUserAbout      EventType = "UserAbout"
)

// Groups
const (
	EventGroupInfo   EventType = "GroupInfo"
	EventJoinedGroup EventType = "JoinedGroup"
)

// Presence
const (
	EventPresence     EventType = "Presence"
	EventChatPresence EventType = "ChatPresence"
)

// Chat State
const (
	EventArchive               EventType = "Archive"
	EventMute                  EventType = "Mute"
	EventPin                   EventType = "Pin"
	EventStar                  EventType = "Star"
	EventClearChat             EventType = "ClearChat"
	EventDeleteChat            EventType = "DeleteChat"
	EventMarkChatAsRead        EventType = "MarkChatAsRead"
	EventUnarchiveChatsSetting EventType = "UnarchiveChatsSetting"
)

// Labels
const (
	EventLabelEdit               EventType = "LabelEdit"
	EventLabelAssociationChat    EventType = "LabelAssociationChat"
	EventLabelAssociationMessage EventType = "LabelAssociationMessage"
)

// Calls
const (
	EventCallOffer        EventType = "CallOffer"
	EventCallAccept       EventType = "CallAccept"
	EventCallTerminate    EventType = "CallTerminate"
	EventCallOfferNotice  EventType = "CallOfferNotice"
	EventCallRelayLatency EventType = "CallRelayLatency"
	EventCallPreAccept    EventType = "CallPreAccept"
	EventCallReject       EventType = "CallReject"
	EventCallTransport    EventType = "CallTransport"
	EventUnknownCallEvent EventType = "UnknownCallEvent"
)

// Newsletter (WhatsApp Channels)
const (
	EventNewsletterJoin       EventType = "NewsletterJoin"
	EventNewsletterLeave      EventType = "NewsletterLeave"
	EventNewsletterMuteChange EventType = "NewsletterMuteChange"
	EventNewsletterLiveUpdate EventType = "NewsletterLiveUpdate"
)

// Sync
const (
	EventHistorySync          EventType = "HistorySync"
	EventAppState             EventType = "AppState"
	EventAppStateSyncComplete EventType = "AppStateSyncComplete"
	EventAppStateSyncError    EventType = "AppStateSyncError"
	EventOfflineSyncCompleted EventType = "OfflineSyncCompleted"
	EventOfflineSyncPreview   EventType = "OfflineSyncPreview"
)

// Privacy & Settings
const (
	EventPrivacySettings EventType = "PrivacySettings"
	EventPushNameSetting EventType = "PushNameSetting"
	EventUserStatusMute  EventType = "UserStatusMute"
	EventBlocklistChange EventType = "BlocklistChange"
	EventBlocklist       EventType = "Blocklist"
)

// FB/Meta Bridge
const (
	EventFBMessage EventType = "FBMessage"
)

var ValidEventTypes = func() map[EventType]bool {
	types := []EventType{
		EventAll,
		// Messages
		EventMessage, EventMessageRevoke, EventMessageEdit, EventUndecryptableMessage, EventMediaRetry, EventReceipt, EventDeleteForMe,
		// Connection
		EventConnected, EventDisconnected, EventManualLoginReconnect,
		// Pairing
		EventQR, EventQRScannedWithoutMultidevice, EventPairSuccess, EventPairError,
		// Connection Errors
		EventConnectFailure, EventLoggedOut, EventStreamError, EventStreamReplaced,
		EventKeepAliveTimeout, EventKeepAliveRestored, EventClientOutdated,
		EventTemporaryBan, EventCATRefreshError,
		// Contacts
		EventContact, EventPushName, EventBusinessName,
		// Profile & Identity
		EventPicture, EventIdentityChange, EventUserAbout,
		// Groups
		EventGroupInfo, EventJoinedGroup,
		// Presence
		EventPresence, EventChatPresence,
		// Chat State
		EventArchive, EventMute, EventPin, EventStar, EventClearChat,
		EventDeleteChat, EventMarkChatAsRead, EventUnarchiveChatsSetting,
		// Labels
		EventLabelEdit, EventLabelAssociationChat, EventLabelAssociationMessage,
		// Calls
		EventCallOffer, EventCallAccept, EventCallTerminate, EventCallOfferNotice,
		EventCallRelayLatency, EventCallPreAccept, EventCallReject, EventCallTransport, EventUnknownCallEvent,
		// Newsletter
		EventNewsletterJoin, EventNewsletterLeave, EventNewsletterMuteChange, EventNewsletterLiveUpdate,
		// Sync
		EventHistorySync, EventAppState, EventAppStateSyncComplete, EventAppStateSyncError,
		EventOfflineSyncCompleted, EventOfflineSyncPreview,
		// Privacy & Settings
		EventPrivacySettings, EventPushNameSetting, EventUserStatusMute, EventBlocklistChange, EventBlocklist,
		// FB/Meta Bridge
		EventFBMessage,
	}
	m := make(map[EventType]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}()

func IsValidEventType(e EventType) bool {
	return ValidEventTypes[e]
}
