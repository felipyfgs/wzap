package broker

import (
	"context"
	"fmt"
	"time"

	"wzap/internal/config"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

type Nats struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

func New(cfg *config.Config) (*Nats, error) {
	nc, err := nats.Connect(cfg.NatsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream instance: %w", err)
	}

	// Create stream if it doesn't exist
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamConfig := jetstream.StreamConfig{
		Name:     "WZAP_EVENTS",
		Subjects: []string{"wzap.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	}

	_, err = js.CreateOrUpdateStream(ctx, streamConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create/update NATS WZAP_EVENTS stream")
	}

	webhookStreamConfig := jetstream.StreamConfig{
		Name:     "WZAP_WEBHOOKS",
		Subjects: []string{"wzap.webhook.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   24 * time.Hour,
	}

	_, err = js.CreateOrUpdateStream(ctx, webhookStreamConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create/update NATS WZAP_WEBHOOKS stream")
	}

	log.Info().Msg("Successfully connected to NATS JetStream")

	return &Nats{
		Conn: nc,
		JS:   js,
	}, nil
}

func (n *Nats) Close() {
	if n.Conn != nil {
		log.Info().Msg("Closing NATS connection")
		n.Conn.Drain()
		n.Conn.Close()
	}
}

func (n *Nats) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := n.JS.Publish(ctx, subject, data)
	return err
}

func (n *Nats) Health() error {
	if !n.Conn.IsConnected() {
		return fmt.Errorf("NATS not connected")
	}
	return nil
}
