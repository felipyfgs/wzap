package dto

type APIResponse struct {
	Success bool        `json:"Success"`
	Data    interface{} `json:"Data,omitempty"`
	Message string      `json:"Message"`
}

type APIError struct {
	Success bool   `json:"Success"`
	Error   string `json:"Error"`
	Message string `json:"Message,omitempty"`
}

type MidResp struct {
	Mid string `json:"Mid"`
}

type ConnectResp struct {
	Status string `json:"Status"`
}

type QRResp struct {
	QRCode string `json:"QRCode"`
	Image  string `json:"Image"`
}

type PictureIDResp struct {
	PictureID string `json:"PictureId"`
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
