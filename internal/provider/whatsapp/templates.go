package whatsapp

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListTemplates(ctx context.Context, sessionID string, limit int) (*TemplateListResponse, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	endpoint := cfg.BusinessAccountID + "/message_templates"
	if limit > 0 {
		endpoint += buildQueryParams(url.Values{"limit": []string{fmt.Sprintf("%d", limit)}})
	}

	body, err := c.doBusinessRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var resp TemplateListResponse
	if err := unmarshalJSON(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates response: %w", err)
	}

	return &resp, nil
}

func (c *Client) CreateTemplate(ctx context.Context, sessionID string, req *CreateTemplateRequest) (*CreateTemplateResponse, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	endpoint := cfg.BusinessAccountID + "/message_templates"
	body, err := c.doBusinessRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	var resp CreateTemplateResponse
	if err := unmarshalJSON(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create template response: %w", err)
	}

	return &resp, nil
}

func (c *Client) GetTemplate(ctx context.Context, sessionID, templateID string) (*TemplateNode, error) {
	body, err := c.doBusinessRequest(ctx, "GET", templateID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	var node TemplateNode
	if err := unmarshalJSON(body, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template response: %w", err)
	}

	return &node, nil
}

func (c *Client) UpdateTemplate(ctx context.Context, sessionID, templateID string, req *UpdateTemplateRequest) (*TemplateNode, error) {
	body, err := c.doBusinessRequest(ctx, "POST", templateID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	var node TemplateNode
	if err := unmarshalJSON(body, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update template response: %w", err)
	}

	return &node, nil
}

func (c *Client) DeleteTemplate(ctx context.Context, sessionID, templateID string) error {
	_, err := c.doBusinessRequest(ctx, "DELETE", templateID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	return nil
}
