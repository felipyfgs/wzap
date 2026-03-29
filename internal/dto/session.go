package dto

type SessionCreateReq struct {
	Name     string                 `json:"name"`
	ApiKey   string                 `json:"apiKey,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
