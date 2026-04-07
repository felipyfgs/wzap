package model

type CapabilitySupport string

type EngineCapability string

const (
	CapabilitySupportComplete    CapabilitySupport = "complete"
	CapabilitySupportPartial    CapabilitySupport = "partial"
	CapabilitySupportUnavailable CapabilitySupport = "unavailable"
)

const (
	CapabilityMessageText       EngineCapability = "message.text"
	CapabilityMessageMedia      EngineCapability = "message.media"
	CapabilityMessageSticker    EngineCapability = "message.sticker"
	CapabilityMessageLocation   EngineCapability = "message.location"
	CapabilityMessageButton     EngineCapability = "message.button"
	CapabilityMessageList       EngineCapability = "message.list"
	CapabilityMessageReaction   EngineCapability = "message.reaction"
	CapabilityMessageMarkRead   EngineCapability = "message.mark_read"
	CapabilityMessageLink       EngineCapability = "message.link"
	CapabilityMessagePoll       EngineCapability = "message.poll"
	CapabilityMessageContact    EngineCapability = "message.contact"
	CapabilityMessageEdit       EngineCapability = "message.edit"
	CapabilityMessageDelete     EngineCapability = "message.delete"
	CapabilityMessageForward    EngineCapability = "message.forward"
	CapabilityMessagePresence   EngineCapability = "message.presence"
	CapabilityMessageStatusText EngineCapability = "message.status_text"
	CapabilityMessageStatusMedia EngineCapability = "message.status_media"
	CapabilitySessionConnect    EngineCapability = "session.connect"
	CapabilitySessionDisconnect EngineCapability = "session.disconnect"
	CapabilitySessionQR         EngineCapability = "session.qr"
	CapabilitySessionPair       EngineCapability = "session.pair"
	CapabilitySessionLogout     EngineCapability = "session.logout"
	CapabilitySessionReconnect  EngineCapability = "session.reconnect"
	CapabilitySessionRestart    EngineCapability = "session.restart"
	CapabilitySessionStatus     EngineCapability = "session.status"
	CapabilitySessionProfile    EngineCapability = "session.profile"
	CapabilityMediaDownload     EngineCapability = "media.download"
)

type EngineCapabilityContract struct {
	support map[string]map[EngineCapability]CapabilitySupport
}

var DefaultEngineCapabilityContract = EngineCapabilityContract{
	support: map[string]map[EngineCapability]CapabilitySupport{
		"whatsmeow": {
			CapabilityMessageText:        CapabilitySupportComplete,
			CapabilityMessageMedia:       CapabilitySupportComplete,
			CapabilityMessageSticker:     CapabilitySupportComplete,
			CapabilityMessageLocation:    CapabilitySupportComplete,
			CapabilityMessageButton:      CapabilitySupportComplete,
			CapabilityMessageList:        CapabilitySupportComplete,
			CapabilityMessageReaction:    CapabilitySupportComplete,
			CapabilityMessageMarkRead:    CapabilitySupportComplete,
			CapabilityMessageLink:        CapabilitySupportComplete,
			CapabilityMessagePoll:        CapabilitySupportComplete,
			CapabilityMessageContact:     CapabilitySupportComplete,
			CapabilityMessageEdit:        CapabilitySupportComplete,
			CapabilityMessageDelete:      CapabilitySupportComplete,
			CapabilityMessageForward:     CapabilitySupportComplete,
			CapabilityMessagePresence:    CapabilitySupportComplete,
			CapabilityMessageStatusText:  CapabilitySupportComplete,
			CapabilityMessageStatusMedia: CapabilitySupportComplete,
			CapabilitySessionConnect:     CapabilitySupportComplete,
			CapabilitySessionDisconnect:  CapabilitySupportComplete,
			CapabilitySessionQR:          CapabilitySupportComplete,
			CapabilitySessionPair:        CapabilitySupportComplete,
			CapabilitySessionLogout:      CapabilitySupportComplete,
			CapabilitySessionReconnect:   CapabilitySupportComplete,
			CapabilitySessionRestart:     CapabilitySupportComplete,
			CapabilitySessionStatus:      CapabilitySupportComplete,
			CapabilitySessionProfile:     CapabilitySupportComplete,
			CapabilityMediaDownload:      CapabilitySupportComplete,
		},
		"cloud_api": {
			CapabilityMessageText:        CapabilitySupportComplete,
			CapabilityMessageMedia:       CapabilitySupportComplete,
			CapabilityMessageSticker:     CapabilitySupportComplete,
			CapabilityMessageLocation:    CapabilitySupportComplete,
			CapabilityMessageButton:      CapabilitySupportComplete,
			CapabilityMessageList:        CapabilitySupportComplete,
			CapabilityMessageReaction:    CapabilitySupportComplete,
			CapabilityMessageMarkRead:    CapabilitySupportComplete,
			CapabilityMessageLink:        CapabilitySupportPartial,
			CapabilityMessagePoll:        CapabilitySupportUnavailable,
			CapabilityMessageContact:     CapabilitySupportUnavailable,
			CapabilityMessageEdit:        CapabilitySupportUnavailable,
			CapabilityMessageDelete:      CapabilitySupportUnavailable,
			CapabilityMessageForward:     CapabilitySupportUnavailable,
			CapabilityMessagePresence:    CapabilitySupportUnavailable,
			CapabilityMessageStatusText:  CapabilitySupportUnavailable,
			CapabilityMessageStatusMedia: CapabilitySupportUnavailable,
			CapabilitySessionConnect:     CapabilitySupportPartial,
			CapabilitySessionDisconnect:  CapabilitySupportPartial,
			CapabilitySessionQR:          CapabilitySupportUnavailable,
			CapabilitySessionPair:        CapabilitySupportUnavailable,
			CapabilitySessionLogout:      CapabilitySupportPartial,
			CapabilitySessionReconnect:   CapabilitySupportPartial,
			CapabilitySessionRestart:     CapabilitySupportPartial,
			CapabilitySessionStatus:      CapabilitySupportPartial,
			CapabilitySessionProfile:     CapabilitySupportPartial,
			CapabilityMediaDownload:      CapabilitySupportUnavailable,
		},
	},
}

func (c EngineCapabilityContract) Support(engine string, capability EngineCapability) CapabilitySupport {
	engineSupport, ok := c.support[engine]
	if !ok {
		return CapabilitySupportUnavailable
	}

	support, ok := engineSupport[capability]
	if !ok {
		return CapabilitySupportUnavailable
	}

	return support
}

func (c EngineCapabilityContract) Supports(engine string, capability EngineCapability) bool {
	return c.Support(engine, capability) != CapabilitySupportUnavailable
}
