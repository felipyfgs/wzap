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
	Phone string `json:"phone" validate:"required"`
}

type GetAvatarResp struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

type BlockContactReq struct {
	Phone string `json:"phone" validate:"required"`
}

type GetUserInfoReq struct {
	Phones []string `json:"phones" validate:"required"`
}

type UserInfoResp struct {
	JID     string   `json:"jid"`
	Status  string   `json:"status"`
	Picture string   `json:"picture"`
	Devices []string `json:"devices"`
}

type UpdateAvatarReq struct {
	Image string `json:"image" validate:"required"`
}

type SubscribePresenceReq struct {
	Phone string `json:"phone" validate:"required"`
}

type SetPrivacyReq struct {
	Setting string `json:"setting" validate:"required"`
	Value   string `json:"value" validate:"required"`
}

type UpdateStatusReq struct {
	Status string `json:"status" validate:"required"`
}
