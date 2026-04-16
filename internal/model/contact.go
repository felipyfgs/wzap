package model

type Contact struct {
	JID          string `json:"jid"`
	Name         string `json:"name,omitempty"`
	PushName     string `json:"pushName,omitempty"`
	BusinessName string `json:"businessName,omitempty"`
	Picture      string `json:"picture,omitempty"`
	IsBusiness   bool   `json:"isBusiness,omitempty"`
}

type Group struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	Participants int    `json:"participants"`
	Subgroups    int    `json:"subgroups,omitempty"`
	IsAdmin      bool   `json:"isAdmin"`
	IsParent     bool   `json:"isParent"`
}
