package dto

type SessionCreateReq struct {
	Name     string                 `json:"name"`
	APIKey   string                 `json:"apiKey,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
