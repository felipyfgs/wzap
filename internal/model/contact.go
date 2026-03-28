package model

type Contact struct {
	JID        string `json:"jid"`
	Name       string `json:"name,omitempty"`
	PushName   string `json:"push_name,omitempty"`
	Picture    string `json:"picture,omitempty"`
	IsBusiness bool   `json:"is_business"`
}

type CheckContactReq struct {
	Phones []string `json:"phones" validate:"required"`
}

type CheckContactResp struct {
	Exists      bool   `json:"exists"`
	JID         string `json:"jid,omitempty"`
	PhoneNumber string `json:"phone_number"`
}

type Group struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	Participants int    `json:"participants"`
	IsAdmin      bool   `json:"is_admin"`
}
