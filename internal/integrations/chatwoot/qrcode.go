package chatwoot

import (
	"github.com/skip2/go-qrcode"
)

func generateQRCodePNG(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, 256)
}
