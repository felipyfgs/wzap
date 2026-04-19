package dto

type CWConfigReq struct {
	URL             string   `json:"url" validate:"required,url"`
	AccountID       int      `json:"accountId" validate:"required,gt=0"`
	Token           string   `json:"token" validate:"required"`
	WebhookToken    string   `json:"webhookToken,omitempty"`
	InboxID         int      `json:"inboxId,omitempty"`
	InboxName       string   `json:"inboxName,omitempty"`
	SignMsg         *bool    `json:"signMsg,omitempty"`
	SignDelimiter   string   `json:"signDelimiter,omitempty"`
	ReopenConv      *bool    `json:"reopenConv,omitempty"`
	MergeBRContacts *bool    `json:"mergeBrContacts,omitempty"`
	IgnoreGroups    *bool    `json:"ignoreGroups,omitempty"`
	IgnoreJIDs      []string `json:"ignoreJids,omitempty"`
	PendingConv     *bool    `json:"pendingConv,omitempty"`
	ImportOnConnect *bool    `json:"importOnConnect,omitempty"`
	ImportPeriod    string   `json:"importPeriod,omitempty"`
	TextTimeout     *int     `json:"textTimeout,omitempty"`
	MediaTimeout    *int     `json:"mediaTimeout,omitempty"`
	LargeTimeout    *int     `json:"largeTimeout,omitempty"`
	MessageRead     *bool    `json:"messageRead,omitempty"`
}

type CWConfigResp struct {
	SessionID       string   `json:"sessionId"`
	URL             string   `json:"url"`
	AccountID       int      `json:"accountId"`
	InboxID         int      `json:"inboxId"`
	InboxName       string   `json:"inboxName"`
	SignMsg         bool     `json:"signMsg"`
	SignDelimiter   string   `json:"signDelimiter"`
	ReopenConv      bool     `json:"reopenConv"`
	MergeBRContacts bool     `json:"mergeBrContacts"`
	IgnoreGroups    bool     `json:"ignoreGroups"`
	IgnoreJIDs      []string `json:"ignoreJids"`
	PendingConv     bool     `json:"pendingConv"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhookUrl"`
	ImportOnConnect bool     `json:"importOnConnect"`
	ImportPeriod    string   `json:"importPeriod"`
	TextTimeout     int      `json:"textTimeout"`
	MediaTimeout    int      `json:"mediaTimeout"`
	LargeTimeout    int      `json:"largeTimeout"`
	MessageRead     bool     `json:"messageRead"`
}

type CWImportReq struct {
	Period     string `json:"period" validate:"required,oneof=24h 7d 30d custom"`
	CustomDays int    `json:"customDays,omitempty"`
}

type CWImportResp struct {
	SessionID string `json:"sessionId"`
	Period    string `json:"period"`
	Status    string `json:"status"`
}

type CWWebhookMsg struct {
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

func (m *CWWebhookMsg) IsOutgoing() bool {
	switch v := m.MessageType.(type) {
	case string:
		return v == "outgoing"
	case float64:
		return int(v) == 1
	}
	return false
}

type CWWebhookPayload struct {
	CWWebhookMsg

	Identifier string `json:"identifier,omitempty"`
	Account    struct {
		ID int `json:"id"`
	} `json:"account"`
	Inbox struct {
		ID int `json:"id"`
	} `json:"inbox"`
	EventType string        `json:"event_type"`
	Event     string        `json:"event,omitempty"`
	Private   bool          `json:"private,omitempty"`
	Message   *CWWebhookMsg `json:"message,omitempty"`

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

func (p *CWWebhookPayload) GetMessage() *CWWebhookMsg {
	if p.Message != nil {
		return p.Message
	}
	if p.ID != 0 {
		return &p.CWWebhookMsg
	}
	return nil
}
