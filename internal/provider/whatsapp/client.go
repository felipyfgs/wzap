package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

func (c *Client) doRequest(ctx context.Context, sessionID, method, endpoint string, body any) (*cloudAPIResponse, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}
	cfg.ApplyDefaults()

	reqURL := c.buildURL(cfg, endpoint)

	var reqBody io.Reader
	var contentType string

	switch v := body.(type) {
	case *multipart.Writer:
		reqBody = v.FormDataContentType()
		contentType = v.FormDataContentType()
	case io.Reader:
		reqBody = v
	case nil:
		reqBody = nil
	default:
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := parseErrorResponse(resp); err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp cloudAPIResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return &apiResp, nil
}

func (c *Client) doRequestRaw(ctx context.Context, sessionID, method, endpoint string, body any) ([]byte, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}
	cfg.ApplyDefaults()

	reqURL := c.buildURL(cfg, endpoint)

	var reqBody io.Reader
	var contentType string

	switch v := body.(type) {
	case io.Reader:
		reqBody = v
	case nil:
		reqBody = nil
	default:
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := parseErrorResponse(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) doRequestDownload(ctx context.Context, downloadURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read download body: %w", err)
	}

	return body, contentType, nil
}

func (c *Client) doBusinessRequest(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	cfg, err := c.configReader.ReadConfig(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	cfg.ApplyDefaults()

	reqURL := c.buildURL(cfg, endpoint)

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := parseErrorResponse(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) buildURL(cfg *Config, endpoint string) string {
	return fmt.Sprintf("%s/%s/%s", cfg.BaseURL, cfg.APIVersion, endpoint)
}

func buildQueryParams(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}
