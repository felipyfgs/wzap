package dto

type CheckContactReq struct {
	Phones []string `json:"phones" validate:"required"`
}

type CheckContactResp struct {
	Exists      bool   `json:"exists"`
	JID         string `json:"jid,omitempty"`
	PhoneNumber string `json:"phoneNumber"`
}

type GetAvatarReq struct {
	JID string `json:"jid" validate:"required"`
}

type GetAvatarResp struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

type BlockContactReq struct {
	JID string `json:"jid" validate:"required"`
}

type GetUserInfoReq struct {
	JIDs []string `json:"jids" validate:"required"`
}

type UserInfoResp struct {
	JID     string   `json:"jid"`
	Status  string   `json:"status"`
	Picture string   `json:"picture"`
	Devices []string `json:"devices"`
}

type SetProfilePictureReq struct {
	Base64 string `json:"base64" validate:"required"`
}
