package service

import (
	"context"
	"fmt"

	cloudWA "wzap/internal/provider/whatsapp"
)

type SessionConfigReader struct {
	resolver *SessionRuntimeResolver
}

func NewSessionConfigReader(resolver *SessionRuntimeResolver) *SessionConfigReader {
	return &SessionConfigReader{resolver: resolver}
}

func (r *SessionConfigReader) ReadConfig(ctx context.Context, sessionID string) (*cloudWA.Config, error) {
	if runtime, ok := sessionRuntimeFromContext(ctx, sessionID); ok {
		return runtime.CloudConfig()
	}
	if r == nil || r.resolver == nil {
		return nil, fmt.Errorf("session runtime resolver is nil")
	}

	runtime, err := r.resolver.Resolve(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session %s: %w", sessionID, err)
	}

	return runtime.CloudConfig()
}
