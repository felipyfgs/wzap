package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"wzap/internal/broker"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	natsDeliverSubject  = "wzap.webhook.deliver"
	httpTimeout         = 10 * time.Second
	maxDeliverAttempts  = 5
	maxHTTPRetries      = 3
	httpRetryBaseDelay  = 2 * time.Second
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

type Dispatcher struct {
	webhookRepo      *repo.WebhookRepository
	nats             *broker.NATS
	httpClient       *http.Client
	globalWebhookURL string
	ws               WSBroadcaster
}

func New(webhookRepo *repo.WebhookRepository, nats *broker.NATS, globalWebhookURL string) *Dispatcher {
	return &Dispatcher{
		webhookRepo:      webhookRepo,
		nats:             nats,
		httpClient:       &http.Client{Timeout: httpTimeout},
		globalWebhookURL: globalWebhookURL,
	}
}

func (d *Dispatcher) SetWSBroadcaster(ws WSBroadcaster) {
	d.ws = ws
}

// Dispatch looks up active webhooks for the session/event and delivers the payload.
func (d *Dispatcher) Dispatch(sessionID string, eventType model.EventType, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	webhooks, err := d.webhookRepo.FindActiveBySessionAndEvent(ctx, sessionID, string(eventType))
	if err != nil {
		logger.Error().Err(err).Str("session", sessionID).Str("event", string(eventType)).Msg("Failed to fetch webhooks for dispatch")
		return
	}

	for _, wh := range webhooks {
		wh := wh
		if wh.NATSEnabled && d.nats != nil {
			go d.publishToNATS(wh, payload)
		} else {
			go d.deliverHTTPWithRetry(wh.URL, wh.Secret, payload)
		}
	}

	if d.globalWebhookURL != "" {
		go d.deliverHTTPWithRetry(d.globalWebhookURL, "", payload)
	}

	if d.ws != nil {
		d.ws.Broadcast(sessionID, payload)
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
		logger.Error().Err(err).Str("webhook", wh.ID).Msg("Failed to marshal NATS webhook delivery message")
		return
	}
	subject := natsDeliverSubject + "." + wh.ID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.nats.Publish(ctx, subject, data); err != nil {
		logger.Error().Err(err).Str("webhook", wh.ID).Msg("Failed to publish webhook delivery to NATS — falling back to direct dispatch")
		go d.deliverHTTPWithRetry(wh.URL, wh.Secret, payload)
	}
}

func (d *Dispatcher) deliverHTTPWithRetry(url, secret string, payload []byte) {
	for attempt := 0; attempt <= maxHTTPRetries; attempt++ {
		if err := d.deliverHTTPWithErr(url, secret, payload); err != nil {
			if attempt < maxHTTPRetries {
				delay := httpRetryBaseDelay * time.Duration(1<<uint(attempt))
				logger.Warn().Err(err).Str("url", url).Int("attempt", attempt+1).Dur("retryIn", delay).Msg("Webhook HTTP delivery failed, retrying")
				time.Sleep(delay)
				continue
			}
			logger.Error().Err(err).Str("url", url).Msg("Webhook HTTP delivery failed after all retries")
		}
		return
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
		logger.Warn().Err(err).Msg("Failed to create NATS webhook consumer — NATS-queued webhooks will fall back to direct dispatch")
		return
	}

	msgCtx, err := cons.Messages()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to subscribe to NATS webhook consumer")
		return
	}

	logger.Info().Msg("NATS webhook consumer started")

	go func() {
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
				logger.Warn().Err(err).Msg("NATS webhook consumer receive error")
				continue
			}

			var dm deliverMsg
			if err := json.Unmarshal(msg.Data(), &dm); err != nil {
				logger.Error().Err(err).Msg("Failed to unmarshal NATS webhook delivery message")
				_ = msg.Term()
				continue
			}

			if err := d.deliverHTTPWithErr(dm.URL, dm.Secret, dm.Payload); err != nil {
				logger.Warn().Err(err).Str("webhook", dm.WebhookID).Str("url", dm.URL).Msg("NATS webhook delivery failed, will retry")
				_ = msg.Nak()
			} else {
				_ = msg.Ack()
			}
		}
	}()
}

func (d *Dispatcher) deliverHTTPWithErr(url, secret string, payload []byte) error {
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
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn().Err(err).Msg("Failed to close webhook response body")
		}
	}()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx status: %d", resp.StatusCode)
	}
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
