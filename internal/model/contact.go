package model

type Contact struct {
	JID        string `json:"Jid"`
	Name       string `json:"Name,omitempty"`
	PushName   string `json:"PushName,omitempty"`
	Picture    string `json:"Picture,omitempty"`
	IsBusiness bool   `json:"IsBusiness"`
}

type Group struct {
	JID          string `json:"Jid"`
	Name         string `json:"Name"`
	Participants int    `json:"Participants"`
	IsAdmin      bool   `json:"IsAdmin"`
}
