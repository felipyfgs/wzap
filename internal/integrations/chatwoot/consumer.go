package chatwoot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

const (
	streamInbound  = "CW_INBOUND"
	streamOutbound = "CW_OUTBOUND"
	workerCount    = 10
	maxDeliver     = 50
	ackWait        = 30 * time.Second
	deadLetterBase = "cw.deadletter"
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
		Subjects: []string{"cw.inbound.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	}
	if _, err := js.CreateOrUpdateStream(ctx, inboundCfg); err != nil {
		return nil, fmt.Errorf("failed to create CW_INBOUND stream: %w", err)
	}

	outboundCfg := jetstream.StreamConfig{
		Name:     streamOutbound,
		Subjects: []string{"cw.outbound.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   24 * time.Hour,
	}
	if _, err := js.CreateOrUpdateStream(ctx, outboundCfg); err != nil {
		return nil, fmt.Errorf("failed to create CW_OUTBOUND stream: %w", err)
	}

	return &Consumer{
		nats:    &natsAdapter{js: js},
		service: svc,
	}, nil
}

func (c *Consumer) startQueueDepthPoller(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.pollQueueDepth(ctx)
			}
		}
	}()
}

func (c *Consumer) pollQueueDepth(ctx context.Context) {
	for _, stream := range []string{streamInbound, streamOutbound} {
		info, err := c.nats.JetStream().Stream(ctx, stream)
		if err != nil {
			continue
		}
		si, err := info.Info(ctx)
		if err != nil {
			continue
		}
		metrics.CWQueueDepth.WithLabelValues(stream).Set(float64(si.State.Msgs))
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	consCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	inboundCons, err := c.nats.JetStream().CreateOrUpdateConsumer(consCtx, streamInbound, jetstream.ConsumerConfig{
		Durable:        "cw-inbound-worker",
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        ackWait,
		MaxDeliver:     maxDeliver,
		BackOff:        backoffSchedule,
		FilterSubjects: []string{"cw.inbound.>"},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create inbound consumer: %w", err)
	}

	outboundCons, err := c.nats.JetStream().CreateOrUpdateConsumer(consCtx, streamOutbound, jetstream.ConsumerConfig{
		Durable:        "cw-outbound-worker",
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        ackWait,
		MaxDeliver:     maxDeliver,
		BackOff:        backoffSchedule,
		FilterSubjects: []string{"cw.outbound.>"},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create outbound consumer: %w", err)
	}

	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.worker(consCtx, inboundCons, c.processInboundWorker)
	}

	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.worker(consCtx, outboundCons, c.processOutboundWorker)
	}

	c.startQueueDepthPoller(consCtx)

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
		logger.Warn().Str("component", "chatwoot").Msg("consumer drain timed out after 30s")
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

func (c *Consumer) processInboundWorker(ctx context.Context, msg jetstream.Msg) {
	c.processInbound(ctx, msg)
}

func (c *Consumer) processOutboundWorker(ctx context.Context, msg jetstream.Msg) {
	c.processOutbound(ctx, msg)
}

type inboundEnvelope struct {
	SessionID string          `json:"sessionID"`
	Event     model.EventType `json:"event"`
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
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to unmarshal inbound envelope")
		_ = msg.Ack()
		return
	}

	spanCtx, span := startSpan(ctx, "chatwoot.consume",
		spanAttrs(env.SessionID, string(env.Event), "inbound")...)
	defer span.End()

	_, cbSpan := startSpan(spanCtx, "chatwoot.check_circuit_breaker",
		spanAttrs(env.SessionID, string(env.Event), "inbound")...)
	cbAllowed := c.service.cb.Allow(env.SessionID)
	cbSpan.End()

	if !cbAllowed {
		logger.Warn().Str("component", "chatwoot").Str("session", env.SessionID).Msg("circuit breaker OPEN, NAK inbound for retry")
		metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(cbOpen))
		_ = msg.Nak()
		return
	}

	start := time.Now()
	if err := c.service.processInboundEvent(spanCtx, env.SessionID, env.Event, env.Payload); err != nil {
		if ctx.Err() != nil {
			logger.Debug().Str("component", "chatwoot").Str("session", env.SessionID).Str("event", string(env.Event)).Msg("inbound processing interrupted by shutdown, will redeliver on restart")
			_ = msg.Nak()
			return
		}
		c.service.cb.RecordFailure(env.SessionID)
		metrics.CWMessagesFailed.WithLabelValues(env.SessionID, string(env.Event), "processing_error").Inc()
		metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(c.service.cb.get(env.SessionID).State()))
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", env.SessionID).Str("event", string(env.Event)).Msg("inbound processing error, will retry")
		_ = msg.Nak()
		return
	}

	c.service.cb.RecordSuccess(env.SessionID)
	metrics.CWMessagesSent.WithLabelValues(env.SessionID, string(env.Event), "inbound").Inc()
	metrics.CWMessageLatency.WithLabelValues(env.SessionID, string(env.Event), "inbound").Observe(time.Since(start).Seconds())
	metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(cbClosed))
	metrics.CWRetryCount.WithLabelValues(env.SessionID).Observe(float64(meta.NumDelivered - 1))
	_ = msg.Ack()
}

