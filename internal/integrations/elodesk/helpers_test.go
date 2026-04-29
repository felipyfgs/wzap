package elodesk

import "testing"

func TestExtractForwardingFromMap_Nil(t *testing.T) {
	isFwd, score := extractForwardingFromMap(nil)
	if isFwd || score != 0 {
		t.Fatalf("expected (false,0) for nil msg, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwardingFromMap_PlainConversation(t *testing.T) {
	msg := map[string]any{"conversation": "oi"}
	isFwd, score := extractForwardingFromMap(msg)
	if isFwd || score != 0 {
		t.Fatalf("expected (false,0), got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwardingFromMap_ExtendedText(t *testing.T) {
	// Score chega como float64 quando vindo de json.Unmarshal de JSON number.
	msg := map[string]any{
		"extendedTextMessage": map[string]any{
			"text": "oi",
			"contextInfo": map[string]any{
				"isForwarded":     true,
				"forwardingScore": float64(3),
			},
		},
	}
	isFwd, score := extractForwardingFromMap(msg)
	if !isFwd || score != 3 {
		t.Fatalf("expected (true,3), got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwardingFromMap_ImageMessage(t *testing.T) {
	msg := map[string]any{
		"imageMessage": map[string]any{
			"caption": "foto",
			"contextInfo": map[string]any{
				"isForwarded":     true,
				"forwardingScore": float64(7),
			},
		},
	}
	isFwd, score := extractForwardingFromMap(msg)
	if !isFwd || score != 7 {
		t.Fatalf("expected (true,7), got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwardingFromMap_EphemeralWrapper(t *testing.T) {
	msg := map[string]any{
		"ephemeralMessage": map[string]any{
			"message": map[string]any{
				"extendedTextMessage": map[string]any{
					"contextInfo": map[string]any{
						"isForwarded":     true,
						"forwardingScore": float64(2),
					},
				},
			},
		},
	}
	isFwd, score := extractForwardingFromMap(msg)
	if !isFwd || score != 2 {
		t.Fatalf("expected (true,2) inside ephemeral, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwardingFromMap_ScoreOnlyWithoutFlag(t *testing.T) {
	// Defensivo: WhatsApp sempre envia isForwarded junto, mas se score vier
	// solto sem o flag, devolvemos (false, score>0). Caller decide.
	msg := map[string]any{
		"imageMessage": map[string]any{
			"contextInfo": map[string]any{
				"forwardingScore": float64(4),
			},
		},
	}
	isFwd, score := extractForwardingFromMap(msg)
	if isFwd || score != 4 {
		t.Fatalf("expected (false,4), got (%v,%d)", isFwd, score)
	}
}
