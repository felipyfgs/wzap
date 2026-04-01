package model

type MessageType string

const (
	MsgTypeText     MessageType = "text"
	MsgTypeImage    MessageType = "image"
	MsgTypeVideo    MessageType = "video"
	MsgTypeAudio    MessageType = "audio"
	MsgTypeDocument MessageType = "document"
)
