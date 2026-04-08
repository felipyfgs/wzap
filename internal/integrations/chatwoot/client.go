package chatwoot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"wzap/internal/logger"
)

type CWClient interface {
	FilterContacts(ctx context.Context, phone string) ([]Contact, error)
	CreateContact(ctx context.Context, req CreateContactReq) (*Contact, error)
	UpdateContact(ctx context.Context, id int, req UpdateContactReq) error
	ListContactConversations(ctx context.Context, contactID int) ([]Conversation, error)
	CreateConversation(ctx context.Context, req CreateConversationReq) (*Conversation, error)
	UpdateConversationStatus(ctx context.Context, convID int, status string) error
	GetConversation(ctx context.Context, convID int) (*Conversation, error)
	MergeContacts(ctx context.Context, baseID, mergeeID int) error
	CreateMessage(ctx context.Context, convID int, req MessageReq) (*Message, error)
	CreateMessageWithAttachment(ctx context.Context, convID int, content string, filename string, data []byte, mimeType string, messageType string, sourceID string) (*Message, error)
	DeleteMessage(ctx context.Context, convID, msgID int) error
	UpdateMessage(ctx context.Context, convID, msgID int, content string) error
	UpdateLastSeen(ctx context.Context, inboxIdentifier, sourceID string, convID int) error
	ListInboxes(ctx context.Context) ([]Inbox, error)
	CreateInbox(ctx context.Context, name, webhookURL string) (*Inbox, error)
	UpdateInboxWebhook(ctx context.Context, inboxID int, webhookURL string) error
}

type Client struct {
	baseURL    string
	accountID  int
	token      string
	httpClient *http.Client
}

