package dto

type CreateNewsletterReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Picture     string `json:"picture"`
}

type NewsletterMessageReq struct {
	NewsletterJID string `json:"newsletterJid" validate:"required"`
	Count         int    `json:"count"`
	BeforeID      int    `json:"beforeId"`
}

type NewsletterSubscribeReq struct {
	NewsletterJID string `json:"newsletterJid" validate:"required"`
}

type NewsletterMuteReq struct {
	NewsletterJID string `json:"newsletterJid" validate:"required"`
	Mute          bool   `json:"mute"`
}

type NewsletterReactReq struct {
	JID       string `json:"newsletterJid" validate:"required"`
	ServerID  int    `json:"serverId" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
	Reaction  string `json:"reaction" validate:"required"`
}

type NewsletterMarkViewedReq struct {
	JID       string `json:"newsletterJid" validate:"required"`
	ServerIDs []int  `json:"serverIds" validate:"required"`
}
