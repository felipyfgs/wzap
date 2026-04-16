package dto

type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
}

type APIError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type MessageIDResp struct {
	MessageID string `json:"messageId"`
}

type ConnectResp struct {
	Status string `json:"status"`
}

type QRResp struct {
	QRCode string `json:"qrCode"`
	Image  string `json:"image"`
}

type PictureIDResp struct {
	PictureID string `json:"pictureId"`
}

func SuccessResp(data any) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Message: "success",
	}
}

func ErrorResp(err string, message string) APIError {
	return APIError{
		Success: false,
		Error:   err,
		Message: message,
	}
}
