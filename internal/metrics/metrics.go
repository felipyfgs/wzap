package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SessionsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wzap_sessions_total",
		Help: "Total number of sessions created",
	})

	SessionsConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wzap_sessions_connected",
		Help: "Number of currently connected sessions",
	})

	MessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wzap_messages_sent_total",
		Help: "Total number of messages sent",
	})

	MessagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wzap_messages_received_total",
		Help: "Total number of messages received",
	})

	WebhooksDelivered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wzap_webhooks_delivered_total",
		Help: "Total number of webhooks delivered successfully",
	})

	WebhooksFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wzap_webhooks_failed_total",
		Help: "Total number of webhook deliveries that failed",
	})

	WebhooksDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "wzap_webhooks_duration_seconds",
		Help:    "Duration of webhook delivery attempts",
		Buckets: prometheus.DefBuckets,
	})
)
