package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"wzap/internal/async"
	"wzap/internal/broker"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/repo"

	"github.com/nats-io/nats.go/jetstream"
)

type permanentDeliveryError struct {
	statusCode int
	err        error
}

func (e *permanentDeliveryError) Error() string { return e.err.Error() }
func (e *permanentDeliveryError) Unwrap() error { return e.err }

const (
	natsDeliverSubject = "wzap.webhook.deliver"
	httpTimeout        = 10 * time.Second
	maxDeliverAttempts = 5
	maxHTTPRetries     = 3
	httpRetryBaseDelay = 2 * time.Second
	maxWebhookPayload  = 512 * 1024 // 512 KB

	globalBackoffBase = 5 * time.Second
	globalBackoffMax  = 30 * time.Minute
)

type deliverMsg struct {
	WebhookID string          `json:"webhookId"`
	URL       string          `json:"url"`
	Secret    string          `json:"secret"`
	Payload   json.RawMessage `json:"payload"`
}

type WSBroadcaster interface {
	Broadcast(sessionID string, payload []byte)
}

type EventListener interface {
	OnEvent(ctx context.Context, sessionID string, event model.EventType, payload []byte)
}

type Dispatcher struct {
	webhookRepo       *repo.WebhookRepository
	nats              *broker.NATS
	httpClient        *http.Client
	globalWebhookURL  string
	globalFailures    atomic.Uint64
	globalLastAttempt atomic.Int64
	ws                WSBroadcaster
	listeners         []EventListener
	pool              *async.Pool
	wg                sync.WaitGroup
}

func New(webhookRepo *repo.WebhookRepository, nats *broker.NATS, globalWebhookURL string, pool *async.Pool) *Dispatcher {
	return &Dispatcher{
		webhookRepo:      webhookRepo,
		nats:             nats,
		httpClient:       &http.Client{Timeout: httpTimeout},
		globalWebhookURL: globalWebhookURL,
		pool:             pool,
	}
}

func (d *Dispatcher) SetWSBroadcaster(ws WSBroadcaster) {
	d.ws = ws
}

func (d *Dispatcher) AddListener(l EventListener) {
	d.listeners = append(d.listeners, l)
}

func (d *Dispatcher) DispatchAsync(sessionID string, eventType model.EventType, payload []byte) {
	_ = d.pool.Submit(func(_ context.Context) {
		d.Dispatch(sessionID, eventType, payload)
	})
}

// Dispatch looks up active webhooks for the session/event and delivers the payload.
func (d *Dispatcher) Dispatch(sessionID string, eventType model.EventType, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	webhooks, err := d.webhookRepo.FindActiveBySessionAndEvent(ctx, sessionID, string(eventType))
	if err != nil {
		logger.Error().Str("component", "webhook").Err(err).Str("session", sessionID).Str("event", string(eventType)).Msg("Failed to fetch webhooks for dispatch")
		return
	}

	logger.Debug().Str("component", "webhook").Str("session", sessionID).Str("event", string(eventType)).Int("webhooks", len(webhooks)).Str("globalURL", d.globalWebhookURL).Msg("Dispatching webhook")

	if len(payload) > maxWebhookPayload {
		logger.Debug().Str("component", "webhook").Str("session", sessionID).Str("event", string(eventType)).Int("size", len(payload)).Msg("Event payload too large for webhook delivery, skipping")
		if d.ws != nil {
			d.ws.Broadcast(sessionID, payload)
		}
		return
	}

	for _, wh := range webhooks {
		wh := wh
		if wh.NATSEnabled && d.nats != nil {
			_ = d.pool.Submit(func(ctx context.Context) {
				d.publishToNATS(wh, payload)
			})
		} else {
			_ = d.pool.Submit(func(ctx context.Context) {
				d.deliverHTTPWithRetry(wh.URL, wh.Secret, payload)
			})
		}
	}

	if d.globalWebhookURL != "" && d.shouldAttemptGlobal() {
		logger.Debug().Str("component", "webhook").Str("url", d.globalWebhookURL).Msg("Sending to global webhook")
		_ = d.pool.Submit(func(ctx context.Context) {
			d.deliverGlobalWebhook(payload)
		})
	}

	if d.ws != nil {
		d.ws.Broadcast(sessionID, payload)
	}

	for _, listener := range d.listeners {
		_ = d.pool.Submit(func(ctx context.Context) {
			listener.OnEvent(ctx, sessionID, eventType, payload)
		})
	}
}

func (d *Dispatcher) publishToNATS(wh model.Webhook, payload []byte) {
	msg := deliverMsg{
		WebhookID: wh.ID,
		URL:       wh.URL,
		Secret:    wh.Secret,
		Payload:   payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Str("component", "webhook").Err(err).Str("webhook", wh.ID).Msg("Failed to marshal NATS webhook delivery message")
		return
	}
	subject := natsDeliverSubject + "." + wh.ID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.nats.Publish(ctx, subject, data); err != nil {
		logger.Error().Str("component", "webhook").Err(err).Str("webhook", wh.ID).Msg("Failed to publish webhook delivery to NATS — falling back to direct dispatch")
		_ = d.pool.Submit(func(_ context.Context) {
			d.deliverHTTPWithRetry(wh.URL, wh.Secret, payload)
		})
	}
}

