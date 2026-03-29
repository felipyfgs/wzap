package model

type CreateCommunityReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type CommunityParticipantReq struct {
	JID          string   `json:"jid" validate:"required"`
	Participants []string `json:"participants" validate:"required"`
}
