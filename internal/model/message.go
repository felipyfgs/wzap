package model

import "time"

type Message struct {
	ID              string     `json:"id"`
	SessionID       string     `json:"sessionId"`
	ChatJID         string     `json:"chatJid"`
	SenderJID       string     `json:"senderJid"`
	FromMe          bool       `json:"fromMe"`
	MsgType         string     `json:"msgType"`
	Body            string     `json:"body"`
	MediaType       string     `json:"mediaType,omitempty"`
	MediaURL        string     `json:"mediaUrl,omitempty"`
	Source          string     `json:"source"`
	SyncType        string     `json:"syncType,omitempty"`
	ChunkOrder      *int       `json:"chunkOrder,omitempty"`
	MsgOrder        *int64     `json:"msgOrder,omitempty"`
	Raw             any        `json:"raw,omitempty"`
	IsForwarded     bool       `json:"isForwarded,omitempty"`
	ForwardingScore uint32     `json:"forwardingScore,omitempty"`
	Timestamp       time.Time  `json:"timestamp"`
	CreatedAt       time.Time  `json:"createdAt"`
	CWImportedAt    *time.Time `json:"cwImportedAt,omitempty"`

	CWMessageID *int    `json:"cwMessageId,omitempty"`
	CWConvID    *int    `json:"cwConvId,omitempty"`
	CWSrcID     *string `json:"cwSrcId,omitempty"`

	ElodeskMessageID *int64  `json:"elodeskMessageId,omitempty"`
	ElodeskConvID    *int64  `json:"elodeskConvId,omitempty"`
	ElodeskSrcID     *string `json:"elodeskSrcId,omitempty"`
}
