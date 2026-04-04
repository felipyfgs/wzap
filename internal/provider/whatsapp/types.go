package whatsapp

import (
	"context"
	"net/http"
)

const (
	DefaultBaseURL    = "https://graph.facebook.com"
	DefaultAPIVersion = "v23.0"
)

type Config struct {
	BaseURL            string
	APIVersion         string
	AccessToken        string
	PhoneNumberID      string
	BusinessAccountID  string
	AppSecret          string
	WebhookVerifyToken string
}

func (c *Config) ApplyDefaults() {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.APIVersion == "" {
		c.APIVersion = DefaultAPIVersion
	}
}

type ConfigReader interface {
	ReadConfig(ctx context.Context, sessionID string) (*Config, error)
}

type Client struct {
	httpClient   *http.Client
	configReader ConfigReader
}

func NewClient(httpClient *http.Client, reader ConfigReader) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient:   httpClient,
		configReader: reader,
	}
}

// ── Send Options ──────────────────────────────────────────────────────────────

type SendOption func(*SendOptions)

type SendOptions struct {
	ReplyTo     string
	PreviewURL  *bool
	CustomID    string
	ContextInfo *MessageContext
}

func WithReplyTo(messageID string) SendOption {
	return func(o *SendOptions) {
		o.ReplyTo = messageID
	}
}

func WithPreviewURL(preview bool) SendOption {
	return func(o *SendOptions) {
		o.PreviewURL = &preview
	}
}

func WithCustomID(id string) SendOption {
	return func(o *SendOptions) {
		o.CustomID = id
	}
}

func applySendOptions(opts []SendOption) SendOptions {
	so := SendOptions{}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

// ── Message Types ─────────────────────────────────────────────────────────────

type MessageContext struct {
	MessageID string `json:"message_id"`
}

type TextMessage struct {
	Body         string `json:"body"`
	PreviewURL   *bool  `json:"preview_url,omitempty"`
}

type MediaIDOrURL struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type LocationMessage struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type ContactAddress struct {
	Street       string `json:"street,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Zip          string `json:"zip,omitempty"`
	Country      string `json:"country,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	Type         string `json:"type,omitempty"`
}

type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
	Type  string `json:"type,omitempty"`
}

type ContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type ContactURL struct {
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

type Contact struct {
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Birthday  string           `json:"birthday,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Name      *ContactName     `json:"name"`
	Org       *ContactOrg      `json:"org,omitempty"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	URLs      []ContactURL     `json:"urls,omitempty"`
}

type ReactionMessage struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// ── Template Types ────────────────────────────────────────────────────────────

type Template struct {
	Name       string              `json:"name"`
	Language   *TemplateLanguage   `json:"language,omitempty"`
	Components []TemplateComponent `json:"components,omitempty"`
}

type TemplateLanguage struct {
	Code   string `json:"code"`
	Policy string `json:"policy,omitempty"`
}

type TemplateComponent struct {
	Type       string               `json:"type"`
	SubType    string               `json:"sub_type,omitempty"`
	Index      int                  `json:"index,omitempty"`
	Parameters []TemplateParameter  `json:"parameters,omitempty"`
}

type TemplateParameter struct {
	Type       string                    `json:"type"`
	Text       string                    `json:"text,omitempty"`
	Currency   *TemplateCurrency         `json:"currency,omitempty"`
	DateTime   *TemplateDateTime         `json:"date_time,omitempty"`
	Image      *TemplateMediaParam       `json:"image,omitempty"`
	Video      *TemplateMediaParam       `json:"video,omitempty"`
	Document   *TemplateMediaParam       `json:"document,omitempty"`
	Location   *TemplateLocationParam    `json:"location,omitempty"`
	Payload    string                    `json:"payload,omitempty"`
}

type TemplateCurrency struct {
	FallbackValue string  `json:"fallback_value"`
	Code          string  `json:"code"`
	Amount1000    float64 `json:"amount_1000"`
}

type TemplateDateTime struct {
	FallbackValue string `json:"fallback_value"`
}

type TemplateMediaParam struct {
	ID   string `json:"id,omitempty"`
	Link string `json:"link,omitempty"`
}

type TemplateLocationParam struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ── Interactive Types ─────────────────────────────────────────────────────────

type Interactive struct {
	Type   string              `json:"type"`
	Action *InteractiveAction  `json:"action,omitempty"`
	Body   *InteractiveBody    `json:"body,omitempty"`
	Footer *InteractiveFooter  `json:"footer,omitempty"`
	Header *InteractiveHeader  `json:"header,omitempty"`
}

type InteractiveAction struct {
	Button             string                    `json:"button,omitempty"`
	Buttons            []InteractiveButton       `json:"buttons,omitempty"`
	CatalogID          string                    `json:"catalog_id,omitempty"`
	ProductRetailerID  string                    `json:"product_retailer_id,omitempty"`
	Sections           []InteractiveSection      `json:"sections,omitempty"`
	Name               string                    `json:"name,omitempty"`
	Parameters         *InteractiveActionParam   `json:"parameters,omitempty"`
}

type InteractiveButton struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	ID    string `json:"reply,omitempty"`
}

