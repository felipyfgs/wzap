package service

import (
	"context"
	"fmt"

	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
)

type SessionConfigReader struct {
	repo *repo.SessionRepository
}

func NewSessionConfigReader(r *repo.SessionRepository) *SessionConfigReader {
	return &SessionConfigReader{repo: r}
}

func (r *SessionConfigReader) ReadConfig(ctx context.Context, sessionID string) (*cloudWA.Config, error) {
	session, err := r.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session %s: %w", sessionID, err)
	}

	if session.Engine != "cloud_api" {
		return nil, fmt.Errorf("session %s engine is %s, not cloud_api", sessionID, session.Engine)
	}

	return &cloudWA.Config{
		AccessToken:        session.AccessToken,
		PhoneNumberID:      session.PhoneNumberID,
		BusinessAccountID:  session.BusinessAccountID,
		AppSecret:          session.AppSecret,
		WebhookVerifyToken: session.WebhookVerifyToken,
	}, nil
}
