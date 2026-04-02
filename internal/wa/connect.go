package wa

import (
	"context"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // registers the PostgreSQL driver used by whatsmeow sqlstore
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/logger"
	"wzap/internal/repo"
	"wzap/internal/webhook"
)

func NewManager(ctx context.Context, cfg *config.Config, sessionRepo *repo.SessionRepository, n *broker.NATS, d *webhook.Dispatcher) (*Manager, error) {
	waLevel, err := zerolog.ParseLevel(strings.ToLower(cfg.WALogLevel))
	if err != nil {
		waLevel = zerolog.InfoLevel
	}
	waLogger := waLog.Zerolog(logger.Logger().Level(waLevel).With().Str("module", "wzap").Logger())

	container, err := sqlstore.New(ctx, "postgres", cfg.DatabaseURL, waLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create whatsmeow store container: %w", err)
	}

	return &Manager{
		clients:      make(map[string]*whatsmeow.Client),
		sessionNames: make(map[string]string),
		ctx:          ctx,
		sessionRepo:  sessionRepo,
		container:    container,
		nats:         n,
		dispatcher:   d,
		cfg:          cfg,
		waLog:        waLogger,
	}, nil
}

// ReconnectAll reconnects all previously paired devices on startup.
func (m *Manager) ReconnectAll(ctx context.Context) error {
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all devices: %w", err)
	}

	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		jidStr := device.ID.String()
		sessionID, err := m.sessionRepo.FindSessionIDByJID(ctx, jidStr)
		if err != nil {
			logger.Warn().Str("jid", jidStr).Msg("Orphan device without matching session, removing from sqlstore")
			if delErr := device.Delete(ctx); delErr != nil {
				logger.Error().Err(delErr).Str("jid", jidStr).Msg("Failed to delete orphan device")
			}
			continue
		}

		// Buscar e armazenar nome da sessão
		session, err := m.sessionRepo.FindByID(ctx, sessionID)
		if err == nil {
			m.mu.Lock()
			m.sessionNames[sessionID] = session.Name
			m.mu.Unlock()
		}

		client := whatsmeow.NewClient(device, m.waLog)
		client.AddEventHandler(func(evt interface{}) {
			m.handleEvent(sessionID, evt)
		})

		m.mu.Lock()
		m.clients[sessionID] = client
		m.mu.Unlock()

		if err := client.Connect(); err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to reconnect session")
			continue
		}

		if err := m.sessionRepo.SetConnected(ctx, sessionID, 1); err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to set connected status")
		}
		if err := m.sessionRepo.UpdateStatus(ctx, sessionID, "connected"); err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to update status to connected")
		}
		logger.Info().Str("session", sessionID).Str("jid", jidStr).Msg("Reconnected session")
	}

	return nil
}

// GetClient returns an existing connected client.
func (m *Manager) GetClient(sessionID string) (*whatsmeow.Client, error) {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not connected", sessionID)
	}
	return client, nil
}

