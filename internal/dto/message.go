package dto

type SendTextReq struct {
	Phone string `json:"phone" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

type SendMediaReq struct {
	Phone    string `json:"phone" validate:"required"`
	MimeType string `json:"mimeType" validate:"required"`
	Caption  string `json:"caption"`
	FileName string `json:"fileName"`
	Base64   string `json:"base64"`
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
	State string `json:"state" validate:"required"`
}
