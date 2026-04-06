package chatwoot

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
)

func GetMIMETypeAndExt(url string, data []byte) (mimeType, ext string) {
	if url != "" {
		if e := filepath.Ext(url); e != "" {
			ext = e
			switch strings.ToLower(e) {
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".png":
				mimeType = "image/png"
			case ".gif":
				mimeType = "image/gif"
			case ".webp":
				mimeType = "image/webp"
			case ".mp4":
				mimeType = "video/mp4"
			case ".ogg":
				mimeType = "audio/ogg"
			case ".mp3":
				mimeType = "audio/mpeg"
			case ".pdf":
				mimeType = "application/pdf"
			case ".doc", ".docx":
				mimeType = "application/msword"
			default:
				mimeType = "application/octet-stream"
			}
			return
		}
	}

	detected := http.DetectContentType(data)
	if detected != "" && detected != "application/octet-stream" {
		mimeType = detected
		ext = mimeTypeToExt(mimeType)
		return
	}

	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			mimeType = "image/jpeg"
			ext = ".jpg"
			return
		}
		if data[0] == 0x89 && string(data[1:4]) == "PNG" {
			mimeType = "image/png"
			ext = ".png"
			return
		}
		if string(data[:4]) == "RIFF" {
			mimeType = "audio/wav"
			ext = ".wav"
			return
		}
	}

	if len(data) >= 2 {
		if data[0] == 0x4F && data[1] == 0x67 {
			mimeType = "audio/ogg"
			ext = ".ogg"
			return
		}
	}

	mimeType = "application/octet-stream"
	ext = ""
	return
}

func mimeTypeToExt(mimeType string) string {
	base := strings.Split(mimeType, ";")[0]
	base = strings.TrimSpace(base)
	switch base {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

func httpGetWithContext(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
