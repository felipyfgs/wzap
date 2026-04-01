package dto

type LabelChatReq struct {
	JID     string `json:"jid" validate:"required"`
	LabelID string `json:"labelId" validate:"required"`
}

type LabelMessageReq struct {
	JID       string `json:"jid" validate:"required"`
	LabelID   string `json:"labelId" validate:"required"`
	MessageID string `json:"mid" validate:"required"`
}

type EditLabelReq struct {
	Color   int32  `json:"color"`
	Deleted bool   `json:"deleted"`
	LabelID string `json:"labelId" validate:"required"`
	Name    string `json:"name"`
}
