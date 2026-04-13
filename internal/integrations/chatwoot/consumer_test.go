package chatwoot

import (
	"context"
	"encoding/json"
	"testing"

	"wzap/internal/model"
)

func TestPublishInbound_MarshalEnvelope(t *testing.T) {
	env := inboundEnvelope{
		SessionID: "sess-1",
		Event:     model.EventMessage,
		Payload:   json.RawMessage(`{"key":"value"}`),
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal inbound envelope: %v", err)
	}

	var decoded inboundEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal inbound envelope: %v", err)
	}

	if decoded.SessionID != "sess-1" {
		t.Errorf("expected SessionID=sess-1, got %s", decoded.SessionID)
	}
	if decoded.Event != model.EventMessage {
		t.Errorf("expected Event=Message, got %s", decoded.Event)
	}
}

func TestPublishOutbound_MarshalEnvelope(t *testing.T) {
	env := outboundEnvelope{
		SessionID: "sess-2",
		Payload:   json.RawMessage(`{"event_type":"message_created"}`),
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal outbound envelope: %v", err)
	}

	var decoded outboundEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal outbound envelope: %v", err)
	}

	if decoded.SessionID != "sess-2" {
		t.Errorf("expected SessionID=sess-2, got %s", decoded.SessionID)
	}
}

func TestConsumer_BackoffSchedule(t *testing.T) {
	if len(backoffSchedule) == 0 {
		t.Error("expected non-empty backoff schedule")
	}
	for i := 1; i < len(backoffSchedule); i++ {
		if backoffSchedule[i] < backoffSchedule[i-1] {
			t.Errorf("expected backoff to be non-decreasing, but [%d]=%v < [%d]=%v",
				i, backoffSchedule[i], i-1, backoffSchedule[i-1])
		}
	}
}

func TestConsumer_FallbackSync_NoNATS(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	// No JS set — should fallback to sync processing
	if svc.js != nil {
		t.Skip("this test requires js=nil")
	}

	payload := buildMsgPayload(t, "sync-msg")
	svc.OnEvent(context.Background(), "sess", model.EventMessage, payload)

	if len(client.messages) == 0 {
		t.Error("expected message created via sync fallback")
	}
}

func TestConsumer_DeadLetterSubject(t *testing.T) {
	inboundSubject := "cw.deadletter.inbound"
	outboundSubject := "cw.deadletter.outbound"

	if inboundSubject != deadLetterBase+".inbound" {
		t.Errorf("unexpected dead letter subject: %s", inboundSubject)
	}
	if outboundSubject != deadLetterBase+".outbound" {
		t.Errorf("unexpected dead letter subject: %s", outboundSubject)
	}
}

func TestConsumer_ProcessInboundEvent_Unknown(t *testing.T) {
	svc := newTestService(&mockClient{})
	// Unknown event type should be handled gracefully
	err := svc.processInboundEvent(context.Background(), "sess", "UnknownEvent", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("expected no error for unknown event, got: %v", err)
	}
}
