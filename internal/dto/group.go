package dto

type CreateGroupReq struct {
	Name         string   `json:"name" example:"My Awesome Group"`
	Participants []string `json:"participants" example:"5511999999999,5511888888888"`
}

type GroupInfoReq struct {
	JID string `query:"jid" validate:"required" example:"123456789@g.us"`
}

type GroupInviteLinkResp struct {
	Link string `json:"link"`
}

type GroupJoinReq struct {
	InviteCode string `json:"inviteCode" validate:"required"`
}

type GroupParticipantReq struct {
	GroupJID     string   `json:"groupJid" validate:"required"`
	Participants []string `json:"participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"action" validate:"required" example:"add"`
}

type GroupRequestActionReq struct {
	GroupJID     string   `json:"groupJid" validate:"required"`
	Participants []string `json:"participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"action" validate:"required" example:"approve"`
}

type GroupTextReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Text     string `json:"text" validate:"required"`
}

type GroupPhotoReq struct {
	GroupJID    string `json:"groupJid" validate:"required"`
	PhotoBase64 string `json:"photoBase64" validate:"required"`
}

type GroupSettingReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Enabled  bool   `json:"enabled"`
}

type GroupJIDReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
}
