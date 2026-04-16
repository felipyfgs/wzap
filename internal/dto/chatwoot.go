package dto

type ChatwootConfigReq struct {
	URL                 string   `json:"url" validate:"required,url"`
	AccountID           int      `json:"accountId" validate:"required,gt=0"`
	Token               string   `json:"token" validate:"required"`
	WebhookToken        string   `json:"webhookToken,omitempty"`
	InboxID             int      `json:"inboxId,omitempty"`
	InboxName           string   `json:"inboxName,omitempty"`
	InboxType           *string  `json:"inboxType,omitempty" validate:"omitempty,oneof=api cloud"`
	SignMsg             *bool    `json:"signMsg,omitempty"`
	SignDelimiter       string   `json:"signDelimiter,omitempty"`
	ReopenConversation  *bool    `json:"reopenConversation,omitempty"`
	MergeBRContacts     *bool    `json:"mergeBrContacts,omitempty"`
	IgnoreGroups        *bool    `json:"ignoreGroups,omitempty"`
	IgnoreJIDs          []string `json:"ignoreJids,omitempty"`
	ConversationPending *bool    `json:"conversationPending,omitempty"`
	ImportOnConnect     *bool    `json:"importOnConnect,omitempty"`
	ImportPeriod        string   `json:"importPeriod,omitempty"`
	TimeoutTextSeconds  *int     `json:"timeoutTextSeconds,omitempty"`
	TimeoutMediaSeconds *int     `json:"timeoutMediaSeconds,omitempty"`
	TimeoutLargeSeconds *int     `json:"timeoutLargeSeconds,omitempty"`
	MessageRead         *bool    `json:"messageRead,omitempty"`
	DatabaseURI         string   `json:"databaseUri,omitempty"`
	RedisURL            string   `json:"redisUrl,omitempty"`
}

type ChatwootConfigResp struct {
	SessionID           string   `json:"sessionId"`
	URL                 string   `json:"url"`
	AccountID           int      `json:"accountId"`
	InboxID             int      `json:"inboxId"`
	InboxName           string   `json:"inboxName"`
	InboxType           string   `json:"inboxType"`
	SignMsg             bool     `json:"signMsg"`
	SignDelimiter       string   `json:"signDelimiter"`
	ReopenConversation  bool     `json:"reopenConversation"`
	MergeBRContacts     bool     `json:"mergeBrContacts"`
	IgnoreGroups        bool     `json:"ignoreGroups"`
	IgnoreJIDs          []string `json:"ignoreJids"`
	ConversationPending bool     `json:"conversationPending"`
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	ImportOnConnect     bool     `json:"importOnConnect"`
	ImportPeriod        string   `json:"importPeriod"`
	TimeoutTextSeconds  int      `json:"timeoutTextSeconds"`
	TimeoutMediaSeconds int      `json:"timeoutMediaSeconds"`
	TimeoutLargeSeconds int      `json:"timeoutLargeSeconds"`
	MessageRead         bool     `json:"messageRead"`
	DatabaseURI         string   `json:"databaseUri,omitempty"`
	RedisURL            string   `json:"redisUrl,omitempty"`
}

type ImportHistoryReq struct {
	Period     string `json:"period" validate:"required,oneof=24h 7d 30d custom"`
	CustomDays int    `json:"customDays,omitempty"`
}

type ImportHistoryResp struct {
	SessionID string `json:"sessionId"`
	Period    string `json:"period"`
	Status    string `json:"status"`
}

type ChatwootWebhookMessage struct {
	ID                int            `json:"id"`
	Content           string         `json:"content,omitempty"`
	MessageType       any            `json:"message_type"`
	Private           bool           `json:"private,omitempty"`
	ContentType       string         `json:"content_type,omitempty"`
	ContentAttributes map[string]any `json:"content_attributes,omitempty"`
	SourceID          string         `json:"source_id,omitempty"`
	InboxID           int            `json:"inbox_id"`
	ConversationID    int            `json:"conversation_id"`
	Sender            *struct {
		ID                   int            `json:"id"`
		Name                 string         `json:"name"`
		PhoneNumber          string         `json:"phone_number,omitempty"`
		AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	} `json:"sender,omitempty"`
	Attachments []struct {
		ID        int    `json:"id,omitempty"`
		MessageID int    `json:"message_id,omitempty"`
		FileType  string `json:"file_type"`
		AccountID int    `json:"account_id"`
		Extension string `json:"extension,omitempty"`
		DataURL   string `json:"data_url"`
		URL       string `json:"url"`
		SenderID  int    `json:"sender_id,omitempty"`
		ThumbURL  string `json:"thumb_url,omitempty"`
		FileSize  int    `json:"file_size,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	} `json:"attachments,omitempty"`
	CreatedAt any `json:"created_at"`
}

func (m *ChatwootWebhookMessage) IsOutgoing() bool {
	switch v := m.MessageType.(type) {
	case string:
		return v == "outgoing"
	case float64:
		return int(v) == 1
	}
	return false
}

type ChatwootWebhookPayload struct {
	ChatwootWebhookMessage

	Identifier string `json:"identifier,omitempty"`
	Account    struct {
		ID int `json:"id"`
	} `json:"account"`
	Inbox struct {
		ID int `json:"id"`
	} `json:"inbox"`
	EventType string                  `json:"event_type"`
	Event     string                  `json:"event,omitempty"`
	Private   bool                    `json:"private,omitempty"`
	Message   *ChatwootWebhookMessage `json:"message,omitempty"`

	Conversation *struct {
		ID           int `json:"id"`
		ContactInbox struct {
			SourceID string `json:"source_id"`
		} `json:"contact_inbox"`
		ContactID int    `json:"contact_id"`
		InboxID   int    `json:"inbox_id"`
		Status    string `json:"status"`
		Messages  []struct {
			ID      int    `json:"id"`
			Content string `json:"content,omitempty"`
			Sender  *struct {
				AvailableName string `json:"available_name,omitempty"`
			} `json:"sender,omitempty"`
		} `json:"messages,omitempty"`
		Meta struct {
			Sender struct {
				ID                   int            `json:"id"`
				Name                 string         `json:"name"`
				Identifier           string         `json:"identifier,omitempty"`
				PhoneNumber          string         `json:"phone_number,omitempty"`
				AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
			} `json:"sender"`
		} `json:"meta"`
	} `json:"conversation,omitempty"`
}

func (p *ChatwootWebhookPayload) GetMessage() *ChatwootWebhookMessage {
	if p.Message != nil {
		return p.Message
	}
	if p.ID != 0 {
		return &p.ChatwootWebhookMessage
	}
	return nil
}
