package elodesk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("elodesk API error: status=%d, body=%s", e.StatusCode, e.Message)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	return true
}

// Contact no contrato elodesk Channel::Api. Identifier é o JID do WA.
type Contact struct {
	ID                   int            `json:"id"`
	Name                 string         `json:"name"`
	PhoneNumber          string         `json:"phoneNumber,omitempty"`
	Identifier           string         `json:"identifier,omitempty"`
	Email                string         `json:"email,omitempty"`
	Thumbnail            string         `json:"thumbnail,omitempty"`
	AdditionalAttributes map[string]any `json:"additionalAttributes,omitempty"`
	SourceID             string         `json:"sourceId,omitempty"`
}

type Conversation struct {
	ID        int `json:"id"`
	ContactID int `json:"contactId"`
	InboxID   int `json:"inboxId"`
	Status    int `json:"status"`
}

const (
	ConversationStatusOpen     = 0
	ConversationStatusResolved = 1
	ConversationStatusPending  = 2
	ConversationStatusSnoozed  = 3
)

type MessageReq struct {
	Content           string         `json:"message,omitempty"`
	MessageType       string         `json:"message_type,omitempty"`
	ContentType       string         `json:"content_type,omitempty"`
	Private           bool           `json:"private,omitempty"`
	SourceID          string         `json:"source_id,omitempty"`
	ContentAttributes map[string]any `json:"content_attributes,omitempty"`
	Echo              bool           `json:"echo_id,omitempty"`
	// SenderContactID is the actual sender contact in group conversations,
	// where the chat-level contact (group) differs from the message author
	// (group member). Maps to the polymorphic `messages.sender_id` on the
	// Elodesk side via dto.CreateMessageReq.
	SenderContactID *int64 `json:"sender_contact_id,omitempty"`
}

type Message struct {
	ID             int64  `json:"id"`
	Content        string `json:"content,omitempty"`
	MessageType    int    `json:"messageType"`
	ContentType    int    `json:"contentType,omitempty"`
	SourceID       string `json:"sourceId,omitempty"`
	ConversationID int64  `json:"conversationId"`
}

// UpsertContactReq é o corpo de POST/PATCH de contato no endpoint
// /public/api/v1/inboxes/{identifier}/contacts.
type UpsertContactReq struct {
	SourceID             string         `json:"source_id,omitempty"`
	Identifier           string         `json:"identifier,omitempty"`
	Name                 string         `json:"name,omitempty"`
	PhoneNumber          string         `json:"phone_number,omitempty"`
	Email                string         `json:"email,omitempty"`
	AvatarURL            string         `json:"avatar_url,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

type GetOrCreateConvReq struct {
	ContactIdentifier string `json:"contact_identifier,omitempty"`
	Status            string `json:"status,omitempty"`
}

type CreateInboxResp struct {
	ID          int64  `json:"id"`
	AccountID   int64  `json:"accountId"`
	ChannelID   int64  `json:"channelId"`
	Name        string `json:"name"`
	ChannelType string `json:"channelType"`
	Identifier  string `json:"identifier"`
	ApiToken    string `json:"apiToken"`
	HmacToken   string `json:"hmacToken"`
	Secret      string `json:"secret"`
	CreatedAt   string `json:"createdAt"`
}

type Client interface {
	UpsertContact(ctx context.Context, identifier string, req UpsertContactReq) (*Contact, error)
	GetOrCreateConversation(ctx context.Context, identifier, contactSrcID string, req GetOrCreateConvReq) (*Conversation, error)
	CreateMessage(ctx context.Context, identifier, contactSrcID string, convID int64, req MessageReq) (*Message, error)
	CreateAttachment(ctx context.Context, identifier, contactSrcID string, convID int64, content, filename string, data []byte, mimeType, messageType, sourceID string, contentAttrs map[string]any) (*Message, error)
	UpdateConversationStatus(ctx context.Context, identifier, contactSourceID string, convID int64, status string) error
	CreateInbox(ctx context.Context, accountID int, name, webhookURL, userAccessToken string) (*CreateInboxResp, error)
	UpdateInboxWebhook(ctx context.Context, accountID, channelID int, webhookURL, userAccessToken string) error
}

type HTTPClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, httpClient *http.Client) *HTTPClient {
	baseURL = strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body any, result any, contentType string) error {
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

	fullURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
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
		return &APIError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		var envelope struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
			Message string          `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, result); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
	}

	return nil
}

