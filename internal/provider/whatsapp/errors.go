package whatsapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
	Message string     `json:"message"`
	Type    string     `json:"type"`
	Code    int        `json:"code"`
	Subcode int        `json:"error_subcode"`
	Data    *ErrorData `json:"error_data,omitempty"`
}

type ErrorData struct {
	MessagingProduct string `json:"messaging_product"`
	Details          string `json:"details"`
}

type errorResponse struct {
	Error *APIError `json:"error"`
}

func (e *APIError) Error() string {
	if e.Subcode != 0 {
		return fmt.Sprintf("whatsapp cloud api error (code=%d, subcode=%d): %s", e.Code, e.Subcode, e.Message)
	}
	return fmt.Sprintf("whatsapp cloud api error (code=%d): %s", e.Code, e.Message)
}

func parseErrorResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response body (status %d): %w", resp.StatusCode, err)
	}

	var errResp errorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr != nil || errResp.Error == nil {
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	return errResp.Error
}
