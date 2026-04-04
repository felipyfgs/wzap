package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

func (c *Client) UploadMedia(ctx context.Context, sessionID, filename, mimeType string, data []byte) (*UploadMediaResponse, error) {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)

	fw, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(fw, bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.WriteField("messaging_product", "whatsapp"); err != nil {
		return nil, fmt.Errorf("failed to write messaging_product field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	reqURL := c.buildURL(cfg, cfg.PhoneNumberID+"/media")

	req, err := c.newMultipartRequest(ctx, "POST", reqURL, cfg.AccessToken, &buf, writer.FormDataContentType())
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := parseErrorResponse(resp); err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var uploadResp UploadMediaResponse
	if err := unmarshalJSON(respBody, &uploadResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal upload response: %w", err)
	}

	return &uploadResp, nil
}

func (c *Client) GetMediaURL(ctx context.Context, sessionID, mediaID string) (*MediaInfo, error) {
	body, err := c.doBusinessRequest(ctx, sessionID, "GET", mediaID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get media info: %w", err)
	}

	var info MediaInfo
	if err := unmarshalJSON(body, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media info: %w", err)
	}

	return &info, nil
}

func (c *Client) DownloadMedia(ctx context.Context, downloadURL string) ([]byte, string, error) {
	return c.doRequestDownload(ctx, downloadURL)
}

func (c *Client) DeleteMedia(ctx context.Context, sessionID, mediaID string) error {
	cfg, err := c.configReader.ReadConfig(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to read config for session %s: %w", sessionID, err)
	}

	reqURL := c.buildURL(cfg, mediaID)

	req, err := c.newRequest(ctx, "DELETE", reqURL, cfg.AccessToken, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := parseErrorResponse(resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) newMultipartRequest(ctx context.Context, method, url, token string, body io.Reader, contentType string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

func (c *Client) newRequest(ctx context.Context, method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
