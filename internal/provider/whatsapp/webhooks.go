package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func ParseWebhook(body []byte, signature, appSecret string) (*WebhookNotification, error) {
	if appSecret != "" {
		if err := VerifySignature(body, signature, appSecret); err != nil {
			return nil, fmt.Errorf("invalid webhook signature: %w", err)
		}
	}

	var notification WebhookNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	return &notification, nil
}

func VerifySignature(body []byte, signature, appSecret string) error {
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	expected := computeHMAC(body, appSecret)

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func VerifyWebhook(mode, token, challenge, expectedToken string) (string, error) {
	if mode != "subscribe" {
		return "", fmt.Errorf("invalid webhook verification mode: %s", mode)
	}
	if token != expectedToken {
		return "", fmt.Errorf("webhook verify token mismatch")
	}
	return challenge, nil
}

func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
