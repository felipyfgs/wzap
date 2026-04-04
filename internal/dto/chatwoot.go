package dto

type ChatwootConfigReq struct {
	URL                 string   `json:"url" validate:"required,url"`
	AccountID           int      `json:"accountId" validate:"required,gt=0"`
	Token               string   `json:"token" validate:"required"`
	InboxName           string   `json:"inboxName,omitempty"`
	SignMsg             *bool    `json:"signMsg,omitempty"`
	SignDelimiter       string   `json:"signDelimiter,omitempty"`
	ReopenConversation  *bool    `json:"reopenConversation,omitempty"`
	MergeBRContacts     *bool    `json:"mergeBrContacts,omitempty"`
	IgnoreGroups        *bool    `json:"ignoreGroups,omitempty"`
	IgnoreJIDs          []string `json:"ignoreJids,omitempty"`
	ConversationPending *bool    `json:"conversationPending,omitempty"`
	AutoCreateInbox     *bool    `json:"autoCreateInbox,omitempty"`
}

type ChatwootConfigResp struct {
	SessionID           string   `json:"sessionId"`
	URL                 string   `json:"url"`
	AccountID           int      `json:"accountId"`
	InboxID             int      `json:"inboxId"`
	InboxName           string   `json:"inboxName"`
	SignMsg             bool     `json:"signMsg"`
	SignDelimiter       string   `json:"signDelimiter"`
	ReopenConversation  bool     `json:"reopenConversation"`
	MergeBRContacts     bool     `json:"mergeBrContacts"`
	IgnoreGroups        bool     `json:"ignoreGroups"`
	IgnoreJIDs          []string `json:"ignoreJids"`
	ConversationPending bool     `json:"conversationPending"`
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
}

type ChatwootWebhookPayload struct {
	Identifier string `json:"identifier,omitempty"`
	Account    struct {
		ID int `json:"id"`
	} `json:"account"`
	Inbox struct {
		ID int `json:"id"`
	} `json:"inbox"`
	EventType string `json:"event_type"`
	Message   *struct {
		ID                int            `json:"id"`
		Content           string         `json:"content,omitempty"`
		MessageType       string         `json:"message_type"`
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
		CreatedAt int64 `json:"created_at"`
	} `json:"message,omitempty"`
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
		} `json:"messages,omitempty"`
		Meta struct {
			Sender struct {
				ID                   int            `json:"id"`
				Name                 string         `json:"name"`
				PhoneNumber          string         `json:"phone_number,omitempty"`
				AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
			} `json:"sender"`
		} `json:"meta"`
	} `json:"conversation,omitempty"`
}
