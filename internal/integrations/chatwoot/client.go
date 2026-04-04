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
	"path/filepath"
	"strings"
)

type Client struct {
	baseURL   string
	accountID int
	token     string
	httpClient *http.Client
}

func NewClient(baseURL string, accountID int, token string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:   baseURL,
		accountID: accountID,
		token:     token,
		httpClient: httpClient,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var reqBody io.Reader
	var contentType string

	if body != nil {
		if buf, ok := body.(io.Reader); ok {
			reqBody = buf
		} else if data, ok := body.([]byte); ok {
			reqBody = bytes.NewReader(data)
		} else {
			data, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(data)
			contentType = "application/json"
		}
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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
		bodyBytes, _ := io.ReadAll(resp.Body)
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
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	PhoneNumber  string           `json:"phone_number"`
	Email        string           `json:"email,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

type ContactFilterReq struct {
	Q string `json:"q"`
}

func (c *Client) FilterContacts(ctx context.Context, phone string) ([]Contact, error) {
	var result struct {
		Payload struct {
			Contacts []Contact `json:"contacts"`
		} `json:"payload"`
	}
	reqBody := ContactFilterReq{Q: phone}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/public/api/v1/accounts/%d/contacts/filter", c.accountID)
	if err := c.do(ctx, http.MethodPost, path, data, &result); err != nil {
		return nil, err
	}
	return result.Payload.Contacts, nil
}

type CreateContactReq struct {
	InboxID         int    `json:"inbox_id"`
	Name            string `json:"name,omitempty"`
	PhoneNumber     string `json:"phone_number,omitempty"`
	Email           string `json:"email,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

func (c *Client) CreateContact(ctx context.Context, req CreateContactReq) (*Contact, error) {
	var result Contact
	path := fmt.Sprintf("/public/api/v1/accounts/%d/contacts", c.accountID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type UpdateContactReq struct {
	Name            string `json:"name,omitempty"`
	Email           string `json:"email,omitempty"`
	PhoneNumber     string `json:"phone_number,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

func (c *Client) UpdateContact(ctx context.Context, id int, req UpdateContactReq) error {
	path := fmt.Sprintf("/public/api/v1/accounts/%d/contacts/%d", c.accountID, id)
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPatch, path, data, nil)
}

type Conversation struct {
	ID           int    `json:"id"`
	ContactID    int    `json:"contact_id"`
	InboxID      int    `json:"inbox_id"`
	Status       string `json:"status"`
	Messages     []Message `json:"messages,omitempty"`
}

func (c *Client) ListContactConversations(ctx context.Context, contactID int) ([]Conversation, error) {
	var result []Conversation
	path := fmt.Sprintf("/public/api/v1/accounts/%d/contacts/%d/conversations", c.accountID, contactID)
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type CreateConversationReq struct {
	InboxID  int    `json:"inbox_id"`
	SourceID string `json:"source_id,omitempty"`
}

func (c *Client) CreateConversation(ctx context.Context, req CreateConversationReq) (*Conversation, error) {
	var result Conversation
	path := fmt.Sprintf("/public/api/v1/accounts/%d/conversations", c.accountID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateConversationStatus(ctx context.Context, convID int, status string) error {
	path := fmt.Sprintf("/public/api/v1/accounts/%d/conversations/%d/toggle_status", c.accountID, convID)
	body := map[string]string{"status": status}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, nil)
}

type MessageReq struct {
	Content     string `json:"content,omitempty"`
	MessageType string `json:"message_type"`
	Private     bool   `json:"private,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
}

type Message struct {
	ID          int    `json:"id"`
	Content     string `json:"content,omitempty"`
	MessageType string `json:"message_type"`
	ContentType string `json:"content_type,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
	ConversationID int `json:"conversation_id"`
}

func (c *Client) CreateMessage(ctx context.Context, convID int, req MessageReq) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/public/api/v1/accounts/%d/conversations/%d/messages", c.accountID, convID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateMessageWithAttachment(ctx context.Context, convID int, content string, filename string, data []byte, mimeType string) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/public/api/v1/accounts/%d/conversations/%d/messages", c.accountID, convID)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if content != "" {
		_ = writer.WriteField("content", content)
	}
	_ = writer.WriteField("message_type", "outgoing")

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

	return &result, c.doWithContentType(ctx, http.MethodPost, path, &buf, &result, writer.FormDataContentType())
}

func (c *Client) doWithContentType(ctx context.Context, method, path string, body io.Reader, result any, contentType string) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chatwoot API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) DeleteMessage(ctx context.Context, convID, msgID int) error {
	path := fmt.Sprintf("/public/api/v1/accounts/%d/conversations/%d/messages/%d", c.accountID, convID, msgID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

type Inbox struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ChannelType string `json:"channel_type,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

func (c *Client) ListInboxes(ctx context.Context) ([]Inbox, error) {
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	path := fmt.Sprintf("/platform/api/v1/accounts/%d/inboxes", c.accountID)
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func (c *Client) CreateInbox(ctx context.Context, name, webhookURL string) (*Inbox, error) {
	var result Inbox
	path := fmt.Sprintf("/platform/api/v1/accounts/%d/inboxes", c.accountID)
	body := map[string]any{
		"name": name,
		"channel": map[string]any{
			"type": "Channel::Api",
			"webhook_url": webhookURL,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if err := c.do(ctx, http.MethodPost, path, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateLastSeen(ctx context.Context, inboxIdentifier, sourceID string, convID int) error {
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contact_inboxes/conversations/%d/update_last_seen", inboxIdentifier, convID)
	body := map[string]any{
		"source_id": sourceID,
		"last_seen": "2006-01-02T15:04:05.000Z",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, nil)
}

func GetMIMETypeAndExt(url string, data []byte) (mimeType, ext string) {
	if url != "" {
		if e := filepath.Ext(url); e != "" {
			ext = e
			switch strings.ToLower(e) {
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".png":
				mimeType = "image/png"
			case ".gif":
				mimeType = "image/gif"
			case ".webp":
				mimeType = "image/webp"
			case ".mp4":
				mimeType = "video/mp4"
			case ".ogg":
				mimeType = "audio/ogg"
			case ".mp3":
				mimeType = "audio/mpeg"
			case ".pdf":
				mimeType = "application/pdf"
			case ".doc", ".docx":
				mimeType = "application/msword"
			default:
				mimeType = "application/octet-stream"
			}
			return
		}
	}

	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			mimeType = "image/jpeg"
			ext = ".jpg"
			return
		}
		if data[0] == 0x89 && string(data[1:4]) == "PNG" {
			mimeType = "image/png"
			ext = ".png"
			return
		}
		if string(data[:4]) == "RIFF" {
			mimeType = "audio/wav"
			ext = ".wav"
			return
		}
	}

	if len(data) >= 2 {
		if data[0] == 0x4F && data[1] == 0x67 {
			mimeType = "audio/ogg"
			ext = ".ogg"
			return
		}
	}

	mimeType = "application/octet-stream"
	ext = ""
	return
}
