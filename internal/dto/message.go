package dto

type SendTextReq struct {
	Phone string `json:"Phone" validate:"required"`
	Body  string `json:"Body" validate:"required"`
}

type SendMediaReq struct {
	Phone    string `json:"Phone" validate:"required"`
	MimeType string `json:"MimeType" validate:"required"`
	Caption  string `json:"Caption"`
	FileName string `json:"FileName"`
	Base64   string `json:"Base64"`
}

type SendContactReq struct {
	Phone string `json:"Phone" validate:"required"`
	Name  string `json:"Name" validate:"required"`
	Vcard string `json:"Vcard" validate:"required"`
}

type SendLocationReq struct {
	Phone     string  `json:"Phone" validate:"required"`
	Latitude  float64 `json:"Latitude" validate:"required"`
	Longitude float64 `json:"Longitude" validate:"required"`
	Name      string  `json:"Name"`
	Address   string  `json:"Address"`
}

type SendPollReq struct {
	Phone           string   `json:"Phone" validate:"required"`
	Name            string   `json:"Name" validate:"required"`
	Options         []string `json:"Options" validate:"required,min=2"`
	SelectableCount int      `json:"SelectableCount"`
}

type SendStickerReq struct {
	Phone    string `json:"Phone" validate:"required"`
	MimeType string `json:"MimeType" validate:"required"`
	Base64   string `json:"Base64" validate:"required"`
}

type SendLinkReq struct {
	Phone       string `json:"Phone" validate:"required"`
	URL         string `json:"URL" validate:"required"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
}

type EditMessageReq struct {
	Phone     string `json:"Phone" validate:"required"`
	MessageID string `json:"Mid" validate:"required"`
	Body      string `json:"Body" validate:"required"`
}

type DeleteMessageReq struct {
	Phone     string `json:"Phone" validate:"required"`
	MessageID string `json:"Mid" validate:"required"`
}

type ReactMessageReq struct {
	Phone     string `json:"Phone" validate:"required"`
	MessageID string `json:"Mid" validate:"required"`
	Reaction  string `json:"Reaction" validate:"required"`
}

type MarkReadReq struct {
	Phone     string `json:"Phone" validate:"required"`
	MessageID string `json:"Mid" validate:"required"`
}

type SetPresenceReq struct {
	Phone string `json:"Phone" validate:"required"`
	State string `json:"State" validate:"required"`
}
