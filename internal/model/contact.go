package model

type Contact struct {
	JID        string `json:"jid"`
	Name       string `json:"name,omitempty"`
	PushName   string `json:"pushName,omitempty"`
	Picture    string `json:"picture,omitempty"`
	IsBusiness bool   `json:"isBusiness"`
}

type Group struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	Participants int    `json:"participants"`
	IsAdmin      bool   `json:"isAdmin"`
}
