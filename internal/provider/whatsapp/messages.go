package whatsapp

import (
	"context"
	"fmt"
)

type messagePayload struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type,omitempty"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Context          *MessageContext `json:"context,omitempty"`

	Text       *TextMessage       `json:"text,omitempty"`
	Image      *MediaIDOrURL      `json:"image,omitempty"`
	Video      *MediaIDOrURL      `json:"video,omitempty"`
	Audio      *MediaIDOrURL      `json:"audio,omitempty"`
	Document   *MediaIDOrURL      `json:"document,omitempty"`
	Sticker    *MediaIDOrURL      `json:"sticker,omitempty"`
	Location   *LocationMessage   `json:"location,omitempty"`
	Contacts   []Contact          `json:"contacts,omitempty"`
	Reaction   *ReactionMessage   `json:"reaction,omitempty"`
	Template   *Template          `json:"template,omitempty"`
	Interactive *Interactive      `json:"interactive,omitempty"`
	TypingIndicator *TypingIndicator `json:"typing_indicator,omitempty"`
}

func (c *Client) buildMessagePayload(sessionID, recipient, msgType string, opts []SendOption) *messagePayload {
	so := applySendOptions(opts)
	p := &messagePayload{
		MessagingProduct: "whatsapp",
		To:               recipient,
		Type:             msgType,
	}
	if so.ReplyTo != "" {
		p.Context = &MessageContext{MessageID: so.ReplyTo}
	}
	return p
}

func (c *Client) SendText(ctx context.Context, sessionID, recipient, text string, opts ...SendOption) (*MessageResponse, error) {
	so := applySendOptions(opts)
	p := c.buildMessagePayload(sessionID, recipient, "text", nil)
	p.Text = &TextMessage{Body: text}
	if so.PreviewURL != nil {
		p.Text.PreviewURL = so.PreviewURL
	}

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send text message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendImage(ctx context.Context, sessionID, recipient string, media *MediaIDOrURL, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "image", opts)
	p.Image = media

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send image message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendVideo(ctx context.Context, sessionID, recipient string, media *MediaIDOrURL, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "video", opts)
	p.Video = media

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send video message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendAudio(ctx context.Context, sessionID, recipient string, media *MediaIDOrURL, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "audio", opts)
	p.Audio = media

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send audio message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendDocument(ctx context.Context, sessionID, recipient string, media *MediaIDOrURL, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "document", opts)
	p.Document = media

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send document message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendSticker(ctx context.Context, sessionID, recipient string, media *MediaIDOrURL, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "sticker", opts)
	p.Sticker = media

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send sticker message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendLocation(ctx context.Context, sessionID, recipient string, lat, lng float64, name, address string, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "location", opts)
	p.Location = &LocationMessage{
		Latitude:  lat,
		Longitude: lng,
		Name:      name,
		Address:   address,
	}

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send location message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendContacts(ctx context.Context, sessionID, recipient string, contacts []Contact, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "contacts", opts)
	p.Contacts = contacts

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send contacts: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendReaction(ctx context.Context, sessionID, recipient, messageID, emoji string, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "reaction", opts)
	p.Reaction = &ReactionMessage{
		MessageID: messageID,
		Emoji:     emoji,
	}

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send reaction: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) MarkRead(ctx context.Context, sessionID, messageID string) error {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	_, err = c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), payload)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}
	return nil
}

func (c *Client) SendTemplate(ctx context.Context, sessionID, recipient string, template *Template, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "template", opts)
	p.Template = template

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send template message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendInteractive(ctx context.Context, sessionID, recipient string, interactive *Interactive, opts ...SendOption) (*MessageResponse, error) {
	p := c.buildMessagePayload(sessionID, recipient, "interactive", opts)
	p.Interactive = interactive

	resp, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), p)
	if err != nil {
		return nil, fmt.Errorf("failed to send interactive message: %w", err)
	}
	r := resp.toMessageResponse()
	return &r, nil
}

func (c *Client) SendTypingIndicator(ctx context.Context, sessionID, recipient string) error {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                recipient,
		"typing_indicator": map[string]string{
			"type": "text",
		},
	}

	_, err := c.doRequest(ctx, sessionID, "POST", c.messagesEndpoint(sessionID), payload)
	if err != nil {
		return fmt.Errorf("failed to send typing indicator: %w", err)
	}
	return nil
}

func (c *Client) messagesEndpoint(sessionID string) string {
	cfg, _ := c.configReader.ReadConfig(context.Background(), sessionID)
	if cfg == nil {
		return "messages"
	}
	return cfg.PhoneNumberID + "/messages"
}
