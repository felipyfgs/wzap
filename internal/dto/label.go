package dto

type LabelChatReq struct {
	JID     string `json:"Jid" validate:"required"`
	LabelID string `json:"LabelId" validate:"required"`
}

type LabelMessageReq struct {
	JID       string `json:"Jid" validate:"required"`
	LabelID   string `json:"LabelId" validate:"required"`
	MessageID string `json:"Mid" validate:"required"`
}

type EditLabelReq struct {
	Color   int32  `json:"Color"`
	Deleted bool   `json:"Deleted"`
	LabelID string `json:"LabelId" validate:"required"`
	Name    string `json:"Name"`
}
