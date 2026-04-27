package elodesk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"wzap/internal/logger"
	"wzap/internal/model"
)

const (
	streamInbound  = "ELODESK_INBOUND"
	streamOutbound = "ELODESK_OUTBOUND"
	workerCount    = 10
	maxDeliver     = 50
	ackWait        = 30 * time.Second
	deadLetterBase = "elodesk.deadletter"
)

var backoffSchedule = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
	30 * time.Second,
	60 * time.Second,
	5 * time.Minute,
}

type NATSClient interface {
	Publish(ctx context.Context, subject string, data []byte) error
	JetStream() jetstream.JetStream
}

type natsAdapter struct {
	js jetstream.JetStream
}

func (n *natsAdapter) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := n.js.Publish(ctx, subject, data)
	return err
}

func (n *natsAdapter) JetStream() jetstream.JetStream { return n.js }

type Consumer struct {
	nats    NATSClient
	service *Service
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewConsumer(js jetstream.JetStream, svc *Service) (*Consumer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	inboundCfg := jetstream.StreamConfig{
		Name:     streamInbound,
		Subjects: []string{"elodesk.inbound.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	}
	if _, err := js.CreateOrUpdateStream(ctx, inboundCfg); err != nil {
		return nil, fmt.Errorf("failed to create ELODESK_INBOUND stream: %w", err)
	}

	outboundCfg := jetstream.StreamConfig{
		Name:     streamOutbound,
		Subjects: []string{"elodesk.outbound.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   24 * time.Hour,
	}
	if _, err := js.CreateOrUpdateStream(ctx, outboundCfg); err != nil {
		return nil, fmt.Errorf("failed to create ELODESK_OUTBOUND stream: %w", err)
	}

	return &Consumer{
		nats:    &natsAdapter{js: js},
		service: svc,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	consCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	inboundCons, err := c.nats.JetStream().CreateOrUpdateConsumer(consCtx, streamInbound, jetstream.ConsumerConfig{
		Durable:        "elodesk-inbound-worker",
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        ackWait,
		MaxDeliver:     maxDeliver,
		BackOff:        backoffSchedule,
		FilterSubjects: []string{"elodesk.inbound.>"},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create inbound consumer: %w", err)
	}

	outboundCons, err := c.nats.JetStream().CreateOrUpdateConsumer(consCtx, streamOutbound, jetstream.ConsumerConfig{
		Durable:        "elodesk-outbound-worker",
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        ackWait,
		MaxDeliver:     maxDeliver,
		BackOff:        backoffSchedule,
		FilterSubjects: []string{"elodesk.outbound.>"},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create outbound consumer: %w", err)
	}

	for range workerCount {
		c.wg.Add(1)
		go c.worker(consCtx, inboundCons, c.processInbound)
	}
	for range workerCount {
		c.wg.Add(1)
		go c.worker(consCtx, outboundCons, c.processOutbound)
	}

	return nil
}

func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		logger.Warn().Str("component", "elodesk").Msg("consumer drain timed out after 30s")
	}
}

func (c *Consumer) worker(ctx context.Context, cons jetstream.Consumer, processFn func(context.Context, jetstream.Msg)) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := cons.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			continue
		}
		for msg := range msgs.Messages() {
			select {
			case <-ctx.Done():
				_ = msg.Nak()
				return
			default:
			}
			processFn(ctx, msg)
		}
	}
}

type inboundEnvelope struct {
	SessionID string          `json:"sessionID"`
	Event     model.EventType `json:"event"`
	Payload   json.RawMessage `json:"payload"`
}

type outboundEnvelope struct {
	SessionID string          `json:"sessionID"`
	Payload   json.RawMessage `json:"payload"`
}

func (c *Consumer) processInbound(ctx context.Context, msg jetstream.Msg) {
	meta, err := msg.Metadata()
	if err != nil {
		_ = msg.Nak()
		return
	}
	if meta.NumDelivered >= maxDeliver {
		c.sendToDeadLetter(ctx, "inbound", msg.Data())
		_ = msg.Ack()
		return
	}

	var env inboundEnvelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Msg("failed to unmarshal inbound envelope")
		_ = msg.Ack()
		return
	}

	if !c.service.cb.Allow(env.SessionID) {
		_ = msg.Nak()
		return
	}

	if err := c.service.processInboundEvent(ctx, env.SessionID, env.Event, env.Payload); err != nil {
		if ctx.Err() != nil {
			_ = msg.Nak()
			return
		}
		c.service.cb.RecordFailure(env.SessionID)
		_ = msg.Nak()
		return
	}

	c.service.cb.RecordSuccess(env.SessionID)
	_ = msg.Ack()
}

func (c *Consumer) processOutbound(ctx context.Context, msg jetstream.Msg) {
	meta, err := msg.Metadata()
	if err != nil {
		_ = msg.Nak()
		return
	}
	if meta.NumDelivered >= maxDeliver {
		c.sendToDeadLetter(ctx, "outbound", msg.Data())
		_ = msg.Ack()
		return
	}

	var env outboundEnvelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Msg("failed to unmarshal outbound envelope")
		_ = msg.Ack()
		return
	}

	if !c.service.cb.Allow(env.SessionID) {
		_ = msg.Nak()
		return
	}

	if err := c.service.processOutbound(ctx, env.SessionID, env.Payload); err != nil {
		if ctx.Err() != nil {
			_ = msg.Nak()
			return
		}
		c.service.cb.RecordFailure(env.SessionID)
		_ = msg.Nak()
		return
	}

	c.service.cb.RecordSuccess(env.SessionID)
	_ = msg.Ack()
}

func (c *Consumer) sendToDeadLetter(ctx context.Context, direction string, data []byte) {
	subject := fmt.Sprintf("%s.%s", deadLetterBase, direction)
	if err := c.nats.Publish(ctx, subject, data); err != nil {
		logger.Error().Str("component", "elodesk").Err(err).Str("direction", direction).Msg("failed to publish to dead letter")
	} else {
		logger.Error().Str("component", "elodesk").Str("direction", direction).Msg("message sent to dead letter after max retries")
	}
}

func publishInbound(ctx context.Context, js jetstream.JetStream, sessionID string, event model.EventType, payload []byte) error {
	env := inboundEnvelope{SessionID: sessionID, Event: event, Payload: payload}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal inbound envelope: %w", err)
	}
	subject := fmt.Sprintf("elodesk.inbound.%s", sessionID)
	natsMsg := &nats.Msg{Subject: subject, Data: data, Header: make(nats.Header)}
	_, err = js.PublishMsg(ctx, natsMsg)
	return err
}

func publishOutbound(ctx context.Context, js jetstream.JetStream, sessionID string, payload []byte) error {
	env := outboundEnvelope{SessionID: sessionID, Payload: payload}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal outbound envelope: %w", err)
	}
	subject := fmt.Sprintf("elodesk.outbound.%s", sessionID)
	natsMsg := &nats.Msg{Subject: subject, Data: data, Header: make(nats.Header)}
	_, err = js.PublishMsg(ctx, natsMsg)
	return err
}