type outboundEnvelope struct {
	SessionID string          `json:"sessionID"`
	Payload   json.RawMessage `json:"payload"`
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
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to unmarshal outbound envelope")
		_ = msg.Ack()
		return
	}

	spanCtx, span := startSpan(ctx, "chatwoot.consume",
		spanAttrs(env.SessionID, "outbound", "outbound")...)
	defer span.End()

	_, cbSpan := startSpan(spanCtx, "chatwoot.check_circuit_breaker",
		spanAttrs(env.SessionID, "outbound", "outbound")...)
	cbAllowed := c.service.cb.Allow(env.SessionID)
	cbSpan.End()

	if !cbAllowed {
		logger.Warn().Str("component", "chatwoot").Str("session", env.SessionID).Msg("circuit breaker OPEN, NAK outbound for retry")
		metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(cbOpen))
		_ = msg.Nak()
		return
	}

	start := time.Now()
	if err := c.service.processOutboundWebhook(spanCtx, env.SessionID, env.Payload); err != nil {
		if ctx.Err() != nil {
			logger.Debug().Str("component", "chatwoot").Str("session", env.SessionID).Msg("outbound processing interrupted by shutdown, will redeliver on restart")
			_ = msg.Nak()
			return
		}
		c.service.cb.RecordFailure(env.SessionID)
		metrics.CWMessagesFailed.WithLabelValues(env.SessionID, "outbound", "processing_error").Inc()
		metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(c.service.cb.get(env.SessionID).State()))
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", env.SessionID).Msg("outbound processing error, will retry")
		_ = msg.Nak()
		return
	}

	c.service.cb.RecordSuccess(env.SessionID)
	metrics.CWMessagesSent.WithLabelValues(env.SessionID, "outbound", "outbound").Inc()
	metrics.CWMessageLatency.WithLabelValues(env.SessionID, "outbound", "outbound").Observe(time.Since(start).Seconds())
	metrics.CWCircuitBreakerState.WithLabelValues(env.SessionID).Set(float64(cbClosed))
	metrics.CWRetryCount.WithLabelValues(env.SessionID).Observe(float64(meta.NumDelivered - 1))
	_ = msg.Ack()
}

func (c *Consumer) sendToDeadLetter(ctx context.Context, direction string, data []byte) {
	subject := fmt.Sprintf("%s.%s", deadLetterBase, direction)
	if err := c.nats.Publish(ctx, subject, data); err != nil {
		logger.Error().Str("component", "chatwoot").Err(err).Str("direction", direction).Msg("failed to publish to dead letter")
	} else {
		logger.Error().Str("component", "chatwoot").Str("direction", direction).Msg("message sent to dead letter after max retries")
		metrics.CWDeadLetterCount.WithLabelValues(direction).Inc()
	}
}

func publishInbound(ctx context.Context, js jetstream.JetStream, sessionID string, event model.EventType, payload []byte) error {
	spanCtx, span := startSpan(ctx, "chatwoot.publish_nats",
		spanAttrs(sessionID, string(event), "inbound")...)
	defer span.End()

	env := inboundEnvelope{
		SessionID: sessionID,
		Event:     event,
		Payload:   payload,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal inbound envelope: %w", err)
	}
	subject := fmt.Sprintf("cw.inbound.%s", sessionID)
	natsMsg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  make(nats.Header),
	}
	InjectNATSHeaders(spanCtx, natsMsg)
	_, err = js.PublishMsg(spanCtx, natsMsg)
	return err
}

func publishOutbound(ctx context.Context, js jetstream.JetStream, sessionID string, payload []byte) error {
	spanCtx, span := startSpan(ctx, "chatwoot.publish_nats",
		spanAttrs(sessionID, "outbound", "outbound")...)
	defer span.End()

	env := outboundEnvelope{
		SessionID: sessionID,
		Payload:   payload,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal outbound envelope: %w", err)
	}
	subject := fmt.Sprintf("cw.outbound.%s", sessionID)
	natsMsg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  make(nats.Header),
	}
	InjectNATSHeaders(spanCtx, natsMsg)
	_, err = js.PublishMsg(spanCtx, natsMsg)
	return err
}
