package model_test

import (
	"testing"

	"wzap/internal/model"
)

func TestDefaultCapabilitiesSupport(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		capability model.EngineCapability
		want       model.CapabilitySupport
	}{
		{
			name:       "whatsmeow complete",
			engine:     "whatsmeow",
			capability: model.CapabilityMessageText,
			want:       model.SupportComplete,
		},
		{
			name:       "unknown engine unavailable",
			engine:     "unknown",
			capability: model.CapabilityMessageText,
			want:       model.SupportUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.DefaultCapabilities.Support(tt.engine, tt.capability)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDefaultCapabilitiesUnknownEngine(t *testing.T) {
	got := model.DefaultCapabilities.Support("unknown", model.CapabilityMessageText)
	if got != model.SupportUnavailable {
		t.Fatalf("expected %q, got %q", model.SupportUnavailable, got)
	}
}
