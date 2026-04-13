package service

import (
	"errors"
	"testing"

	"wzap/internal/model"
)

func TestRequireCapability(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		capability model.EngineCapability
		want       model.CapabilitySupport
		wantErr    bool
	}{
		{
			name:       "complete support",
			engine:     "whatsmeow",
			capability: model.CapabilityMessageText,
			want:       model.SupportComplete,
		},
		{
			name:       "partial support",
			engine:     "cloud_api",
			capability: model.CapabilityMessageLink,
			want:       model.SupportPartial,
		},
		{
			name:       "unavailable support",
			engine:     "cloud_api",
			capability: model.CapabilityMessagePoll,
			want:       model.SupportUnavailable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireCapability(tt.engine, tt.capability)
			if got != tt.want {
				t.Fatalf("expected support %q, got %q", tt.want, got)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				var capabilityErr *CapabilityError
				if !errors.As(err, &capabilityErr) {
					t.Fatalf("expected CapabilityError, got %T", err)
				}
				if capabilityErr.Support != tt.want {
					t.Fatalf("expected error support %q, got %q", tt.want, capabilityErr.Support)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
