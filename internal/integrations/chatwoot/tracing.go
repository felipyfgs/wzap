package chatwoot

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/nats-io/nats.go"
)

const tracerName = "wzap/chatwoot"

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

func spanAttrs(sessionID, msgType, direction string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", "whatsapp"),
		attribute.String("session.id", sessionID),
		attribute.String("message.type", msgType),
		attribute.String("message.direction", direction),
	}
}

// natsHeaderCarrier adapts nats.Header to the TextMapCarrier interface.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	return nats.Header(c).Get(key)
}
func (c natsHeaderCarrier) Set(key, val string) {
	nats.Header(c).Set(key, val)
}
func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectNATSHeaders injects the current trace context into NATS message headers.
func InjectNATSHeaders(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))
}
