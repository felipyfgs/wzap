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
