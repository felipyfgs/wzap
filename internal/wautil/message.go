package wautil

import (
	"strings"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

// ExtractMessageContent extracts the type, body, and media type from a WhatsApp proto message.
func ExtractMessageContent(msg *waE2E.Message) (msgType, body, mediaType string) {
	if msg == nil {
		return "unknown", "", ""
	}
	switch {
	case msg.GetConversation() != "":
		return "text", msg.GetConversation(), ""
	case msg.GetExtendedTextMessage() != nil:
		return "text", msg.GetExtendedTextMessage().GetText(), ""
	case msg.GetImageMessage() != nil:
		return "image", msg.GetImageMessage().GetCaption(), msg.GetImageMessage().GetMimetype()
	case msg.GetVideoMessage() != nil:
		return "video", msg.GetVideoMessage().GetCaption(), msg.GetVideoMessage().GetMimetype()
	case msg.GetAudioMessage() != nil:
		return "audio", "", msg.GetAudioMessage().GetMimetype()
	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		return "document", dm.GetFileName(), dm.GetMimetype()
	case msg.GetStickerMessage() != nil:
		return "sticker", "", msg.GetStickerMessage().GetMimetype()
	case msg.GetContactMessage() != nil:
		return "contact", msg.GetContactMessage().GetDisplayName(), ""
	case msg.GetLocationMessage() != nil:
		return "location", msg.GetLocationMessage().GetName(), ""
	case msg.GetListMessage() != nil:
		return "list", msg.GetListMessage().GetTitle(), ""
	case msg.GetButtonsMessage() != nil:
		return "buttons", msg.GetButtonsMessage().GetContentText(), ""
	case msg.GetPollCreationMessage() != nil:
		return "poll", msg.GetPollCreationMessage().GetName(), ""
	case msg.GetReactionMessage() != nil:
		return "reaction", msg.GetReactionMessage().GetText(), ""
	case msg.GetTemplateMessage() != nil:
		t := msg.GetTemplateMessage()
		if t.GetHydratedTemplate() != nil {
			hydrated := t.GetHydratedTemplate()
			return "template", hydrated.GetHydratedContentText(), ""
		}
		return "template", "", ""
	case msg.GetInteractiveMessage() != nil:
		im := msg.GetInteractiveMessage()
		subtitle := ""
		if im.GetHeader() != nil {
			subtitle = im.GetHeader().GetSubtitle()
		}
		return "interactive", subtitle, ""
	case msg.GetPollUpdateMessage() != nil:
		return "poll_update", "", ""
	case msg.GetDocumentWithCaptionMessage() != nil:
		return "document", "", ""
	default:
		return "unknown", "", ""
	}
}

// ExtractMediaDownloadInfo extracts media download parameters from a WhatsApp proto message.
func ExtractMediaDownloadInfo(msg *waE2E.Message) (directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, ok bool) {
	if msg == nil {
		return "", nil, nil, nil, 0, false
	}
	switch {
	case msg.GetImageMessage() != nil:
		im := msg.GetImageMessage()
		return im.GetDirectPath(), im.GetFileEncSHA256(), im.GetFileSHA256(), im.GetMediaKey(), int(im.GetFileLength() & 0x7FFFFFFF), true
	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		return vm.GetDirectPath(), vm.GetFileEncSHA256(), vm.GetFileSHA256(), vm.GetMediaKey(), int(vm.GetFileLength() & 0x7FFFFFFF), true
	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		return am.GetDirectPath(), am.GetFileEncSHA256(), am.GetFileSHA256(), am.GetMediaKey(), int(am.GetFileLength() & 0x7FFFFFFF), true
	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		return dm.GetDirectPath(), dm.GetFileEncSHA256(), dm.GetFileSHA256(), dm.GetMediaKey(), int(dm.GetFileLength() & 0x7FFFFFFF), true
	case msg.GetStickerMessage() != nil:
		sm := msg.GetStickerMessage()
		return sm.GetDirectPath(), sm.GetFileEncSHA256(), sm.GetFileSHA256(), sm.GetMediaKey(), int(sm.GetFileLength() & 0x7FFFFFFF), true
	default:
		return "", nil, nil, nil, 0, false
	}
}

// ExtractContextInfo retorna o ContextInfo aninhado dentro do tipo concreto da
// mensagem. waE2E.Message não expõe um getter de topo — cada subtipo tem o seu
// próprio. Cobre os tipos que carregam ContextInfo no protocol e que podem ser
// recebidos como mensagem normal de usuário; tipos puramente protocolares
// (ProtocolMessage/SenderKeyDistribution) ficam de fora porque não chegam aqui
// como mensagens visíveis.
func ExtractContextInfo(msg *waE2E.Message) *waE2E.ContextInfo {
	if msg == nil {
		return nil
	}
	switch {
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		return msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		return msg.GetVideoMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		return msg.GetAudioMessage().GetContextInfo()
	case msg.GetDocumentMessage() != nil:
		return msg.GetDocumentMessage().GetContextInfo()
	case msg.GetStickerMessage() != nil:
		return msg.GetStickerMessage().GetContextInfo()
	case msg.GetContactMessage() != nil:
		return msg.GetContactMessage().GetContextInfo()
	case msg.GetContactsArrayMessage() != nil:
		return msg.GetContactsArrayMessage().GetContextInfo()
	case msg.GetLiveLocationMessage() != nil:
		return msg.GetLiveLocationMessage().GetContextInfo()
	case msg.GetButtonsMessage() != nil:
		return msg.GetButtonsMessage().GetContextInfo()
	case msg.GetButtonsResponseMessage() != nil:
		return msg.GetButtonsResponseMessage().GetContextInfo()
	case msg.GetListMessage() != nil:
		return msg.GetListMessage().GetContextInfo()
	case msg.GetListResponseMessage() != nil:
		return msg.GetListResponseMessage().GetContextInfo()
	case msg.GetTemplateMessage() != nil:
		return msg.GetTemplateMessage().GetContextInfo()
	case msg.GetTemplateButtonReplyMessage() != nil:
		return msg.GetTemplateButtonReplyMessage().GetContextInfo()
	case msg.GetPollCreationMessage() != nil:
		return msg.GetPollCreationMessage().GetContextInfo()
	case msg.GetGroupInviteMessage() != nil:
		return msg.GetGroupInviteMessage().GetContextInfo()
	case msg.GetProductMessage() != nil:
		return msg.GetProductMessage().GetContextInfo()
	case msg.GetOrderMessage() != nil:
		return msg.GetOrderMessage().GetContextInfo()
	case msg.GetEventMessage() != nil:
		return msg.GetEventMessage().GetContextInfo()
	case msg.GetEventInviteMessage() != nil:
		return msg.GetEventInviteMessage().GetContextInfo()
	}
	// Wrappers descem um nível: ephemeral / viewOnce / viewOnceV2 /
	// documentWithCaption embrulham outra Message. Recursão raso porque o
	// whatsmeow não aninha mais que isso.
	if eph := msg.GetEphemeralMessage(); eph != nil {
		return ExtractContextInfo(eph.GetMessage())
	}
	if vo := msg.GetViewOnceMessage(); vo != nil {
		return ExtractContextInfo(vo.GetMessage())
	}
	if vo := msg.GetViewOnceMessageV2(); vo != nil {
		return ExtractContextInfo(vo.GetMessage())
	}
	if vo := msg.GetViewOnceMessageV2Extension(); vo != nil {
		return ExtractContextInfo(vo.GetMessage())
	}
	if doc := msg.GetDocumentWithCaptionMessage(); doc != nil {
		return ExtractContextInfo(doc.GetMessage())
	}
	return nil
}

// ExtractForwarding lê IsForwarded/ForwardingScore do ContextInfo da mensagem
// recebida. Score zero quando IsForwarded é falso. WhatsApp considera score >=
// 5 como "encaminhada várias vezes" (frequently forwarded).
func ExtractForwarding(msg *waE2E.Message) (isForwarded bool, score uint32) {
	ci := ExtractContextInfo(msg)
	if ci == nil {
		return false, 0
	}
	return ci.GetIsForwarded(), ci.GetForwardingScore()
}

// InferChatType infers the chat type from a JID string.
func InferChatType(chatJID string) string {
	switch {
	case strings.HasPrefix(chatJID, "status@"):
		return "status"
	case strings.HasSuffix(chatJID, "@g.us"):
		return "group"
	case strings.HasSuffix(chatJID, "@broadcast"):
		return "broadcast"
	case strings.Contains(chatJID, "@newsletter"):
		return "newsletter"
	case chatJID == "":
		return "unknown"
	default:
		return "direct"
	}
}
