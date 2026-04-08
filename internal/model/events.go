package model

type EventType string

type EventCategory string

const EventAll EventType = "All"

const (
	CategoryMessages         EventCategory = "Messages"
	CategoryConnection       EventCategory = "Connection"
	CategoryPairing          EventCategory = "Pairing"
	CategoryConnectionErrors EventCategory = "Connection Errors"
	CategoryContacts         EventCategory = "Contacts"
	CategoryProfileIdentity  EventCategory = "Profile & Identity"
	CategoryGroups           EventCategory = "Groups"
	CategoryPresence         EventCategory = "Presence"
	CategoryChatState        EventCategory = "Chat State"
	CategoryLabels           EventCategory = "Labels"
	CategoryCalls            EventCategory = "Calls"
	CategoryNewsletter       EventCategory = "Newsletter"
	CategorySync             EventCategory = "Sync"
	CategoryPrivacySettings  EventCategory = "Privacy & Settings"
	CategoryFBMetaBridge     EventCategory = "FB/Meta Bridge"
	CategorySpecial          EventCategory = "Special"
)

// Messages
const (
	EventMessage              EventType = "Message"
	EventMessageRevoke        EventType = "MessageRevoke"
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
		EventMessage, EventUndecryptableMessage, EventMediaRetry, EventReceipt, EventDeleteForMe,
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

func CategoryFor(e EventType) EventCategory {
	switch e {
	case EventAll:
		return CategorySpecial
	case EventMessage, EventUndecryptableMessage, EventMediaRetry, EventReceipt, EventDeleteForMe:
		return CategoryMessages
	case EventConnected, EventDisconnected, EventManualLoginReconnect:
		return CategoryConnection
	case EventQR, EventQRScannedWithoutMultidevice, EventPairSuccess, EventPairError:
		return CategoryPairing
	case EventConnectFailure, EventLoggedOut, EventStreamError, EventStreamReplaced,
		EventKeepAliveTimeout, EventKeepAliveRestored, EventClientOutdated,
		EventTemporaryBan, EventCATRefreshError:
		return CategoryConnectionErrors
	case EventContact, EventPushName, EventBusinessName:
		return CategoryContacts
	case EventPicture, EventIdentityChange, EventUserAbout:
		return CategoryProfileIdentity
	case EventGroupInfo, EventJoinedGroup:
		return CategoryGroups
	case EventPresence, EventChatPresence:
		return CategoryPresence
	case EventArchive, EventMute, EventPin, EventStar, EventClearChat,
		EventDeleteChat, EventMarkChatAsRead, EventUnarchiveChatsSetting:
		return CategoryChatState
	case EventLabelEdit, EventLabelAssociationChat, EventLabelAssociationMessage:
		return CategoryLabels
	case EventCallOffer, EventCallAccept, EventCallTerminate, EventCallOfferNotice,
		EventCallRelayLatency, EventCallPreAccept, EventCallReject, EventCallTransport, EventUnknownCallEvent:
		return CategoryCalls
	case EventNewsletterJoin, EventNewsletterLeave, EventNewsletterMuteChange, EventNewsletterLiveUpdate:
		return CategoryNewsletter
	case EventHistorySync, EventAppState, EventAppStateSyncComplete, EventAppStateSyncError,
		EventOfflineSyncCompleted, EventOfflineSyncPreview:
		return CategorySync
	case EventPrivacySettings, EventPushNameSetting, EventUserStatusMute, EventBlocklistChange, EventBlocklist:
		return CategoryPrivacySettings
	case EventFBMessage:
		return CategoryFBMetaBridge
	default:
		return ""
	}
}
