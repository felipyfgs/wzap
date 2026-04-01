package wa

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"wzap/internal/logger"
)

// GetQRCode reads the latest QR code from the database.
func (m *Manager) GetQRCode(ctx context.Context, sessionID string) (string, error) {
	session, err := m.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %w", err)
	}
	return session.QRCode, nil
}

// consumeQRChannel reads QR events in background and saves raw QR text to DB.
func (m *Manager) consumeQRChannel(sessionID string, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

			if err := m.sessionRepo.UpdateQRCode(m.ctx, sessionID, evt.Code); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Msg("Failed to save QR code to database")
			}
			logger.Info().Str("session", sessionID).Msg("QR code saved to database")

		case "timeout":
			logger.Warn().Str("session", sessionID).Msg("QR code timed out")
			_ = m.sessionRepo.UpdateQRCode(m.ctx, sessionID, "")
			_ = m.sessionRepo.UpdateStatus(m.ctx, sessionID, "disconnected")

			m.mu.Lock()
			if client, exists := m.clients[sessionID]; exists {
				client.Disconnect()
				delete(m.clients, sessionID)
			}
			m.mu.Unlock()

		case "success":
			logger.Info().Str("session", sessionID).Msg("QR pairing completed")
			_ = m.sessionRepo.UpdateQRCode(m.ctx, sessionID, "")
			_ = m.sessionRepo.UpdateStatus(m.ctx, sessionID, "connected")
		}
	}
}
