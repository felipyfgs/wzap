package chatwoot

import (
	"strings"
)

type cloudWebhookEnvelope struct {
	Object string              `json:"object"`
	Entry  []cloudWebhookEntry `json:"entry"`
}

type cloudWebhookEntry struct {
	ID      string               `json:"id"`
	Changes []cloudWebhookChange `json:"changes"`
}

type cloudWebhookChange struct {
	Value cloudWebhookValue `json:"value"`
	Field string            `json:"field"`
}

type cloudWebhookValue struct {
	MessagingProduct string                `json:"messaging_product"`
	Metadata         *cloudWebhookMetadata `json:"metadata,omitempty"`
	Messages         []map[string]any      `json:"messages,omitempty"`
	MessageEchoes    []map[string]any      `json:"message_echoes,omitempty"`
	Contacts         []map[string]any      `json:"contacts,omitempty"`
	Statuses         []map[string]any      `json:"statuses,omitempty"`
	Errors           []map[string]any      `json:"errors,omitempty"`
}

type cloudWebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

func buildCloudWebhookEnvelope(sessionPhone string, outgoing bool, msg map[string]any, contact map[string]any) *cloudWebhookEnvelope {
	if msg == nil {
		return nil
	}

	field := "messages"
	value := cloudWebhookValue{
		MessagingProduct: "whatsapp",
		Metadata: &cloudWebhookMetadata{
			DisplayPhoneNumber: sessionPhone,
			PhoneNumberID:      sessionPhone,
		},
		Statuses: []map[string]any{},
		Errors:   []map[string]any{},
	}
	if outgoing {
		field = "smb_message_echoes"
		value.MessageEchoes = []map[string]any{msg}
	} else {
		value.Messages = []map[string]any{msg}
		value.Contacts = []map[string]any{}
	}

	envelope := &cloudWebhookEnvelope{
		Object: "whatsapp_business_account",
		Entry: []cloudWebhookEntry{
			{
				ID: sessionPhone,
				Changes: []cloudWebhookChange{
					{
						Field: field,
						Value: value,
					},
				},
			},
		},
	}

	if !outgoing && contact != nil {
		envelope.Entry[0].Changes[0].Value.Contacts = []map[string]any{contact}
	}

	return envelope
}

func buildCloudTextMessage(body, msgID, from, timestamp string) map[string]any {
	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "text",
		"text": map[string]any{
			"body": body,
		},
	}
}

func buildCloudMediaMessage(mediaType, link, mimeType, caption, filename, msgID, from, timestamp string) map[string]any {
	// Chatwoot Cloud inbox resolves media via `id` → GET /v{version}/{phone}/{id}
	// (handled by CloudAPIHandler.GetMedia). `link` is kept as a fallback for
	// clients that read it directly.
	mediaObj := map[string]any{
		"id":        msgID,
		"link":      link,
		"mime_type": mimeType,
	}
	if caption != "" {
		mediaObj["caption"] = caption
	}
	if filename != "" {
		mediaObj["filename"] = filename
	}

	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      mediaType,
		mediaType:   mediaObj,
	}
}

func buildCloudLocationMessage(lat, lng float64, name, address, msgID, from, timestamp string) map[string]any {
	loc := map[string]any{
		"latitude":  lat,
		"longitude": lng,
	}
	if name != "" {
		loc["name"] = name
	}
	if address != "" {
		loc["address"] = address
	}

	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "location",
		"location":  loc,
	}
}

func buildCloudContactMessage(contacts []map[string]any, msgID, from, timestamp string) map[string]any {
	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "contacts",
		"contacts":  contacts,
	}
}

func buildCloudContact(phone, name string) map[string]any {
	contact := map[string]any{}
	if name != "" {
		contact["profile"] = map[string]any{
			"name": name,
		}
	}
	if phone != "" {
		contact["wa_id"] = strings.TrimPrefix(phone, "+")
	}
	return contact
}

func cloudMediaType(waType string) string {
	switch waType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "document":
		return "document"
	case "sticker":
		return "sticker"
	default:
		return "document"
	}
}

func parseVCardToCloudContacts(vcard, displayName string) []map[string]any {
	if vcard == "" {
		return []map[string]any{
			{"name": map[string]any{"formatted_name": displayName}},
		}
	}

	lines := strings.Split(vcard, "\n")
	var phones []map[string]any
	fn := displayName

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FN:") {
			fn = strings.TrimPrefix(line, "FN:")
		}
		if strings.HasPrefix(line, "TEL") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				phone := parts[1]
				phones = append(phones, map[string]any{
					"phone": phone,
					"type":  "CELL",
					"wa_id": strings.ReplaceAll(phone, "+", ""),
				})
			}
		}
	}

	contact := map[string]any{
		"name": map[string]any{
			"formatted_name": fn,
		},
	}
	if len(phones) > 0 {
		contact["phones"] = phones
	}

	return []map[string]any{contact}
}
