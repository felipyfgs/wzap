package dto

type CreateGroupReq struct {
	Name         string   `json:"name" validate:"required,min=1" example:"My Awesome Group"`
	Participants []string `json:"participants" validate:"required,min=1" example:"5511999999999,5511888888888"`
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

type GroupJoinActionReq struct {
	GroupJID     string   `json:"groupJid" validate:"required"`
	Participants []string `json:"participants" validate:"required" example:"5511999999999"`
	Action       string   `json:"action" validate:"required" example:"approve"`
}

type GroupTextReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Text     string `json:"text" validate:"required"`
}

type GroupPhotoReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Image    string `json:"image" validate:"required"`
}

type GroupSettingReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Enabled  bool   `json:"enabled"`
}

type GroupJIDReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
}

type GroupEphemeralReq struct {
	GroupJID string `json:"groupJid" validate:"required"`
	Duration int    `json:"duration" validate:"required"`
}

type GroupMemberResp struct {
	JID          string `json:"jid"`
	PhoneNumber  string `json:"phoneNumber,omitempty"`
	LID          string `json:"lid,omitempty"`
	IsAdmin      bool   `json:"isAdmin"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
	DisplayName  string `json:"displayName,omitempty"`
}

type SubgroupResp struct {
	JID  string `json:"jid"`
	Name string `json:"name,omitempty"`
}

type GroupDetailResp struct {
	JID            string            `json:"jid"`
	Name           string            `json:"name"`
	Topic          string            `json:"topic,omitempty"`
	IsAdmin        bool              `json:"isAdmin"`
	IsParent       bool              `json:"isParent"`
	IsLocked       bool              `json:"isLocked"`
	IsAnnounce     bool              `json:"isAnnounce"`
	JoinApproval   bool              `json:"joinApproval"`
	IsEphemeral    bool              `json:"isEphemeral"`
	EphemeralTimer uint32            `json:"ephemeralTimer"`
	Participants   []GroupMemberResp `json:"participants"`
	Subgroups      []SubgroupResp    `json:"subgroups,omitempty"`
	CreatedAt      string            `json:"createdAt,omitempty"`
}