func NewClient(baseURL string, accountID int, token string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    baseURL,
		accountID:  accountID,
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any, contentType string) error {
	var reqBody io.Reader
	var ct string

	if body != nil {
		if buf, ok := body.(io.Reader); ok {
			reqBody = buf
			ct = contentType
		} else if data, ok := body.([]byte); ok {
			reqBody = bytes.NewReader(data)
			ct = "application/json"
		} else {
			data, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(data)
			ct = "application/json"
		}
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("api_access_token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if len(bodyBytes) > 512 {
			bodyBytes = bodyBytes[:512]
		}
		return fmt.Errorf("chatwoot API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

type Contact struct {
	ID                   int            `json:"id"`
	Name                 string         `json:"name"`
	PhoneNumber          string         `json:"phone_number"`
	Identifier           string         `json:"identifier,omitempty"`
	Email                string         `json:"email,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

func (c *Client) FilterContacts(ctx context.Context, phone string) ([]Contact, error) {
	var result struct {
		Payload []Contact `json:"payload"`
	}
	path := fmt.Sprintf("/api/v1/accounts/%d/contacts/search?q=%s&include_contacts=true", c.accountID, phone)
	if err := c.do(ctx, http.MethodGet, path, nil, &result, ""); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

type CreateContactReq struct {
	InboxID              int            `json:"inbox_id"`
	Name                 string         `json:"name,omitempty"`
	Identifier           string         `json:"identifier,omitempty"`
	PhoneNumber          string         `json:"phone_number,omitempty"`
	Email                string         `json:"email,omitempty"`
	AvatarURL            string         `json:"avatar_url,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

func (c *Client) CreateContact(ctx context.Context, req CreateContactReq) (*Contact, error) {
	var result struct {
		Payload struct {
			Contact Contact `json:"contact"`
		} `json:"payload"`
	}
	path := fmt.Sprintf("/api/v1/accounts/%d/contacts", c.accountID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result, ""); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

type UpdateContactReq struct {
	Name                 string         `json:"name,omitempty"`
	Identifier           string         `json:"identifier,omitempty"`
	Email                string         `json:"email,omitempty"`
	PhoneNumber          string         `json:"phone_number,omitempty"`
	AvatarURL            string         `json:"avatar_url,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

func (c *Client) UpdateContact(ctx context.Context, id int, req UpdateContactReq) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/contacts/%d", c.accountID, id)
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPatch, path, data, nil, "")
}

type Conversation struct {
	ID        int       `json:"id"`
	ContactID int       `json:"contact_id"`
	InboxID   int       `json:"inbox_id"`
	Status    string    `json:"status"`
	Messages  []Message `json:"messages,omitempty"`
}

func (c *Client) ListContactConversations(ctx context.Context, contactID int) ([]Conversation, error) {
	var result struct {
		Payload []Conversation `json:"payload"`
	}
	path := fmt.Sprintf("/api/v1/accounts/%d/contacts/%d/conversations", c.accountID, contactID)
	if err := c.do(ctx, http.MethodGet, path, nil, &result, ""); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

type CreateConversationReq struct {
	InboxID   int    `json:"inbox_id"`
	SourceID  string `json:"source_id,omitempty"`
	ContactID int    `json:"contact_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

func (c *Client) CreateConversation(ctx context.Context, req CreateConversationReq) (*Conversation, error) {
	var result Conversation
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations", c.accountID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateConversationStatus(ctx context.Context, convID int, status string) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d/toggle_status", c.accountID, convID)
	body := map[string]string{"status": status}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, nil, "")
}

func (c *Client) GetConversation(ctx context.Context, convID int) (*Conversation, error) {
	var result Conversation
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d", c.accountID, convID)
	if err := c.do(ctx, http.MethodGet, path, nil, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) MergeContacts(ctx context.Context, baseID, mergeeID int) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/actions/contact_merge", c.accountID)
	body := map[string]int{
		"base_contact_id":   baseID,
		"mergee_contact_id": mergeeID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, nil, "")
}

type MessageReq struct {
	Content           string         `json:"content,omitempty"`
	MessageType       string         `json:"message_type"`
	Private           bool           `json:"private,omitempty"`
	ContentType       string         `json:"content_type,omitempty"`
	SourceID          string         `json:"source_id,omitempty"`
	SourceReplyID     int            `json:"source_reply_id,omitempty"`
	ContentAttributes map[string]any `json:"content_attributes,omitempty"`
}

type Message struct {
	ID             int    `json:"id"`
	Content        string `json:"content,omitempty"`
	MessageType    int    `json:"message_type"`
	ContentType    string `json:"content_type,omitempty"`
	SourceID       string `json:"source_id,omitempty"`
	ConversationID int    `json:"conversation_id"`
}

func (c *Client) CreateMessage(ctx context.Context, convID int, req MessageReq) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d/messages", c.accountID, convID)

	if len(req.ContentAttributes) > 0 || req.SourceReplyID > 0 {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		if req.Content != "" {
			_ = writer.WriteField("content", req.Content)
		}
		_ = writer.WriteField("message_type", req.MessageType)
		if req.SourceID != "" {
			_ = writer.WriteField("source_id", req.SourceID)
		}
		if req.SourceReplyID > 0 {
			_ = writer.WriteField("source_reply_id", fmt.Sprintf("%d", req.SourceReplyID))
		}
		if len(req.ContentAttributes) > 0 {
			caJSON, err := json.Marshal(req.ContentAttributes)
			if err == nil {
				logger.Debug().Str("content_attributes", string(caJSON)).Int("source_reply_id", req.SourceReplyID).Msg("[CW] sending FormData to Chatwoot")
				_ = writer.WriteField("content_attributes", string(caJSON))
			}
		}

		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close multipart: %w", err)
		}

		return &result, c.do(ctx, http.MethodPost, path, &buf, &result, writer.FormDataContentType())
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateMessageWithAttachment(ctx context.Context, convID int, content string, filename string, data []byte, mimeType string, messageType string, sourceID string) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d/messages", c.accountID, convID)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if content != "" {
		_ = writer.WriteField("content", content)
	}
	_ = writer.WriteField("message_type", messageType)
	if sourceID != "" {
		_ = writer.WriteField("source_id", sourceID)
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachments[]"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("create form part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("write attachment: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	return &result, c.do(ctx, http.MethodPost, path, &buf, &result, writer.FormDataContentType())
}

func (c *Client) DeleteMessage(ctx context.Context, convID, msgID int) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d/messages/%d", c.accountID, convID, msgID)
	return c.do(ctx, http.MethodDelete, path, nil, nil, "")
}

func (c *Client) UpdateMessage(ctx context.Context, convID, msgID int, content string) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/conversations/%d/messages/%d", c.accountID, convID, msgID)
	body := map[string]string{"content": content}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPatch, path, data, nil, "")
}

type Inbox struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type,omitempty"`
	WebhookURL  string `json:"webhook_url,omitempty"`
}

func (c *Client) ListInboxes(ctx context.Context) ([]Inbox, error) {
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	path := fmt.Sprintf("/api/v1/accounts/%d/inboxes", c.accountID)
	if err := c.do(ctx, http.MethodGet, path, nil, &result, ""); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func (c *Client) CreateInbox(ctx context.Context, name, webhookURL string) (*Inbox, error) {
	var result Inbox
	path := fmt.Sprintf("/api/v1/accounts/%d/inboxes", c.accountID)
	body := map[string]any{
		"name": name,
		"channel": map[string]any{
			"type":        "api",
			"webhook_url": webhookURL,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateInboxWebhook(ctx context.Context, inboxID int, webhookURL string) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/inboxes/%d", c.accountID, inboxID)
	body := map[string]any{
		"channel": map[string]any{
			"webhook_url": webhookURL,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPatch, path, data, nil, "")
}

func (c *Client) UpdateLastSeen(ctx context.Context, inboxIdentifier, sourceID string, convID int) error {
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contact_inboxes/conversations/%d/update_last_seen", inboxIdentifier, convID)
	body := map[string]any{
		"source_id": sourceID,
		"last_seen": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, nil, "")
}
