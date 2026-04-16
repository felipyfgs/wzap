package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildEventEnvelope_Structure(t *testing.T) {
	data := map[string]any{"key": "value"}
	bytes, err := BuildEventEnvelope("sess-1", "MySession", EventMessage, data)
	if err != nil {
		t.Fatalf("BuildEventEnvelope returned error: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(bytes, &raw); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if _, ok := raw["event"]; !ok {
		t.Error("missing 'event' field")
	}
	if _, ok := raw["eventId"]; !ok {
		t.Error("missing 'eventId' field")
	}
	if _, ok := raw["timestamp"]; !ok {
		t.Error("missing 'timestamp' field")
	}
	if _, ok := raw["data"]; !ok {
		t.Error("missing 'data' field")
	}
	if _, ok := raw["session"]; !ok {
		t.Error("missing 'session' field")
	}

	if raw["event"] != "Message" {
		t.Errorf("expected event=Message, got %v", raw["event"])
	}

	session, _ := raw["session"].(map[string]any)
	if session["id"] != "sess-1" || session["name"] != "MySession" {
		t.Errorf("unexpected session: %v", session)
	}

	eventID, _ := raw["eventId"].(string)
	if eventID == "" {
		t.Error("eventId should not be empty")
	}

	ts, _ := raw["timestamp"].(string)
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp not RFC3339: %v", ts)
	}
}

func TestBuildEventEnvelopeFromRaw_Structure(t *testing.T) {
	rawData := json.RawMessage(`{"from":"5511999999999","body":"hello"}`)
	bytes, err := BuildEventEnvelopeFromRaw("sess-2", "S2", EventReceipt, rawData)
	if err != nil {
		t.Fatalf("BuildEventEnvelopeFromRaw returned error: %v", err)
	}

	var env EventEnvelope
	if err := json.Unmarshal(bytes, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if env.Event != "Receipt" {
		t.Errorf("expected event=Receipt, got %s", env.Event)
	}
	if env.Session.ID != "sess-2" {
		t.Errorf("expected session.id=sess-2, got %s", env.Session.ID)
	}
	if string(env.Data) != `{"from":"5511999999999","body":"hello"}` {
		t.Errorf("unexpected data: %s", string(env.Data))
	}
}

func TestBuildEventEnvelope_DistinctEventIDs(t *testing.T) {
	data := map[string]any{"x": 1}
	b1, _ := BuildEventEnvelope("s", "n", EventConnected, data)
	b2, _ := BuildEventEnvelope("s", "n", EventConnected, data)

	var e1, e2 EventEnvelope
	_ = json.Unmarshal(b1, &e1)
	_ = json.Unmarshal(b2, &e2)

	if e1.EventID == e2.EventID {
		t.Error("each envelope should have a unique eventId")
	}
}

func TestParseEventEnvelope_RoundTrip(t *testing.T) {
	original := map[string]any{
		"Chat":   "5511@s.whatsapp.net",
		"Sender": "5511@s.whatsapp.net",
		"ID":     "msg-1",
	}
	bytes, err := BuildEventEnvelope("s1", "S1", EventMessage, original)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	env, err := ParseEventEnvelope(bytes)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if env.Event != "Message" {
		t.Errorf("expected Message, got %s", env.Event)
	}
	if env.Session.ID != "s1" {
		t.Errorf("expected s1, got %s", env.Session.ID)
	}

	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if data["Chat"] != "5511@s.whatsapp.net" {
		t.Errorf("unexpected Chat in data: %v", data["Chat"])
	}
}

func TestEnvelopeParity_BothEnginesProduceSameTopLevel(t *testing.T) {
	whatsmeowData := map[string]any{
		"Info": map[string]any{
			"Chat":     "5511@s.whatsapp.net",
			"IsFromMe": false,
			"ID":       "wa-msg-1",
		},
		"Message": map[string]any{"conversation": "hello from whatsmeow"},
	}
	waBytes, _ := BuildEventEnvelope("sess", "MySess", EventMessage, whatsmeowData)

	cloudData := map[string]any{
		"from": "5511",
		"id":   "cloud-msg-1",
		"type": "text",
		"body": "hello from cloud",
	}
	cloudBytes, _ := BuildEventEnvelope("sess", "", EventMessage, cloudData)

	topLevelFields := []string{"event", "eventId", "session", "timestamp", "data"}

	for label, raw := range map[string][]byte{"whatsmeow": waBytes, "custom": cloudBytes} {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal %s envelope: %v", label, err)
		}
		for _, f := range topLevelFields {
			if _, ok := m[f]; !ok {
				t.Errorf("%s envelope missing top-level field %s", label, f)
			}
		}
		if m["event"] != "Message" {
			t.Errorf("%s envelope event=%v, want Message", label, m["event"])
		}
	}
}

func TestParseEventEnvelope_Invalid(t *testing.T) {
	_, err := ParseEventEnvelope([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseEventEnvelope_Empty(t *testing.T) {
	_, err := ParseEventEnvelope([]byte("{}"))
	if err != nil {
		t.Fatalf("empty object should parse: %v", err)
	}
}

func TestBuildEventEnvelope_LargeData(t *testing.T) {
	data := map[string]any{
		"body": strings.Repeat("x", 1024*1024),
	}
	bytes, err := BuildEventEnvelope("s", "n", EventMessage, data)
	if err != nil {
		t.Fatalf("large data: %v", err)
	}
	if len(bytes) < 1024*1024 {
		t.Error("expected large payload")
	}

	env, err := ParseEventEnvelope(bytes)
	if err != nil {
		t.Fatalf("parse large: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(env.Data, &parsed)
	if len(parsed["body"].(string)) != 1024*1024 {
		t.Error("body length mismatch")
	}
}
