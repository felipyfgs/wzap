package dto

type SendTextReq struct {
	JID  string `json:"jid" validate:"required"`
	Text string `json:"text" validate:"required"`
}

type SendMediaReq struct {
	JID      string `json:"jid" validate:"required"`
	MimeType string `json:"mimeType" validate:"required"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"`
	Base64   string `json:"base64"`
}

type SendContactReq struct {
	JID   string `json:"jid" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Vcard string `json:"vcard" validate:"required"`
}

type SendLocationReq struct {
	JID     string  `json:"jid" validate:"required"`
	Lat     float64 `json:"lat" validate:"required"`
	Lng     float64 `json:"lng" validate:"required"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
}

type SendPollReq struct {
	JID             string   `json:"jid" validate:"required"`
	Name            string   `json:"name" validate:"required"`
	Options         []string `json:"options" validate:"required,min=2"`
	SelectableCount int      `json:"selectableCount"`
}

type SendStickerReq struct {
	JID      string `json:"jid" validate:"required"`
	MimeType string `json:"mimeType" validate:"required"`
	Base64   string `json:"base64" validate:"required"`
}

type SendLinkReq struct {
	JID         string `json:"jid" validate:"required"`
	URL         string `json:"url" validate:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type EditMessageReq struct {
	JID       string `json:"jid" validate:"required"`
	MessageID string `json:"mid" validate:"required"`
	Text      string `json:"text" validate:"required"`
}

type DeleteMessageReq struct {
	JID       string `json:"jid" validate:"required"`
	MessageID string `json:"mid" validate:"required"`
}

type ReactMessageReq struct {
	JID       string `json:"jid" validate:"required"`
	MessageID string `json:"mid" validate:"required"`
	Reaction  string `json:"reaction" validate:"required"`
}

type MarkReadReq struct {
	JID       string `json:"jid" validate:"required"`
	MessageID string `json:"mid" validate:"required"`
}

type SetPresenceReq struct {
	JID      string `json:"jid" validate:"required"`
	Presence string `json:"presence" validate:"required"`
}
