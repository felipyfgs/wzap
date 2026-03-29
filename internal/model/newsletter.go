package model

type CreateNewsletterReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Picture     string `json:"picture"` // base64
}

type NewsletterMessageReq struct {
	JID      string `json:"jid" validate:"required"`
	Count    int    `json:"count"`
	BeforeID int    `json:"before_id"`
}
