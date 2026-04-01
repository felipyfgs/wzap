package wa

import (
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/repo"
	"wzap/internal/webhook"
)

type Manager struct {
	clients map[string]*whatsmeow.Client
	mu      sync.RWMutex

	sessionRepo *repo.SessionRepository
	container   *sqlstore.Container
	nats        *broker.Nats
	dispatcher  *webhook.Dispatcher
	cfg         *config.Config
	waLog       waLog.Logger
}
