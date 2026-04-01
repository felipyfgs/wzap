package wa

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog/log"
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
		switch evt.Event {
		case "code":
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

			if err := m.sessionRepo.UpdateQRCode(context.Background(), sessionID, evt.Code); err != nil {
				log.Error().Err(err).Str("session", sessionID).Msg("Failed to save QR code to database")
			}
			log.Info().Str("session", sessionID).Msg("QR code saved to database")

		case "timeout":
			log.Warn().Str("session", sessionID).Msg("QR code timed out")
			_ = m.sessionRepo.UpdateQRCode(context.Background(), sessionID, "")
			_ = m.sessionRepo.UpdateStatus(context.Background(), sessionID, "disconnected")

			m.mu.Lock()
			if client, exists := m.clients[sessionID]; exists {
				client.Disconnect()
				delete(m.clients, sessionID)
			}
			m.mu.Unlock()

		case "success":
			log.Info().Str("session", sessionID).Msg("QR pairing completed")
			_ = m.sessionRepo.UpdateQRCode(context.Background(), sessionID, "")
			_ = m.sessionRepo.UpdateStatus(context.Background(), sessionID, "connected")
		}
	}
}