// UpsertContact cria ou atualiza um contato no inbox público.
// POST /public/api/v1/inboxes/{identifier}/contacts
func (c *HTTPClient) UpsertContact(ctx context.Context, identifier string, req UpsertContactReq) (*Contact, error) {
	var result Contact
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contacts", url.PathEscape(identifier))
	if err := c.do(ctx, http.MethodPost, path, req, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrCreateConversation abre (ou retorna) uma conversation para o contato.
// POST /public/api/v1/inboxes/{identifier}/contacts/{source_id}/conversations
func (c *HTTPClient) GetOrCreateConversation(ctx context.Context, identifier, contactSrcID string, req GetOrCreateConvReq) (*Conversation, error) {
	var result Conversation
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contacts/%s/conversations",
		url.PathEscape(identifier), url.PathEscape(contactSrcID))
	if err := c.do(ctx, http.MethodPost, path, req, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateMessage posta uma mensagem (texto ou com content_attributes).
// POST /public/api/v1/inboxes/{identifier}/contacts/{source_id}/conversations/{conv_id}/messages
func (c *HTTPClient) CreateMessage(ctx context.Context, identifier, contactSrcID string, convID int64, req MessageReq) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contacts/%s/conversations/%d/messages",
		url.PathEscape(identifier), url.PathEscape(contactSrcID), convID)
	if err := c.do(ctx, http.MethodPost, path, req, &result, ""); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateAttachment posta uma mensagem com anexo via multipart.
func (c *HTTPClient) CreateAttachment(ctx context.Context, identifier, contactSrcID string, convID int64, content, filename string, data []byte, mimeType, messageType, sourceID string, contentAttrs map[string]any) (*Message, error) {
	var result Message
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contacts/%s/conversations/%d/messages",
		url.PathEscape(identifier), url.PathEscape(contactSrcID), convID)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if content != "" {
		_ = writer.WriteField("content", content)
	}
	if messageType != "" {
		_ = writer.WriteField("message_type", messageType)
	}
	if sourceID != "" {
		_ = writer.WriteField("source_id", sourceID)
	}
	if len(contentAttrs) > 0 {
		if caJSON, err := json.Marshal(contentAttrs); err == nil {
			_ = writer.WriteField("content_attributes", string(caJSON))
		}
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

// UpdateConversationStatus toggles conversation status (open/resolved/pending).
// The Elodesk public API requires the contact source_id in the path, mirroring
// CreateMessage and GetOrCreateConversation — without it the route returns 404.
func (c *HTTPClient) UpdateConversationStatus(ctx context.Context, identifier, contactSourceID string, convID int64, status string) error {
	path := fmt.Sprintf("/public/api/v1/inboxes/%s/contacts/%s/conversations/%d/toggle_status",
		url.PathEscape(identifier), url.PathEscape(contactSourceID), convID)
	body := map[string]string{"status": status}
	return c.do(ctx, http.MethodPost, path, body, nil, "")
}

func (c *HTTPClient) CreateInbox(ctx context.Context, accountID int, name, webhookURL, userAccessToken string) (*CreateInboxResp, error) {
	path := fmt.Sprintf("/api/v1/accounts/%d/inboxes", accountID)
	body := map[string]any{
		"name":       name,
		"webhookUrl": webhookURL,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user_access_token", userAccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
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
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	var envelope struct {
		Data CreateInboxResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &envelope.Data, nil
}

func (c *HTTPClient) UpdateInboxWebhook(ctx context.Context, accountID, channelID int, webhookURL, userAccessToken string) error {
	path := fmt.Sprintf("/api/v1/accounts/%d/inboxes/api/%d", accountID, channelID)
	body := map[string]any{
		"webhookUrl": webhookURL,
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user_access_token", userAccessToken)

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
		return &APIError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	return nil
}

// =====================================================
// Circuit breaker (duplicado do chatwoot/client.go até aparecer
// 3ª integração e fazer sentido fatorar em pacote shared).
// =====================================================

type cbState int

const (
	cbClosed   cbState = 0
	cbOpen     cbState = 1
	cbHalfOpen cbState = 2

	cbThreshold = 5
	cbTimeout   = 30 * time.Second
)

type circuitBreaker struct {
	mu       sync.Mutex
	state    cbState
	failures int
	lastFail time.Time
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{state: cbClosed}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.lastFail) > cbTimeout {
			cb.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		return true
	}
	return true
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = cbClosed
	cb.failures = 0
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFail = time.Now()
	if cb.state == cbHalfOpen || cb.failures >= cbThreshold {
		cb.state = cbOpen
	}
}

type cbManager struct {
	mu  sync.RWMutex
	cbs map[string]*circuitBreaker
}

func newCircuitBreakerManager() *cbManager {
	return &cbManager{cbs: make(map[string]*circuitBreaker)}
}

func (m *cbManager) get(sessionID string) *circuitBreaker {
	m.mu.RLock()
	cb, ok := m.cbs[sessionID]
	m.mu.RUnlock()
	if ok {
		return cb
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cb, ok = m.cbs[sessionID]; ok {
		return cb
	}
	cb = newCircuitBreaker()
	m.cbs[sessionID] = cb
	return cb
}

func (m *cbManager) Allow(sessionID string) bool    { return m.get(sessionID).Allow() }
func (m *cbManager) RecordSuccess(sessionID string) { m.get(sessionID).RecordSuccess() }
func (m *cbManager) RecordFailure(sessionID string) { m.get(sessionID).RecordFailure() }
