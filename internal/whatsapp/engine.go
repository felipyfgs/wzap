package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver required by whatsmeow sqlstore
	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/config"
	"wzap/internal/broker"
	"wzap/internal/repo"
)

type Engine struct {
	clients map[string]*whatsmeow.Client
	mu      sync.RWMutex

	sessionRepo *repo.SessionRepository
	container   *sqlstore.Container
	nats        *broker.Nats
	cfg         *config.Config
	waLog       waLog.Logger
}

func NewEngine(cfg *config.Config, sessionRepo *repo.SessionRepository, n *broker.Nats) (*Engine, error) {
	waLogger := waLog.Stdout("wzap", cfg.WALogLevel, true)

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "postgres", cfg.DatabaseURL, waLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create whatsmeow store container: %w", err)
	}

	return &Engine{
		clients:     make(map[string]*whatsmeow.Client),
		sessionRepo: sessionRepo,
		container:   container,
		nats:        n,
		cfg:         cfg,
		waLog:       waLogger,
	}, nil
}

// ReconnectAll reconnects all previously paired devices on startup.
func (e *Engine) ReconnectAll(ctx context.Context) error {
	devices, err := e.container.GetAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all devices: %w", err)
	}

	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		jidStr := device.ID.String()
		sessionID, err := e.sessionRepo.FindSessionIDByJID(ctx, jidStr)
		if err != nil {
			log.Warn().Str("jid", jidStr).Msg("Device without matching session, skipping")
			continue
		}

		client := whatsmeow.NewClient(device, e.waLog)
		client.AddEventHandler(func(evt interface{}) {
			e.handleEvent(sessionID, evt)
		})

		e.mu.Lock()
		e.clients[sessionID] = client
		e.mu.Unlock()

		if err := client.Connect(); err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("Failed to reconnect session")
			continue
		}

		if err := e.sessionRepo.SetConnected(ctx, sessionID, 1); err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("Failed to set connected status")
		}
		if err := e.sessionRepo.UpdateStatus(ctx, sessionID, "connected"); err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("Failed to update status to connected")
		}
		log.Info().Str("session", sessionID).Str("jid", jidStr).Msg("Reconnected session")
	}

	return nil
}

// GetClient returns an existing connected client.
func (e *Engine) GetClient(sessionID string) (*whatsmeow.Client, error) {
	e.mu.RLock()
	client, exists := e.clients[sessionID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not connected", sessionID)
	}
	return client, nil
}

// Connect connects or pairs a session.
// For new devices (not yet paired), it starts a background goroutine that
// consumes the QR channel and saves each QR code to the database.
// Use GetQRCode to retrieve the latest QR from DB.
func (e *Engine) Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Already connected
	if client, exists := e.clients[sessionID]; exists {
		if client.IsConnected() {
			return client, nil, nil
		}
		err := client.Connect()
		return client, nil, err
	}

	deviceJID, err := e.sessionRepo.GetJid(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("session not found: %w", err)
	}

	var device *store.Device

	if deviceJID != "" {
		jid, parseErr := types.ParseJID(deviceJID)
		if parseErr == nil {
			device, err = e.container.GetDevice(ctx, jid)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to load device from store, creating new")
			}
		}
	}

	if device == nil {
		device = e.container.NewDevice()
	}

	client := whatsmeow.NewClient(device, e.waLog)

	// Event handler for NATS publishing
	client.AddEventHandler(func(evt interface{}) {
		e.handleEvent(sessionID, evt)
	})

	// Event handler for connection lifecycle
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			if client.Store.ID != nil {
				jidStr := client.Store.ID.String()
				if err := e.sessionRepo.UpdateJid(context.Background(), sessionID, jidStr); err != nil {
					log.Error().Err(err).Str("session", sessionID).Str("jid", jidStr).Msg("Failed to update jid")
				}
				log.Info().Str("session", sessionID).Str("jid", jidStr).Msg("Session paired")
			}
		case *events.PairSuccess:
			jidStr := v.ID.String()
			if err := e.sessionRepo.UpdateJid(context.Background(), sessionID, jidStr); err != nil {
				log.Error().Err(err).Str("session", sessionID).Str("jid", jidStr).Msg("Failed to update jid on pair")
			}
			_ = e.sessionRepo.UpdateQrCode(context.Background(), sessionID, "")
			log.Info().Str("session", sessionID).Str("jid", jidStr).Msg("QR pairing successful")
		case *events.Disconnected:
			if err := e.sessionRepo.SetConnected(context.Background(), sessionID, 0); err != nil {
				log.Error().Err(err).Str("session", sessionID).Msg("Failed to set disconnected status")
			}
		case *events.LoggedOut:
			if err := e.sessionRepo.ClearDevice(context.Background(), sessionID); err != nil {
				log.Error().Err(err).Str("session", sessionID).Msg("Failed to clear device on logout")
			}
			_ = e.sessionRepo.UpdateQrCode(context.Background(), sessionID, "")
		}
	})

	e.clients[sessionID] = client

	if client.Store.ID == nil {
		qrChan, qrErr := client.GetQRChannel(ctx)
		if qrErr != nil {
			return nil, nil, fmt.Errorf("failed to get QR channel: %w", qrErr)
		}
		if err = client.Connect(); err != nil {
			return nil, nil, err
		}

		_ = e.sessionRepo.UpdateStatus(context.Background(), sessionID, "connecting")

		go e.consumeQRChannel(sessionID, qrChan)

		return client, qrChan, nil
	}

	err = client.Connect()
	return client, nil, err
}

