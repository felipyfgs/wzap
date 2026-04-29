package dto

type ElodeskConfigReq struct {
	URL             string   `json:"url" validate:"required,url"`
	InboxIdentifier string   `json:"inboxIdentifier,omitempty"`
	APIToken        string   `json:"apiToken,omitempty"`
	HMACToken       string   `json:"hmacToken,omitempty"`
	WebhookSecret   string   `json:"webhookSecret,omitempty"`
	UserAccessToken string   `json:"userAccessToken,omitempty"`
	AccountID       *int     `json:"accountId,omitempty"`
	InboxName       string   `json:"inboxName,omitempty"`
	SignMsg         *bool    `json:"signMsg,omitempty"`
	SignDelimiter   string   `json:"signDelimiter,omitempty"`
	ReopenConv      *bool    `json:"reopenConv,omitempty"`
	MergeBRContacts *bool    `json:"mergeBrContacts,omitempty"`
	IgnoreGroups    *bool    `json:"ignoreGroups,omitempty"`
	IgnoreJIDs      []string `json:"ignoreJids,omitempty"`
	PendingConv     *bool    `json:"pendingConv,omitempty"`
	ImportOnConnect *bool    `json:"importOnConnect,omitempty"`
	ImportPeriod    string   `json:"importPeriod,omitempty" validate:"omitempty,oneof=24h 7d 30d custom"`
	TextTimeout     *int     `json:"textTimeout,omitempty"`
	MediaTimeout    *int     `json:"mediaTimeout,omitempty"`
	LargeTimeout    *int     `json:"largeTimeout,omitempty"`
	MessageRead     *bool    `json:"messageRead,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// ElodeskConfigResp é o shape exposto publicamente. Não inclui plaintext de
// api_token/hmac_token — apenas os booleans hasApiToken/hasHmacToken.
type ElodeskConfigResp struct {
	SessionID          string   `json:"sessionId"`
	URL                string   `json:"url"`
	InboxIdentifier    string   `json:"inboxIdentifier"`
	HasAPIToken        bool     `json:"hasApiToken"`
	HasHMACToken       bool     `json:"hasHmacToken"`
	HasUserAccessToken bool     `json:"hasUserAccessToken"`
	AccountID          int      `json:"accountId"`
	SignMsg            bool     `json:"signMsg"`
	SignDelimiter      string   `json:"signDelimiter"`
	ReopenConv         bool     `json:"reopenConv"`
	MergeBRContacts    bool     `json:"mergeBrContacts"`
	IgnoreGroups       bool     `json:"ignoreGroups"`
	IgnoreJIDs         []string `json:"ignoreJids"`
	PendingConv        bool     `json:"pendingConv"`
	Enabled            bool     `json:"enabled"`
	WebhookURL         string   `json:"webhookUrl"`
	ImportOnConnect    bool     `json:"importOnConnect"`
	ImportPeriod       string   `json:"importPeriod"`
	TextTimeout        int      `json:"textTimeout"`
	MediaTimeout       int      `json:"mediaTimeout"`
	LargeTimeout       int      `json:"largeTimeout"`
	MessageRead        bool     `json:"messageRead"`
}

type ElodeskImportReq struct {
	Period     string `json:"period" validate:"required,oneof=24h 7d 30d custom"`
	CustomDays int    `json:"customDays,omitempty"`
}

type ElodeskImportResp struct {
	SessionID string `json:"sessionId"`
	Period    string `json:"period"`
	Status    string `json:"status"`
}

// ElodeskWebhookPayload reflete o shape real enviado por
// elodesk/backend/internal/webhook/outbound_processor.go (publicBody) —
// event + accountId/inboxId top-level + conversation/message em camelCase
// com status int.
type ElodeskWebhookPayload struct {
	Event      string `json:"event"`
	EventType  string `json:"event_type,omitempty"`
	AccountID  int64  `json:"accountId"`
	InboxID    int64  `json:"inboxId"`
	DeliveryID string `json:"deliveryId,omitempty"`
	Private    bool   `json:"private,omitempty"`

	Message      *ElodeskWebhookMessage      `json:"message,omitempty"`
	Conversation *ElodeskWebhookConversation `json:"conversation,omitempty"`
}

type ElodeskWebhookMessage struct {
	ID                     int64                      `json:"id"`
	AccountID              int64                      `json:"accountId,omitempty"`
	InboxID                int64                      `json:"inboxId,omitempty"`
	ConversationID         int64                      `json:"conversationId,omitempty"`
	MessageType            int                        `json:"messageType"`
	ContentType            int                        `json:"contentType,omitempty"`
	Content                string                     `json:"content,omitempty"`
	Private                bool                       `json:"private,omitempty"`
	Status                 int                        `json:"status,omitempty"`
	SourceID               string                     `json:"sourceId,omitempty"`
	Attachments            []ElodeskWebhookAttachment `json:"attachments,omitempty"`
	CreatedAt              string                     `json:"createdAt,omitempty"`
	UpdatedAt              string                     `json:"updatedAt,omitempty"`
	ForwardedFromMessageID *int64                     `json:"forwardedFromMessageId,omitempty"`
}

// ElodeskWebhookAttachment espelha backend/internal/model.Attachment do
// elodesk + um campo extra `dataUrl` que o elodesk enriquece no momento do
// dispatch (URL presigned do MinIO). É a única forma do wzap baixar a
// mídia sem acessar o MinIO do elodesk diretamente.
//
// FileType: 0=image, 1=audio, 2=video, 3=file, 4=location, 5=fallback.
type ElodeskWebhookAttachment struct {
	ID          int64   `json:"id,omitempty"`
	MessageID   int64   `json:"messageId,omitempty"`
	AccountID   int64   `json:"accountId,omitempty"`
	FileType    int     `json:"fileType"`
	FileKey     *string `json:"fileKey,omitempty"`
	ExternalURL *string `json:"externalUrl,omitempty"`
	Extension   *string `json:"extension,omitempty"`
	DataURL     string  `json:"dataUrl,omitempty"`
}

type ElodeskWebhookConversation struct {
	ID             int64                       `json:"id"`
	AccountID      int64                       `json:"accountId,omitempty"`
	InboxID        int64                       `json:"inboxId,omitempty"`
	Status         int                         `json:"status"`
	ContactID      int64                       `json:"contactId,omitempty"`
	ContactInboxID int64                       `json:"contactInboxId,omitempty"`
	DisplayID      int64                       `json:"displayId,omitempty"`
	UUID           string                      `json:"uuid,omitempty"`
	ContactInbox   *ElodeskWebhookContactInbox `json:"contactInbox,omitempty"`
}

// ElodeskWebhookContactInbox carrega o source_id do canal (telefone E.164 para
// WhatsApp/SMS, email para Email, identifier para os demais). Usado como
// fallback de destino quando wz_messages ainda não tem mapeamento
// elodesk_conv_id → chat_jid (caso típico: forward para contato sem histórico).
type ElodeskWebhookContactInbox struct {
	ID       int64  `json:"id"`
	SourceID string `json:"sourceId"`
}

func (p *ElodeskWebhookPayload) GetMessage() *ElodeskWebhookMessage {
	return p.Message
}
