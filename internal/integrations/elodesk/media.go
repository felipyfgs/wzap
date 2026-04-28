package elodesk

import (
	"encoding/base64"
	"net/http"
	"path/filepath"
	"strings"
)

// Helpers de mídia. Espelham o que existe em internal/integrations/chatwoot/
// (extractors.go + media.go) — duplicação aceita até a 3ª integração, ver
// design.md D1.

const maxMediaBytes int64 = 256 * 1024 * 1024 // 256MB — limite do WhatsApp Cloud

type mediaInfo struct {
	DirectPath    string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    int
	MimeType      string
	MediaType     string
	FileName      string
}

var mediaTypeMap = map[string]string{
	"imageMessage":    "image",
	"videoMessage":    "video",
	"audioMessage":    "audio",
	"documentMessage": "document",
	"stickerMessage":  "sticker",
}

func getFloatField(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// extractMediaInfo procura os primeiros campos de mídia conhecidos no payload
// whatsmeow e devolve o necessário pra fazer DownloadMediaByPath. Retorna nil
// quando nenhum dos tipos suportados está presente.
func extractMediaInfo(msg map[string]any) *mediaInfo {
	if msg == nil {
		return nil
	}
	for key, mt := range mediaTypeMap {
		sub := getMapField(msg, key)
		if sub == nil {
			continue
		}
		directPath := getStringField(sub, "directPath")
		if directPath == "" {
			continue
		}

		mediaKey, _ := base64.StdEncoding.DecodeString(getStringField(sub, "mediaKey"))
		fileSHA256, _ := base64.StdEncoding.DecodeString(getStringField(sub, "fileSHA256"))
		fileEncSHA256, _ := base64.StdEncoding.DecodeString(getStringField(sub, "fileEncSHA256"))

		return &mediaInfo{
			DirectPath:    directPath,
			MediaKey:      mediaKey,
			FileSHA256:    fileSHA256,
			FileEncSHA256: fileEncSHA256,
			FileLength:    int(getFloatField(sub, "fileLength")),
			MimeType:      getStringField(sub, "mimetype"),
			MediaType:     mt,
			FileName:      getStringField(sub, "fileName"),
		}
	}
	return nil
}

// detectMIME devolve um mime-type provável a partir dos primeiros bytes do
// arquivo. Usado quando o whatsmeow não preenche `mimetype` (acontece com
// alguns stickers e voice notes antigas).
func detectMIME(data []byte) string {
	mt := http.DetectContentType(data)
	if mt != "" && mt != "application/octet-stream" {
		return mt
	}
	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		if data[0] == 0x89 && string(data[1:4]) == "PNG" {
			return "image/png"
		}
		if string(data[:4]) == "RIFF" {
			return "audio/wav"
		}
	}
	if len(data) >= 2 && data[0] == 0x4F && data[1] == 0x67 {
		return "audio/ogg"
	}
	return "application/octet-stream"
}

func mimeTypeToExt(mt string) string {
	base := strings.TrimSpace(strings.Split(mt, ";")[0])
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
	case "video/3gpp":
		return ".3gp"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4", "audio/m4a":
		return ".m4a"
	case "audio/wav":
		return ".wav"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// resolveFilename combina FileName do payload e fallback baseado em mime/tipo
// pra dar um nome amigável ao arquivo no MinIO.
func resolveFilename(info *mediaInfo, mime string) string {
	if info.FileName != "" {
		return filepath.Base(info.FileName)
	}
	ext := mimeTypeToExt(mime)
	if ext == "" {
		ext = mimeTypeToExt(info.MimeType)
	}
	return info.MediaType + ext
}
