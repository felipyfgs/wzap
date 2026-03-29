package wa

import (
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/repo"
)

type Manager struct {
	clients map[string]*whatsmeow.Client
	mu      sync.RWMutex

	sessionRepo *repo.SessionRepository
	container   *sqlstore.Container
	nats        *broker.Nats
	cfg         *config.Config
	waLog       waLog.Logger
}
