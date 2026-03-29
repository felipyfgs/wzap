package model

type CreateGroupReq struct {
	Name         string   `json:"name" example:"My Awesome Group"`
	Participants []string `json:"participants" example:"5511999999999,5511888888888"` // list of phone numbers
}

type GroupInfoReq struct {
	JID string `query:"jid" validate:"required" example:"123456789@g.us"`
}

type GroupInviteLinkResp struct {
	Link string `json:"link"`
}

type GroupJoinReq struct {
	InviteCode string `json:"invite_code" validate:"required"`
}

type GroupParticipantReq struct {
	GroupJID     string   `json:"groupJid" validate:"required"`
	Participants []string `json:"participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"action" validate:"required" example:"add"` // add, remove, promote, demote
}

type GroupRequestActionReq struct {
	GroupJID     string   `json:"groupJid" validate:"required"`
	Participants []string `json:"participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"action" validate:"required" example:"approve"` // approve, reject
}

type GroupTextReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Text     string `json:"text" validate:"required"`
}

type GroupPhotoReq struct {
	GroupJID    string `json:"groupJid" validate:"required"`
	PhotoBase64 string `json:"photo_base64" validate:"required"` // Base64 encoded image
}

type GroupSettingReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Enabled  bool   `json:"enabled"`
}

type JIDReq struct {
}

// GroupJIDReq standard generic group request
type GroupJIDReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
}
