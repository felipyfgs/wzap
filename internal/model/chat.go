package model

type ChatActionReq struct {
	JID string `json:"jid" validate:"required"`
}
