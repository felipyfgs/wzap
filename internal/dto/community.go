package dto

type CreateCommunityReq struct {
	Name        string `json:"Name" validate:"required"`
	Description string `json:"Description"`
}

type CommunityParticipantReq struct {
	CommunityJID string   `json:"CommunityJid" validate:"required"`
	Participants []string `json:"Participants" validate:"required"`
}
