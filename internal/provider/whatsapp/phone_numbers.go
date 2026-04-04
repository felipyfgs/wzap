package whatsapp

import (
	"context"
	"fmt"
)

func (c *Client) ListPhoneNumbers(ctx context.Context, sessionID string) (*PhoneNumberListResponse, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	endpoint := cfg.BusinessAccountID + "/phone_numbers"
	body, err := c.doBusinessRequest(ctx, sessionID, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list phone numbers: %w", err)
	}

	var resp PhoneNumberListResponse
	if err := unmarshalJSON(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phone numbers response: %w", err)
	}

	return &resp, nil
}

func (c *Client) GetPhoneNumber(ctx context.Context, sessionID, phoneNumberID string) (*PhoneNumber, error) {
	body, err := c.doBusinessRequest(ctx, sessionID, "GET", phoneNumberID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get phone number: %w", err)
	}

	var pn PhoneNumber
	if err := unmarshalJSON(body, &pn); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phone number response: %w", err)
	}

	return &pn, nil
}
