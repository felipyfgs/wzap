package model

import "time"

type Chat struct {
	SessionID          string     `json:"sessionId"`
	ChatJID            string     `json:"chatJid"`
	Name               string     `json:"name,omitempty"`
	DisplayName        string     `json:"displayName,omitempty"`
	ChatType           string     `json:"chatType,omitempty"`
	Archived           bool       `json:"archived"`
	Pinned             int        `json:"pinned,omitempty"`
	ReadOnly           bool       `json:"readOnly"`
	MarkedAsUnread     bool       `json:"markedAsUnread"`
	UnreadCount        int        `json:"unreadCount,omitempty"`
	UnreadMentionCount int        `json:"unreadMentionCount,omitempty"`
	LastMessageID      string     `json:"lastMessageId,omitempty"`
	LastMessageAt      *time.Time `json:"lastMessageAt,omitempty"`
	ConvTimestamp      *time.Time `json:"convTimestamp,omitempty"`
	PnJID              string     `json:"pnJid,omitempty"`
	LidJID             string     `json:"lidJid,omitempty"`
	Username           string     `json:"username,omitempty"`
	AccountLID         string     `json:"accountLid,omitempty"`
	Source             string     `json:"source"`
	SyncType           string     `json:"syncType,omitempty"`
	ChunkOrder         *int       `json:"chunkOrder,omitempty"`
	Raw                any        `json:"raw,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type ChatUpdate struct {
	SessionID          string
	ChatJID            string
	Name               *string
	DisplayName        *string
	ChatType           *string
	Archived           *bool
	Pinned             *int
	ReadOnly           *bool
	MarkedAsUnread     *bool
	UnreadCount        *int
	UnreadMentionCount *int
	LastMessageID      *string
	LastMessageAt      *time.Time
	ConvTimestamp      *time.Time
	PnJID              *string
	LidJID             *string
	Username           *string
	AccountLID         *string
	Source             string
	SyncType           *string
	ChunkOrder         *int
	Raw                any
}