type InteractiveReplyButton struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type InteractiveSection struct {
	Title      string                `json:"title,omitempty"`
	ProductItems []InteractiveProductItem `json:"product_items,omitempty"`
	Rows       []InteractiveSectionRow `json:"rows,omitempty"`
}

type InteractiveProductItem struct {
	ProductRetailerID string `json:"product_retailer_id"`
}

type InteractiveSectionRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type InteractiveBody struct {
	Text string `json:"text"`
}

type InteractiveFooter struct {
	Text string `json:"text,omitempty"`
}

type InteractiveHeader struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Image    *MediaIDOrURL `json:"image,omitempty"`
	Document *MediaIDOrURL `json:"document,omitempty"`
	Video    *MediaIDOrURL `json:"video,omitempty"`
}

type InteractiveActionParam struct {
	DisplayText string `json:"display_text,omitempty"`
	URL         string `json:"url,omitempty"`
	FlowToken   string `json:"flow_token,omitempty"`
	FlowID      string `json:"flow_id,omitempty"`
	FlowCTA     string `json:"flow_actioncta,omitempty"`
	FlowAction  string `json:"flow_action,omitempty"`
}

// ── Response Types ────────────────────────────────────────────────────────────

type MessageResponse struct {
	MessageID string
	Contacts  []ResponseContact
}

type ResponseContact struct {
	Input      string `json:"input"`
	WhatsAppID string `json:"wa_id"`
}

type cloudAPIResponse struct {
	MessagingProduct string            `json:"messaging_product"`
	Contacts         []ResponseContact `json:"contacts"`
	Messages         []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

func (r *cloudAPIResponse) toMessageResponse() MessageResponse {
	resp := MessageResponse{Contacts: r.Contacts}
	if len(r.Messages) > 0 {
		resp.MessageID = r.Messages[0].ID
	}
	return resp
}

// ── Media Types ───────────────────────────────────────────────────────────────

type MediaInfo struct {
	MessagingProduct string `json:"messaging_product"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	ID               string `json:"id"`
}

type UploadMediaResponse struct {
	ID string `json:"id"`
}

// ── Webhook Types ─────────────────────────────────────────────────────────────

type WebhookNotification struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID    string   `json:"id"`
	Time  int64    `json:"time"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string `json:"field"`
	Value *Value `json:"value"`
}

type Value struct {
	MessagingProduct string     `json:"messaging_product"`
	Metadata         *Metadata  `json:"metadata"`
	Contacts         []Contact  `json:"contacts"`
	Messages         []Message  `json:"messages"`
	Statuses         []Status   `json:"statuses"`
	Errors           []ErrorItem `json:"errors"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Message struct {
	From        string          `json:"from"`
	ID          string          `json:"id"`
	Timestamp   string          `json:"timestamp"`
	Type        string          `json:"type"`
	Context     *MessageContext `json:"context,omitempty"`
	Text        *TextPayload    `json:"text,omitempty"`
	Image       *MediaPayload   `json:"image,omitempty"`
	Video       *MediaPayload   `json:"video,omitempty"`
	Audio       *MediaPayload   `json:"audio,omitempty"`
	Document    *MediaPayload   `json:"document,omitempty"`
	Sticker     *StickerPayload `json:"sticker,omitempty"`
	Location    *LocationPayload `json:"location,omitempty"`
	Contacts    []Contact       `json:"contacts,omitempty"`
	Reaction    *ReactionPayload `json:"reaction,omitempty"`
	Interactive *InteractivePayload `json:"interactive,omitempty"`
	Button      *ButtonPayload `json:"button,omitempty"`
	Order       *OrderPayload  `json:"order,omitempty"`
	System      *SystemPayload `json:"system,omitempty"`
	Referral    *ReferralPayload `json:"referral,omitempty"`
	Errors      []ErrorItem    `json:"errors,omitempty"`
}

type TextPayload struct {
	Body string `json:"body"`
}

type MediaPayload struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	Animated bool   `json:"animated,omitempty"`
	Voice    bool   `json:"voice,omitempty"`
	URL      string `json:"url,omitempty"`
}

type StickerPayload struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Animated bool   `json:"animated"`
	URL      string `json:"url,omitempty"`
}

type LocationPayload struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type ReactionPayload struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type InteractivePayload struct {
	Type     string `json:"type"`
	ButtonReply *InteractiveButtonReply `json:"button_reply,omitempty"`
	ListReply  *InteractiveListReply `json:"list_reply,omitempty"`
	NFMReply   *InteractiveNFMReply `json:"nfm_reply,omitempty"`
}

type InteractiveButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type InteractiveListReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type InteractiveNFMReply struct {
	Name              string `json:"name"`
	ResponseJSON      string `json:"response_json"`
	Body              string `json:"body,omitempty"`
}

type ButtonPayload struct {
	Payload string `json:"payload"`
	Text    string `json:"text"`
}

type OrderPayload struct {
	CatalogID    string            `json:"catalog_id"`
	ProductItems []OrderProductItem `json:"product_items"`
	Text         string            `json:"text,omitempty"`
}

type OrderProductItem struct {
	ProductRetailerID string  `json:"product_retailer_id"`
	Quantity          string  `json:"quantity"`
	ItemPrice         string  `json:"item_price"`
	Currency          string  `json:"currency"`
}

type SystemPayload struct {
	Body    string `json:"body"`
	Type    string `json:"type"`
	WaID    string `json:"wa_id"`
}

type ReferralPayload struct {
	SourceURL     string `json:"source_url"`
	SourceType    string `json:"source_type"`
	SourceID      string `json:"source_id"`
	Headline      string `json:"headline"`
	Body          string `json:"body"`
	MediaURL      string `json:"media_url,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	CtwaCLID      string `json:"ctwa_clid,omitempty"`
}

type Status struct {
	ID           string        `json:"id"`
	RecipientID  string        `json:"recipient_id"`
	Status       string        `json:"status"`
	Timestamp    string        `json:"timestamp"`
	Conversation *Conversation `json:"conversation,omitempty"`
	Pricing      *Pricing      `json:"pricing,omitempty"`
	Errors       []ErrorItem   `json:"errors,omitempty"`
}

type Conversation struct {
	ID                string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp"`
	Origin            *Origin `json:"origin,omitempty"`
}

type Origin struct {
	Type string `json:"type"`
}

type Pricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

type ErrorItem struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Href    string `json:"href,omitempty"`
}

