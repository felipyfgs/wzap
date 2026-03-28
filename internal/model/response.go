package model

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type APIError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func SuccessResp(data interface{}, msg string) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Message: msg,
	}
}

func ErrorResp(err string, details string) APIError {
	return APIError{
		Success: false,
		Error:   err,
		Details: details,
	}
}
