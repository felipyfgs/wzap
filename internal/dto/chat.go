package dto

type ChatActionReq struct {
	JID string `json:"jid" validate:"required"`
}

type ChatMarkReadReq struct {
	JID        string   `json:"jid" validate:"required"`
	MessageIDs []string `json:"messageIds" validate:"required"`
}

type ChatMarkUnreadReq struct {
	JID string `json:"jid" validate:"required"`
}
