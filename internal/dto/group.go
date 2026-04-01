package dto

type CreateGroupReq struct {
	Name         string   `json:"Name" example:"My Awesome Group"`
	Participants []string `json:"Participants" example:"5511999999999,5511888888888"`
}

type GroupInviteLinkResp struct {
	Link string `json:"Link"`
}

type GroupJoinReq struct {
	InviteCode string `json:"InviteCode" validate:"required"`
}

type GroupParticipantReq struct {
	GroupJID     string   `json:"GroupJid" validate:"required"`
	Participants []string `json:"Participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"Action" validate:"required" example:"add"`
}

type GroupRequestActionReq struct {
	GroupJID     string   `json:"GroupJid" validate:"required"`
	Participants []string `json:"Participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"Action" validate:"required" example:"approve"`
}

type GroupTextReq struct {
	GroupJID string `json:"GroupJid" validate:"required"`
	Text     string `json:"Text" validate:"required"`
}

type GroupPhotoReq struct {
	GroupJID string `json:"GroupJid" validate:"required"`
	Image    string `json:"Image" validate:"required"`
}

type GroupSettingReq struct {
	GroupJID string `json:"GroupJid" validate:"required"`
	Enabled  bool   `json:"Enabled"`
}

type GroupJIDReq struct {
	GroupJID string `json:"GroupJid" validate:"required"`
}
