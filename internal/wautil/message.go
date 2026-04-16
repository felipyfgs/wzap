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
		return im.GetDirectPath(), im.GetFileEncSHA256(), im.GetFileSHA256(), im.GetMediaKey(), int(im.GetFileLength()), true
	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		return vm.GetDirectPath(), vm.GetFileEncSHA256(), vm.GetFileSHA256(), vm.GetMediaKey(), int(vm.GetFileLength()), true
	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		return am.GetDirectPath(), am.GetFileEncSHA256(), am.GetFileSHA256(), am.GetMediaKey(), int(am.GetFileLength()), true
	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		return dm.GetDirectPath(), dm.GetFileEncSHA256(), dm.GetFileSHA256(), dm.GetMediaKey(), int(dm.GetFileLength()), true
	case msg.GetStickerMessage() != nil:
		sm := msg.GetStickerMessage()
		return sm.GetDirectPath(), sm.GetFileEncSHA256(), sm.GetFileSHA256(), sm.GetMediaKey(), int(sm.GetFileLength()), true
	default:
		return "", nil, nil, nil, 0, false
	}
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
