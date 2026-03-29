package dto

type SessionCreateReq struct {
	Name     string                 `json:"name"`
	Token    string                 `json:"token,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
