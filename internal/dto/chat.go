package dto

type ChatActionReq struct {
	JID string `json:"jid" validate:"required"`
}
