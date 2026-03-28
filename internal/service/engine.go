package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver required by whatsmeow sqlstore
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/config"
	"wzap/internal/queue"
	"wzap/internal/repository"
)

type Engine struct {
	clients map[string]*whatsmeow.Client
	mu      sync.RWMutex

	sessionRepo *repository.SessionRepository
	container   *sqlstore.Container
	nats        *queue.Nats
	cfg         *config.Config
	waLog       waLog.Logger
}

func NewEngine(cfg *config.Config, sessionRepo *repository.SessionRepository, n *queue.Nats) (*Engine, error) {
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

		_ = e.sessionRepo.SetConnected(ctx, sessionID, true)
		_ = e.sessionRepo.UpdateStatus(ctx, sessionID, "READY")
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

	// Check if session has a saved device JID
	deviceJID, err := e.sessionRepo.GetDeviceJID(ctx, sessionID)
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
		switch evt.(type) {
		case *events.Connected:
			if client.Store.ID != nil {
				jidStr := client.Store.ID.String()
				_ = e.sessionRepo.UpdateDeviceJID(ctx, sessionID, jidStr)
				log.Info().Str("session", sessionID).Str("jid", jidStr).Msg("Session paired")
			}
		case *events.Disconnected:
			_ = e.sessionRepo.SetConnected(context.Background(), sessionID, false)
		case *events.LoggedOut:
			_ = e.sessionRepo.ClearDevice(context.Background(), sessionID)
		}
	})

	e.clients[sessionID] = client

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		err = client.Connect()
		return client, qrChan, err
	}

	err = client.Connect()
	return client, nil, err
}

// Disconnect disconnects a session.
func (e *Engine) Disconnect(sessionID string) error {
	e.mu.Lock()
	client, exists := e.clients[sessionID]
	if exists {
		client.Disconnect()
		delete(e.clients, sessionID)
	}
	e.mu.Unlock()

	_ = e.sessionRepo.SetConnected(context.Background(), sessionID, false)
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

	bytes, _ := json.Marshal(payload)
	if e.nats != nil {
		if err := e.nats.Publish(context.Background(), "wzap.events."+sessionID, bytes); err != nil {
			log.Error().Err(err).Msg("failed to publish NATS event")
		}
	}
}
