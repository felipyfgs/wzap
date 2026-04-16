package model

type CapabilitySupport string

type EngineCapability string

const (
	SupportComplete    CapabilitySupport = "complete"
	SupportPartial     CapabilitySupport = "partial"
	SupportUnavailable CapabilitySupport = "unavailable"
)

const (
	CapabilityMessageText        EngineCapability = "message.text"
	CapabilityMessageMedia       EngineCapability = "message.media"
	CapabilityMessageSticker     EngineCapability = "message.sticker"
	CapabilityMessageLocation    EngineCapability = "message.location"
	CapabilityMessageButton      EngineCapability = "message.button"
	CapabilityMessageList        EngineCapability = "message.list"
	CapabilityMessageReaction    EngineCapability = "message.reaction"
	CapabilityMessageMarkRead    EngineCapability = "message.mark_read"
	CapabilityMessageLink        EngineCapability = "message.link"
	CapabilityMessagePoll        EngineCapability = "message.poll"
	CapabilityMessageContact     EngineCapability = "message.contact"
	CapabilityMessageEdit        EngineCapability = "message.edit"
	CapabilityMessageDelete      EngineCapability = "message.delete"
	CapabilityMessageForward     EngineCapability = "message.forward"
	CapabilityMessagePresence    EngineCapability = "message.presence"
	CapabilityMessageStatusText  EngineCapability = "message.status_text"
	CapabilityMessageStatusMedia EngineCapability = "message.status_media"
	CapabilitySessionConnect     EngineCapability = "session.connect"
	CapabilitySessionDisconnect  EngineCapability = "session.disconnect"
	CapabilitySessionQR          EngineCapability = "session.qr"
	CapabilitySessionPair        EngineCapability = "session.pair"
	CapabilitySessionLogout      EngineCapability = "session.logout"
	CapabilitySessionReconnect   EngineCapability = "session.reconnect"
	CapabilitySessionRestart     EngineCapability = "session.restart"
	CapabilitySessionStatus      EngineCapability = "session.status"
	CapabilitySessionProfile     EngineCapability = "session.profile"
	CapabilityMediaDownload      EngineCapability = "media.download"
)

type CapabilityContract struct {
	support map[string]map[EngineCapability]CapabilitySupport
}

var DefaultCapabilities = CapabilityContract{
	support: map[string]map[EngineCapability]CapabilitySupport{
		"whatsmeow": {
			CapabilityMessageText:        SupportComplete,
			CapabilityMessageMedia:       SupportComplete,
			CapabilityMessageSticker:     SupportComplete,
			CapabilityMessageLocation:    SupportComplete,
			CapabilityMessageButton:      SupportComplete,
			CapabilityMessageList:        SupportComplete,
			CapabilityMessageReaction:    SupportComplete,
			CapabilityMessageMarkRead:    SupportComplete,
			CapabilityMessageLink:        SupportComplete,
			CapabilityMessagePoll:        SupportComplete,
			CapabilityMessageContact:     SupportComplete,
			CapabilityMessageEdit:        SupportComplete,
			CapabilityMessageDelete:      SupportComplete,
			CapabilityMessageForward:     SupportComplete,
			CapabilityMessagePresence:    SupportComplete,
			CapabilityMessageStatusText:  SupportComplete,
			CapabilityMessageStatusMedia: SupportComplete,
			CapabilitySessionConnect:     SupportComplete,
			CapabilitySessionDisconnect:  SupportComplete,
			CapabilitySessionQR:          SupportComplete,
			CapabilitySessionPair:        SupportComplete,
			CapabilitySessionLogout:      SupportComplete,
			CapabilitySessionReconnect:   SupportComplete,
			CapabilitySessionRestart:     SupportComplete,
			CapabilitySessionStatus:      SupportComplete,
			CapabilitySessionProfile:     SupportComplete,
			CapabilityMediaDownload:      SupportComplete,
		},
	},
}

func (c CapabilityContract) Support(engine string, capability EngineCapability) CapabilitySupport {
	engineSupport, ok := c.support[engine]
	if !ok {
		return SupportUnavailable
	}

	support, ok := engineSupport[capability]
	if !ok {
		return SupportUnavailable
	}

	return support
}

func (c CapabilityContract) Supports(engine string, capability EngineCapability) bool {
	return c.Support(engine, capability) != SupportUnavailable
}
