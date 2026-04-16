package dto

type CloudAPIMessageReq struct {
	MessagingProduct string            `json:"messaging_product"`
	RecipientType    string            `json:"recipient_type,omitempty"`
	To               string            `json:"to"`
	Type             string            `json:"type"`
	Text             *CloudAPIText     `json:"text,omitempty"`
	Image            *CloudAPIMedia    `json:"image,omitempty"`
	Video            *CloudAPIMedia    `json:"video,omitempty"`
	Audio            *CloudAPIMedia    `json:"audio,omitempty"`
	Document         *CloudAPIDocument `json:"document,omitempty"`
	Sticker          *CloudAPIMedia    `json:"sticker,omitempty"`
	Location         *CloudAPILocation `json:"location,omitempty"`
	Reaction         *CloudAPIReaction `json:"reaction,omitempty"`
	Contacts         []CloudAPIContact `json:"contacts,omitempty"`
	Context          *CloudAPIContext  `json:"context,omitempty"`
	Status           string            `json:"status,omitempty"`
	MessageID        string            `json:"message_id,omitempty"`
}

type CloudAPIText struct {
	Body       string `json:"body"`
	PreviewURL *bool  `json:"preview_url,omitempty"`
}

type CloudAPIMedia struct {
	Link     string `json:"link,omitempty"`
	ID       string `json:"id,omitempty"`
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type CloudAPIDocument struct {
	Link     string `json:"link,omitempty"`
	ID       string `json:"id,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type CloudAPILocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type CloudAPIReaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type CloudAPIContact struct {
	Addresses []struct {
		Street      string `json:"street,omitempty"`
		City        string `json:"city,omitempty"`
		State       string `json:"state,omitempty"`
		Zip         string `json:"zip,omitempty"`
		Country     string `json:"country,omitempty"`
		CountryCode string `json:"country_code,omitempty"`
		Type        string `json:"type,omitempty"`
	} `json:"addresses,omitempty"`
	Birthday string `json:"birthday,omitempty"`
	Emails   []struct {
		Email string `json:"email,omitempty"`
		Type  string `json:"type,omitempty"`
	} `json:"emails,omitempty"`
	Name struct {
		FirstName     string `json:"first_name,omitempty"`
		LastName      string `json:"last_name,omitempty"`
		FormattedName string `json:"formatted_name"`
	} `json:"name"`
	Org struct {
		Company string `json:"company,omitempty"`
	} `json:"org,omitempty"`
	Phones []struct {
		Phone string `json:"phone,omitempty"`
		Type  string `json:"type,omitempty"`
		WaID  string `json:"wa_id,omitempty"`
	} `json:"phones,omitempty"`
	Urls []struct {
		URL  string `json:"url,omitempty"`
		Type string `json:"type,omitempty"`
	} `json:"urls,omitempty"`
}

type CloudAPIContext struct {
	MessageID string `json:"message_id"`
}

type CloudAPIMessageResp struct {
	MessagingProduct string                `json:"messaging_product"`
	Contacts         []CloudAPIRespContact `json:"contacts"`
	Messages         []CloudAPIRespMessage `json:"messages"`
}

type CloudAPIRespContact struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type CloudAPIRespMessage struct {
	ID string `json:"id"`
}

type CloudAPIErrorResp struct {
	Error CloudAPIErrorDetail `json:"error"`
}

type CloudAPIErrorDetail struct {
	Message   string             `json:"message"`
	Type      string             `json:"type"`
	Code      int                `json:"code"`
	ErrorData *CloudAPIErrorData `json:"error_data,omitempty"`
}

type CloudAPIErrorData struct {
	MessagingProduct string `json:"messaging_product"`
	Details          string `json:"details"`
}

type CloudAPIMediaResp struct {
	URL              string `json:"url"`
	MimeType         string `json:"mime_type,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	FileSize         int64  `json:"file_size,omitempty"`
	ID               string `json:"id"`
	MessagingProduct string `json:"messaging_product"`
}
