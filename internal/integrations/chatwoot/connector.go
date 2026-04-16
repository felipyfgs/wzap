package chatwoot

import (
	"context"
	"strings"

	"wzap/internal/repo"
	"wzap/internal/wa"
)

type sessionPhoneGetter struct {
	sessRepo *repo.SessionRepository
}

func NewSessionPhoneGetter(sessRepo *repo.SessionRepository) SessionPhoneGetter {
	return &sessionPhoneGetter{sessRepo: sessRepo}
}

func (g *sessionPhoneGetter) GetSessionPhone(ctx context.Context, sessionID string) string {
	sess, err := g.sessRepo.FindByID(ctx, sessionID)
	if err != nil || sess == nil || sess.JID == "" {
		return ""
	}
	return strings.Split(strings.Split(sess.JID, "@")[0], ":")[0]
}

type managerConnector struct {
	engine *wa.Manager
}

func NewSessionConnector(engine *wa.Manager) SessionConnector {
	return &managerConnector{engine: engine}
}

func (c *managerConnector) Connect(ctx context.Context, sessionID string) error {
	_, _, err := c.engine.Connect(ctx, sessionID)
	return err
}

func (c *managerConnector) Disconnect(ctx context.Context, sessionID string) error {
	return c.engine.Disconnect(ctx, sessionID)
}

func (c *managerConnector) Logout(ctx context.Context, sessionID string) error {
	return c.engine.Logout(ctx, sessionID)
}

func (c *managerConnector) IsConnected(sessionID string) bool {
	client, err := c.engine.GetClient(sessionID)
	if err != nil {
		return false
	}
	return client.IsConnected()
}