// ── Template Management Types ─────────────────────────────────────────────────

type TemplateNode struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Language     string `json:"language"`
	Status       string `json:"status"`
	Components   []TemplateComponentNode `json:"components"`
	QualityScore *TemplateQualityScore `json:"quality_score,omitempty"`
	RejectedReason string `json:"rejected_reason,omitempty"`
}

type TemplateComponentNode struct {
	Type     string `json:"type"`
	Format   string `json:"format,omitempty"`
	Text     string `json:"text,omitempty"`
	Buttons  []TemplateButtonNode `json:"buttons,omitempty"`
	Example  *TemplateExample `json:"example,omitempty"`
}

type TemplateButtonNode struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	URL         string `json:"url,omitempty"`
	FlowID      string `json:"flow_id,omitempty"`
}

type TemplateExample struct {
	HeaderText   []string `json:"header_text,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
	FooterText   []string `json:"footer_text,omitempty"`
	Buttons      [][]string `json:"buttons,omitempty"`
}

type TemplateQualityScore struct {
	Date   string   `json:"date"`
	Score  string   `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

type TemplateListResponse struct {
	Data   []TemplateNode `json:"data"`
	Paging *Paging        `json:"paging,omitempty"`
}

type CreateTemplateRequest struct {
	Name       string                        `json:"name"`
	Category   string                        `json:"category"`
	Language   string                        `json:"language"`
	Components []CreateTemplateComponent     `json:"components"`
}

type CreateTemplateComponent struct {
	Type    string `json:"type"`
	Format  string `json:"format,omitempty"`
	Text    string `json:"text,omitempty"`
	Buttons []CreateTemplateButton `json:"buttons,omitempty"`
	Example *CreateTemplateExample `json:"example,omitempty"`
}

type CreateTemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	URL         string `json:"url,omitempty"`
	Example     string `json:"example,omitempty"`
}

type CreateTemplateExample struct {
	HeaderText []string `json:"header_text,omitempty"`
	BodyText   [][]string `json:"body_text,omitempty"`
	FooterText []string `json:"footer_text,omitempty"`
	Buttons    [][]string `json:"buttons,omitempty"`
}

type CreateTemplateResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

type UpdateTemplateRequest struct {
	Category string                        `json:"category,omitempty"`
	Components []CreateTemplateComponent   `json:"components,omitempty"`
}

// ── Phone Number Types ────────────────────────────────────────────────────────

type PhoneNumber struct {
	ID                     string `json:"id"`
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	QualityRating          string `json:"quality_rating"`
	CodeVerificationStatus string `json:"code_verification_status"`
	Status                 string `json:"status"`
	PlatformType           string `json:"platform_type"`
	NameStatus             string `json:"name_status"`
}

type PhoneNumberListResponse struct {
	Data   []PhoneNumber `json:"data"`
	Paging *Paging       `json:"paging,omitempty"`
}

type Paging struct {
	Cursors *Cursors `json:"cursors,omitempty"`
	Next    string   `json:"next,omitempty"`
	Previous string  `json:"previous,omitempty"`
}

type Cursors struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// ── Webhook Verification ──────────────────────────────────────────────────────

type WebhookVerificationRequest struct {
	Mode      string `json:"hub.mode"`
	Token     string `json:"hub.verify_token"`
	Challenge string `json:"hub.challenge"`
}

// ── Typing Indicator ──────────────────────────────────────────────────────────

type TypingIndicator struct {
	Type string `json:"type"`
}
