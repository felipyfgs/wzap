package chatwoot

import (
	"context"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"

	"wzap/internal/logger"
)

func (s *Service) processConnected(ctx context.Context, cfg *Config, _ []byte) {
	now := time.Now()
	if v, ok := s.lastBotNotify.Load(cfg.SessionID); ok {
		if lastTime, valid := v.(time.Time); valid && now.Sub(lastTime) < 30*time.Second {
			return
		}
	}
	s.lastBotNotify.Store(cfg.SessionID, now)

	convID, ok := s.findOpenBotConversation(ctx, cfg)
	if !ok {
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "✅ WhatsApp conectado com sucesso!",
		MessageType: "incoming",
	})

	if cfg.ImportOnConnect {
		period := cfg.ImportPeriod
		if period == "" {
			period = "7d"
		}
		go func() {
			s.ImportHistoryAsync(context.Background(), cfg.SessionID, period, 0)
		}()
	}
}

func (s *Service) processDisconnected(ctx context.Context, cfg *Config, _ []byte) {
	convID, ok := s.findOpenBotConversation(ctx, cfg)
	if !ok {
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "⚠️ Sessão desconectada do WhatsApp.",
		MessageType: "incoming",
	})
}

func (s *Service) processQR(ctx context.Context, cfg *Config, payload []byte) {
	var data struct {
		Codes       []string `json:"Codes"`
		PairingCode string   `json:"PairingCode"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if len(data.Codes) == 0 {
		return
	}

	qrContent := data.Codes[0]

	convID, err := s.ensureBotConv(ctx, cfg)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for QR event")
		return
	}

	client := s.clientFn(cfg)
	qrPNG, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to generate QR code")
		return
	}

	caption := "⚡️ QR Code gerado com sucesso!\n\nEscaneie o QR Code abaixo no WhatsApp para conectar."
	if len(data.PairingCode) >= 4 {
		caption += fmt.Sprintf("\n\n*Código de pareamento:* %s-%s", data.PairingCode[:4], data.PairingCode[4:])
	} else if data.PairingCode != "" {
		caption += fmt.Sprintf("\n\n*Código de pareamento:* %s", data.PairingCode)
	}

	_, _ = client.CreateAttachment(ctx, convID, caption, "qrcode.png", qrPNG, "image/png", "incoming", "", 0, nil)
}
