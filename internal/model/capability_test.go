package model_test

import (
	"testing"

	"wzap/internal/model"
)

func TestDefaultEngineCapabilityContractSupport(t *testing.T) {
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
			want:       model.CapabilitySupportComplete,
		},
		{
			name:       "cloud partial",
			engine:     "cloud_api",
			capability: model.CapabilitySessionProfile,
			want:       model.CapabilitySupportPartial,
		},
		{
			name:       "cloud unavailable",
			engine:     "cloud_api",
			capability: model.CapabilityMessagePoll,
			want:       model.CapabilitySupportUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.DefaultEngineCapabilityContract.Support(tt.engine, tt.capability)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDefaultEngineCapabilityContractUnknownEngine(t *testing.T) {
	got := model.DefaultEngineCapabilityContract.Support("unknown", model.CapabilityMessageText)
	if got != model.CapabilitySupportUnavailable {
		t.Fatalf("expected %q, got %q", model.CapabilitySupportUnavailable, got)
	}
}
