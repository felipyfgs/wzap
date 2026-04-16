package dto

type CreateCommunityReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type CommunityMembersReq struct {
	CommunityJID string   `json:"communityJid" validate:"required"`
	Participants []string `json:"participants" validate:"required"`
}
