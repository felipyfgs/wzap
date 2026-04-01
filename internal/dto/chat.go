package dto

type ChatActionReq struct {
	JID string `json:"Jid" validate:"required"`
}
