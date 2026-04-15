package wa

import (
	"context"
	"fmt"
	"os"
	"time"

	"wzap/internal/logger"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
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
		opCtx, opCancel := context.WithTimeout(context.Background(), 5*time.Second)
		switch evt.Event {
		case "code":
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

			if err := m.sessionRepo.UpdateQRCode(opCtx, sessionID, evt.Code); err != nil {
				logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to save QR code to database")
			} else {
				logger.Info().Str("component", "wa").Str("session", sessionID).Msg("QR code saved to database")
			}

		case "timeout":
			logger.Warn().Str("component", "wa").Str("session", sessionID).Msg("QR code timed out")
			if err := m.sessionRepo.UpdateQRCode(opCtx, sessionID, ""); err != nil {
				logger.Warn().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to clear QR code on timeout")
			}
			if err := m.sessionRepo.UpdateStatus(opCtx, sessionID, "disconnected"); err != nil {
				logger.Warn().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to update status to disconnected on QR timeout")
			}

			m.mu.Lock()
			if client, exists := m.clients[sessionID]; exists {
				client.Disconnect()
				delete(m.clients, sessionID)
			}
			m.mu.Unlock()

		case "success":
			logger.Info().Str("component", "wa").Str("session", sessionID).Msg("QR pairing completed")
			if err := m.sessionRepo.UpdateQRCode(opCtx, sessionID, ""); err != nil {
				logger.Warn().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to clear QR code on success")
			}
			if err := m.sessionRepo.UpdateStatus(opCtx, sessionID, "connected"); err != nil {
				logger.Warn().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to update status to connected")
			}
		}
		opCancel()
	}
}
