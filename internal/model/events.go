package model

// EventType represents a whatsmeow event name used for webhook filtering.
type EventType string

// EventAll subscribes to every event type.
const EventAll EventType = "All"

// Messages
const (
	EventMessage             EventType = "Message"
	EventUndecryptableMessage EventType = "UndecryptableMessage"
	EventMediaRetry          EventType = "MediaRetry"
	EventReceipt             EventType = "Receipt"
	EventDeleteForMe         EventType = "DeleteForMe"
)

// Connection / Session lifecycle
const (
	EventConnected                  EventType = "Connected"
	EventDisconnected               EventType = "Disconnected"
	EventConnectFailure             EventType = "ConnectFailure"
	EventLoggedOut                  EventType = "LoggedOut"
	EventPairSuccess                EventType = "PairSuccess"
	EventPairError                  EventType = "PairError"
	EventQR                         EventType = "QR"
	EventQRScannedWithoutMultidevice EventType = "QRScannedWithoutMultidevice"
	EventStreamError                EventType = "StreamError"
	EventStreamReplaced             EventType = "StreamReplaced"
	EventKeepAliveTimeout           EventType = "KeepAliveTimeout"
	EventKeepAliveRestored          EventType = "KeepAliveRestored"
	EventClientOutdated             EventType = "ClientOutdated"
	EventTemporaryBan               EventType = "TemporaryBan"
	EventCATRefreshError            EventType = "CATRefreshError"
	EventManualLoginReconnect       EventType = "ManualLoginReconnect"
)

// Contacts & Identity
const (
	EventContact        EventType = "Contact"
	EventPicture        EventType = "Picture"
	EventIdentityChange EventType = "IdentityChange"
	EventUserAbout      EventType = "UserAbout"
	EventPushName       EventType = "PushName"
	EventBusinessName   EventType = "BusinessName"
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

// Chat state
const (
	EventArchive             EventType = "Archive"
	EventMute                EventType = "Mute"
	EventPin                 EventType = "Pin"
	EventStar                EventType = "Star"
	EventClearChat           EventType = "ClearChat"
	EventDeleteChat          EventType = "DeleteChat"
	EventMarkChatAsRead      EventType = "MarkChatAsRead"
	EventUnarchiveChatsSetting EventType = "UnarchiveChatsSetting"
)

// Labels
const (
	EventLabelEdit                EventType = "LabelEdit"
	EventLabelAssociationChat     EventType = "LabelAssociationChat"
	EventLabelAssociationMessage  EventType = "LabelAssociationMessage"
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
	EventNewsletterJoin        EventType = "NewsletterJoin"
	EventNewsletterLeave       EventType = "NewsletterLeave"
	EventNewsletterMuteChange  EventType = "NewsletterMuteChange"
	EventNewsletterLiveUpdate  EventType = "NewsletterLiveUpdate"
)

// Sync
const (
	EventHistorySync         EventType = "HistorySync"
	EventAppState            EventType = "AppState"
	EventAppStateSyncComplete EventType = "AppStateSyncComplete"
	EventAppStateSyncError   EventType = "AppStateSyncError"
	EventOfflineSyncCompleted EventType = "OfflineSyncCompleted"
	EventOfflineSyncPreview  EventType = "OfflineSyncPreview"
)

// Privacy & Settings
const (
	EventPrivacySettings  EventType = "PrivacySettings"
	EventPushNameSetting  EventType = "PushNameSetting"
	EventUserStatusMute   EventType = "UserStatusMute"
	EventBlocklistChange  EventType = "BlocklistChange"
	EventBlocklist        EventType = "Blocklist"
)

// ValidEventTypes is the set of all supported event type values.
// Used for webhook subscription validation.
var ValidEventTypes = func() map[EventType]bool {
	types := []EventType{
		EventAll,
		// Messages
		EventMessage, EventUndecryptableMessage, EventMediaRetry, EventReceipt, EventDeleteForMe,
		// Connection
		EventConnected, EventDisconnected, EventConnectFailure, EventLoggedOut,
		EventPairSuccess, EventPairError, EventQR, EventQRScannedWithoutMultidevice,
		EventStreamError, EventStreamReplaced, EventKeepAliveTimeout, EventKeepAliveRestored,
		EventClientOutdated, EventTemporaryBan, EventCATRefreshError, EventManualLoginReconnect,
		// Contacts
		EventContact, EventPicture, EventIdentityChange, EventUserAbout, EventPushName, EventBusinessName,
		// Groups
		EventGroupInfo, EventJoinedGroup,
		// Presence
		EventPresence, EventChatPresence,
		// Chat state
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
		// Privacy
		EventPrivacySettings, EventPushNameSetting, EventUserStatusMute, EventBlocklistChange, EventBlocklist,
	}
	m := make(map[EventType]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}()

// IsValidEventType reports whether e is a recognised event type.
func IsValidEventType(e EventType) bool {
	return ValidEventTypes[e]
}
