package chatwoot

import (
	"context"

	"wzap/internal/wa"
)

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
