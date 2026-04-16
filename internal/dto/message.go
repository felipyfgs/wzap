package dto

type ReplyContext struct {
	MessageID    string   `json:"messageId,omitempty"`
	Participant  string   `json:"participant,omitempty"`
	MentionedJID []string `json:"mentionedJid,omitempty"`
}

type SendTextReq struct {
	Phone         string        `json:"phone" validate:"required"`
	Body          string        `json:"body" validate:"required"`
	CustomID      string        `json:"customId,omitempty"`
	ReplyTo       *ReplyContext `json:"replyTo,omitempty"`
	MentionedJIDs []string      `json:"mentionedJids,omitempty"`
}

type SendMediaReq struct {
	Phone         string        `json:"phone" validate:"required"`
	MimeType      string        `json:"mimeType" validate:"required"`
	Caption       string        `json:"caption"`
	FileName      string        `json:"fileName"`
	Base64        string        `json:"base64" validate:"required_without=URL"`
	URL           string        `json:"url,omitempty" validate:"required_without=Base64"`
	CustomID      string        `json:"customId,omitempty"`
	ReplyTo       *ReplyContext `json:"replyTo,omitempty"`
	MentionedJIDs []string      `json:"mentionedJids,omitempty"`
}

type SendButtonReq struct {
	Phone         string        `json:"phone" validate:"required"`
	Body          string        `json:"body" validate:"required"`
	Footer        string        `json:"footer,omitempty"`
	Buttons       []ButtonItem  `json:"buttons" validate:"required,min=1"`
	CustomID      string        `json:"customId,omitempty"`
	ReplyTo       *ReplyContext `json:"replyTo,omitempty"`
	MentionedJIDs []string      `json:"mentionedJids,omitempty"`
}

type ButtonItem struct {
	ID   string `json:"id" validate:"required"`
	Text string `json:"text" validate:"required"`
}

type SendListReq struct {
	Phone         string        `json:"phone" validate:"required"`
	Title         string        `json:"title" validate:"required"`
	Body          string        `json:"body" validate:"required"`
	Footer        string        `json:"footer,omitempty"`
	ButtonText    string        `json:"buttonText" validate:"required"`
	Sections      []ListSection `json:"sections" validate:"required,min=1"`
	CustomID      string        `json:"customId,omitempty"`
	ReplyTo       *ReplyContext `json:"replyTo,omitempty"`
	MentionedJIDs []string      `json:"mentionedJids,omitempty"`
}

type ListSection struct {
	Title string    `json:"title"`
	Rows  []ListRow `json:"rows" validate:"required,min=1"`
}

type ListRow struct {
	ID          string `json:"id" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
}

type SendContactReq struct {
	Phone string `json:"phone" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Vcard string `json:"vcard" validate:"required"`
}

type SendLocationReq struct {
	Phone     string  `json:"phone" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
}

type SendPollReq struct {
	Phone           string   `json:"phone" validate:"required"`
	Name            string   `json:"name" validate:"required"`
	Options         []string `json:"options" validate:"required,min=2"`
	SelectableCount int      `json:"selectableCount"`
}

type SendStickerReq struct {
	Phone    string `json:"phone" validate:"required"`
	MimeType string `json:"mimeType" validate:"required"`
	Base64   string `json:"base64" validate:"required"`
}

type SendLinkReq struct {
	Phone       string `json:"phone" validate:"required"`
	URL         string `json:"url" validate:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type EditMessageReq struct {
	Phone     string `json:"phone" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
	Body      string `json:"body" validate:"required"`
}

type DeleteMessageReq struct {
	Phone     string `json:"phone" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
}

type ReactMessageReq struct {
	Phone     string `json:"phone" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
	Reaction  string `json:"reaction" validate:"required"`
}

type MarkReadReq struct {
	Phone     string `json:"phone" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
}

type SetPresenceReq struct {
	Phone string `json:"phone" validate:"required"`
	State string `json:"state" validate:"required,oneof=typing recording paused"`
}

type ForwardMessageReq struct {
	MessageID string `json:"messageId" validate:"required"`
	FromJID   string `json:"fromJid" validate:"required"`
	Phone     string `json:"phone" validate:"required"`
}