// Connect connects or pairs a session.
func (m *Manager) Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error) {
	// Phase 1: check for an existing client without a write lock.
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if exists {
		if client.IsConnected() {
			return client, nil, nil
		}
		err := client.Connect()
		return client, nil, err
	}

	// Phase 2: load device and build client — no lock held, no network calls.
	deviceJID, err := m.sessionRepo.GetJID(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("session not found: %w", err)
	}

	var device *store.Device

	if deviceJID != "" {
		jid, parseErr := types.ParseJID(deviceJID)
		if parseErr == nil {
			device, err = m.container.GetDevice(ctx, jid)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to load device from store, creating new")
			}
		}
	}

	if device == nil {
		device = m.container.NewDevice()
	}

	client = whatsmeow.NewClient(device, m.waLog)

	// Buscar e armazenar nome da sessão
	session, err := m.sessionRepo.FindByID(ctx, sessionID)
	if err == nil {
		m.mu.Lock()
		m.sessionNames[sessionID] = session.Name
		m.mu.Unlock()
	}

	client.AddEventHandler(func(evt interface{}) {
		m.handleEvent(sessionID, evt)
	})

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			if client.Store.ID != nil {
				jidStr := client.Store.ID.String()
				if err := m.sessionRepo.UpdateJID(m.ctx, sessionID, jidStr); err != nil {
					logger.Error().Err(err).Str("session", sessionID).Str("jid", jidStr).Msg("Failed to update jid")
				}
				logger.Info().Str("session", sessionID).Str("jid", jidStr).Msg("Session paired")
			}
		case *events.PairSuccess:
			jidStr := v.ID.String()
			if err := m.sessionRepo.UpdateJID(m.ctx, sessionID, jidStr); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Str("jid", jidStr).Msg("Failed to update jid on pair")
			}
			_ = m.sessionRepo.UpdateQRCode(m.ctx, sessionID, "")
			logger.Info().Str("session", sessionID).Str("jid", jidStr).Msg("QR pairing successful")
		case *events.Disconnected:
			if err := m.sessionRepo.SetConnected(m.ctx, sessionID, 0); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Msg("Failed to set disconnected status")
			}
		case *events.LoggedOut:
			if err := client.Store.Delete(m.ctx); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Msg("Failed to delete device from sqlstore on logout")
			}
			m.mu.Lock()
			delete(m.clients, sessionID)
			m.mu.Unlock()
			if err := m.sessionRepo.ClearDevice(m.ctx, sessionID); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Msg("Failed to clear device on logout")
			}
			_ = m.sessionRepo.UpdateQRCode(m.ctx, sessionID, "")
		}
	})

	// Phase 3: insert into map under write lock — map operation only, no I/O.
	m.mu.Lock()
	if existing, ok := m.clients[sessionID]; ok {
		m.mu.Unlock()
		if existing.IsConnected() {
			return existing, nil, nil
		}
		return existing, nil, existing.Connect()
	}
	m.clients[sessionID] = client
	m.mu.Unlock()

	// Phase 4: connect outside the lock so other goroutines are not blocked.
	if client.Store.ID == nil {
		qrChan, qrErr := client.GetQRChannel(ctx)
		if qrErr != nil {
			m.mu.Lock()
			delete(m.clients, sessionID)
			m.mu.Unlock()
			return nil, nil, fmt.Errorf("failed to get QR channel: %w", qrErr)
		}
		if err = client.Connect(); err != nil {
			m.mu.Lock()
			delete(m.clients, sessionID)
			m.mu.Unlock()
			return nil, nil, err
		}

		_ = m.sessionRepo.UpdateStatus(m.ctx, sessionID, "connecting")
		go m.consumeQRChannel(sessionID, qrChan)

		return client, qrChan, nil
	}

	if err = client.Connect(); err != nil {
		m.mu.Lock()
		delete(m.clients, sessionID)
		m.mu.Unlock()
		return nil, nil, err
	}
	return client, nil, nil
}

// Disconnect disconnects a session without removing the device from the sqlstore.
func (m *Manager) Disconnect(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[sessionID]
	if exists {
		client.Disconnect()
		delete(m.clients, sessionID)
	}

	if err := m.sessionRepo.SetConnected(m.ctx, sessionID, 0); err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to set disconnected status")
	}
	return nil
}

// Logout sends an unpair request to WhatsApp, disconnects and removes the
// device from the whatsmeow sqlstore. This is the proper cleanup for session
// deletion — it prevents orphan devices in the sqlstore tables.
func (m *Manager) Logout(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	client, exists := m.clients[sessionID]
	if exists {
		delete(m.clients, sessionID)
	}
	m.mu.Unlock()

	if exists && client.Store.ID != nil {
		if err := client.Logout(ctx); err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Msg("Logout request failed, forcing device cleanup")
			client.Disconnect()
			if err := client.Store.Delete(ctx); err != nil {
				logger.Error().Err(err).Str("session", sessionID).Msg("Failed to delete device from sqlstore")
			}
		}
	} else if exists {
		client.Disconnect()
	}

	if err := m.sessionRepo.ClearDevice(ctx, sessionID); err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to clear device on logout")
	}
	return nil
}

func (m *Manager) PairPhone(ctx context.Context, sessionID, phone string) (string, error) {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if !exists {
		_, _, err := m.Connect(ctx, sessionID)
		if err != nil {
			return "", fmt.Errorf("failed to connect session for pairing: %w", err)
		}
		m.mu.RLock()
		client, exists = m.clients[sessionID]
		m.mu.RUnlock()
		if !exists {
			return "", fmt.Errorf("session client not available after connect")
		}
	}

	code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "wzap")
	if err != nil {
		return "", fmt.Errorf("failed to pair phone: %w", err)
	}
	return code, nil
}

func (m *Manager) Reconnect(ctx context.Context, sessionID string) error {
	_ = m.Disconnect(sessionID)
	_, _, err := m.Connect(ctx, sessionID)
	return err
}