// consumeQRChannel reads QR events in background and saves raw QR text to DB.
func (e *Engine) consumeQRChannel(sessionID string, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println("QR code:", evt.Code)

			if err := e.sessionRepo.UpdateQrCode(context.Background(), sessionID, evt.Code); err != nil {
				log.Error().Err(err).Str("session", sessionID).Msg("Failed to save QR code to database")
			}
			log.Info().Str("session", sessionID).Msg("QR code saved to database")

		case "timeout":
			log.Warn().Str("session", sessionID).Msg("QR code timed out")
			_ = e.sessionRepo.UpdateQrCode(context.Background(), sessionID, "")
			_ = e.sessionRepo.UpdateStatus(context.Background(), sessionID, "disconnected")

			e.mu.Lock()
			if client, exists := e.clients[sessionID]; exists {
				client.Disconnect()
				delete(e.clients, sessionID)
			}
			e.mu.Unlock()

		case "success":
			log.Info().Str("session", sessionID).Msg("QR pairing completed")
			_ = e.sessionRepo.UpdateQrCode(context.Background(), sessionID, "")
			_ = e.sessionRepo.UpdateStatus(context.Background(), sessionID, "connected")
		}
	}
}

// GetQRCode reads the latest QR code from the database.
func (e *Engine) GetQRCode(ctx context.Context, sessionID string) (string, error) {
	session, err := e.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %w", err)
	}
	return session.QrCode, nil
}

// Disconnect disconnects a session.
func (e *Engine) Disconnect(sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	client, exists := e.clients[sessionID]
	if exists {
		client.Disconnect()
		delete(e.clients, sessionID)
	}

	if err := e.sessionRepo.SetConnected(context.Background(), sessionID, 0); err != nil {
		log.Error().Err(err).Str("session", sessionID).Msg("Failed to set disconnected status")
	}
	return nil
}

// handleEvent publishes whatsmeow events to NATS.
func (e *Engine) handleEvent(sessionID string, evt interface{}) {
	var natsData map[string]interface{}
	eventType := ""

	switch v := evt.(type) {
	case *events.Message:
		eventType = "messages.upsert"
		natsData = map[string]interface{}{
			"id":        v.Info.ID,
			"pushName":  v.Info.PushName,
			"message":   v.Message,
			"timestamp": v.Info.Timestamp.Unix(),
			"fromMe":    v.Info.IsFromMe,
		}
	default:
		return
	}

	payload := map[string]interface{}{
		"event_id":   uuid.NewString(),
		"session_id": sessionID,
		"event":      eventType,
		"data":       natsData,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("session", sessionID).Msg("Failed to marshal NATS event payload")
		return
	}
	if e.nats != nil {
		if err := e.nats.Publish(context.Background(), "wzap.events."+sessionID, bytes); err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("Failed to publish NATS event")
		}
	}
}
