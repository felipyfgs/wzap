package dto

type CheckContactReq struct {
	Phones []string `json:"Phones" validate:"required"`
}

type CheckContactResp struct {
	Exists      bool   `json:"Exists"`
	JID         string `json:"Jid,omitempty"`
	PhoneNumber string `json:"PhoneNumber"`
}

type GetAvatarReq struct {
	Phone string `json:"Phone" validate:"required"`
}

type GetAvatarResp struct {
	URL string `json:"URL"`
	ID  string `json:"Id"`
}

type BlockContactReq struct {
	Phone string `json:"Phone" validate:"required"`
}

type GetUserInfoReq struct {
	Phones []string `json:"Phones" validate:"required"`
}

type UserInfoResp struct {
	JID     string   `json:"Jid"`
	Status  string   `json:"Status"`
	Picture string   `json:"Picture"`
	Devices []string `json:"Devices"`
}

type SetProfilePictureReq struct {
	Image string `json:"Image" validate:"required"`
}