func (d *Dispatcher) deliverHTTPWithRetry(url, secret string, payload []byte) {
	for attempt := 0; attempt <= maxHTTPRetries; attempt++ {
		if err := d.deliverHTTPWithErr(url, secret, payload); err != nil {
			var permErr *permanentDeliveryError
			if errors.As(err, &permErr) {
				logger.Warn().Str("component", "webhook").Err(err).Str("url", url).Int("status", permErr.statusCode).Msg("Webhook HTTP delivery failed permanently (4xx, no retry)")
				return
			}
			if attempt < maxHTTPRetries {
				delay := httpRetryBaseDelay * time.Duration(1<<uint(attempt))
				logger.Warn().Str("component", "webhook").Err(err).Str("url", url).Int("attempt", attempt+1).Dur("retryIn", delay).Msg("Webhook HTTP delivery failed, retrying")
				time.Sleep(delay)
				continue
			}
			logger.Error().Str("component", "webhook").Err(err).Str("url", url).Msg("Webhook HTTP delivery failed after all retries")
		}
		return
	}
}

func (d *Dispatcher) shouldAttemptGlobal() bool {
	failures := d.globalFailures.Load()
	if failures == 0 {
		return true
	}

	backoff := globalBackoffBase * time.Duration(1<<min(failures-1, 10))
	if backoff > globalBackoffMax {
		backoff = globalBackoffMax
	}

	lastAttempt := time.Unix(0, d.globalLastAttempt.Load())
	return time.Since(lastAttempt) >= backoff
}

func (d *Dispatcher) deliverGlobalWebhook(payload []byte) {
	d.globalLastAttempt.Store(time.Now().UnixNano())

	if err := d.deliverHTTPWithErr(d.globalWebhookURL, "", payload); err != nil {
		failures := d.globalFailures.Add(1)
		var permErr *permanentDeliveryError
		if errors.As(err, &permErr) {
			logger.Warn().Str("component", "webhook").Err(err).Str("url", d.globalWebhookURL).Int("status", permErr.statusCode).
				Uint64("failures", failures).Msg("Global webhook URL returned 4xx, backing off")
		} else {
			logger.Warn().Str("component", "webhook").Err(err).Str("url", d.globalWebhookURL).
				Uint64("failures", failures).Msg("Global webhook delivery failed, backing off")
		}
		return
	}

	if failures := d.globalFailures.Swap(0); failures > 0 {
		logger.Info().Str("component", "webhook").Str("url", d.globalWebhookURL).Uint64("previousFailures", failures).Msg("Global webhook recovered")
	}
}

// StartConsumer starts the JetStream consumer that handles NATS-queued webhook deliveries.
func (d *Dispatcher) StartConsumer(ctx context.Context) {
	if d.nats == nil {
		return
	}

	consumerCfg := jetstream.ConsumerConfig{
		Name:          "webhook-dispatcher",
		FilterSubject: natsDeliverSubject + ".>",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    maxDeliverAttempts,
		BackOff:       []time.Duration{10 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute},
		AckWait:       15 * time.Second,
	}

	cons, err := d.nats.JS.CreateOrUpdateConsumer(ctx, "WZAP_WEBHOOKS", consumerCfg)
	if err != nil {
		logger.Warn().Str("component", "webhook").Err(err).Msg("Failed to create NATS webhook consumer — NATS-queued webhooks will fall back to direct dispatch")
		return
	}

	msgCtx, err := cons.Messages()
	if err != nil {
		logger.Warn().Str("component", "webhook").Err(err).Msg("Failed to subscribe to NATS webhook consumer")
		return
	}

	logger.Info().Str("component", "webhook").Msg("NATS webhook consumer started")

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer msgCtx.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg, err := msgCtx.Next()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Warn().Str("component", "webhook").Err(err).Msg("NATS webhook consumer receive error")
				continue
			}

			var dm deliverMsg
			if err := json.Unmarshal(msg.Data(), &dm); err != nil {
				logger.Error().Str("component", "webhook").Err(err).Msg("Failed to unmarshal NATS webhook delivery message")
				_ = msg.Term()
				continue
			}

			if err := d.deliverHTTPWithErr(dm.URL, dm.Secret, dm.Payload); err != nil {
				var permErr *permanentDeliveryError
				if errors.As(err, &permErr) {
					logger.Warn().Str("component", "webhook").Err(err).Str("webhook", dm.WebhookID).Str("url", dm.URL).Int("status", permErr.statusCode).Msg("NATS webhook delivery failed permanently (4xx), terminating message")
					_ = msg.Term()
				} else {
					logger.Warn().Str("component", "webhook").Err(err).Str("webhook", dm.WebhookID).Str("url", dm.URL).Msg("NATS webhook delivery failed, will retry")
					_ = msg.Nak()
				}
			} else {
				_ = msg.Ack()
			}
		}
	}()
}

func (d *Dispatcher) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) deliverHTTPWithErr(url, secret string, payload []byte) error {
	start := time.Now()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		req.Header.Set("X-Wzap-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	eventType := extractEventType(payload)
	if eventType != "" {
		req.Header.Set("X-Wzap-Event", eventType)
	}

	resp, err := d.httpClient.Do(req)
	duration := time.Since(start).Seconds()
	metrics.WebhooksDuration.Observe(duration)
	if err != nil {
		metrics.WebhooksFailed.Inc()
		return fmt.Errorf("http post: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn().Str("component", "webhook").Err(err).Msg("Failed to close webhook response body")
		}
	}()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("non-2xx status: %d", resp.StatusCode)
		metrics.WebhooksFailed.Inc()
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return &permanentDeliveryError{statusCode: resp.StatusCode, err: err}
		}
		return err
	}

	metrics.WebhooksDelivered.Inc()
	logger.Info().Str("component", "webhook").Str("url", url).Int("status", resp.StatusCode).Msg("Webhook delivered successfully")
	return nil
}

func extractEventType(payload []byte) string {
	var m struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(payload, &m); err != nil {
		return ""
	}
	return m.Event
}
