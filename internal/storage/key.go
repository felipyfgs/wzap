package storage

import (
	"fmt"
	"mime"
	"strings"
	"time"
)

type MediaKeyParams struct {
	SessionID string
	ChatJID   string
	FromMe    bool
	MessageID string
	MimeType  string
	Timestamp time.Time
}

func MediaObjectKey(p MediaKeyParams) string {
	direction := "incoming"
	if p.FromMe {
		direction = "outgoing"
	}

	date := p.Timestamp.UTC().Format("2006-01-02")

	mediaCategory := mediaCategory(p.MimeType)

	chatJID := sanitizeJID(p.ChatJID)

	ext := extensionFromMime(p.MimeType)

	key := fmt.Sprintf("%s/%s/%s/%s/%s/%s", p.SessionID, chatJID, direction, date, mediaCategory, p.MessageID)
	if ext != "" {
		key += ext
	}
	return key
}

func mediaCategory(mimeType string) string {
	if mimeType == "" {
		return "other"
	}
	major := strings.SplitN(mimeType, "/", 2)[0]
	switch major {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "application":
		return "document"
	default:
		return "other"
	}
}

func extensionFromMime(mimeType string) string {
	if mimeType == "" {
		return ""
	}

	exts, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(exts) > 0 {
		return exts[len(exts)-1]
	}

	parts := strings.SplitN(mimeType, "/", 2)
	if len(parts) == 2 {
		sub := parts[1]
		sub = strings.TrimPrefix(sub, "x-")
		if idx := strings.Index(sub, ";"); idx != -1 {
			sub = sub[:idx]
		}
		return "." + sub
	}

	return ""
}

func sanitizeJID(jid string) string {
	if jid == "" {
		return "_unknown"
	}
	return strings.ReplaceAll(jid, ":", "_")
}
