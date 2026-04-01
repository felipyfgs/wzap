package dto

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}

type APIError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// MidResp is the response for all message send operations.
type MidResp struct {
	Mid string `json:"mid"`
}

// ConnectResp is the response for the connect endpoint.
type ConnectResp struct {
	// Status is one of: CONNECTED, PAIRING, CONNECTING
	Status string `json:"status"`
}

// QRResp is the response for the QR code endpoint.
type QRResp struct {
	QR    string `json:"qr"`
	Image string `json:"image"`
}

// PictureIDResp is the response for set-profile-picture.
type PictureIDResp struct {
	PictureID string `json:"pictureId"`
}

func SuccessResp(data interface{}) APIResponse {
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
