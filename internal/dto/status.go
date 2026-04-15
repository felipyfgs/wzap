package dto

type StatusTextReq struct {
	Text            string `json:"text" validate:"required,min=1"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Font            *int   `json:"font,omitempty"`
}

type StatusMediaReq struct {
	Base64   string `json:"base64,omitempty"`
	URL      string `json:"url,omitempty"`
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mimeType,omitempty" validate:"required,min=1"`
}
