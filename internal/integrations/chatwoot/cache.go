package chatwoot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"wzap/internal/logger"
)

const (
	convCacheTTL    = time.Hour
	idempotentTTL   = 24 * time.Hour
	circuitCacheTTL = 5 * time.Minute
	cleanupInterval = 5 * time.Minute
)

type convCacheEntry struct {
	ConvID    int       `json:"convID"`
	ContactID int       `json:"contactID"`
	expiresAt time.Time `json:"-"`
}

type idempotentEntry struct {
	expiresAt time.Time
}

type Cache interface {
	GetConv(ctx context.Context, sessionID, chatJID string) (convID, contactID int, ok bool)
	SetConv(ctx context.Context, sessionID, chatJID string, convID, contactID int)
	DeleteConv(ctx context.Context, sessionID, chatJID string)
	GetIdempotent(ctx context.Context, sessionID, sourceID string) bool
	SetIdempotent(ctx context.Context, sessionID, sourceID string)
}

type RedisCache struct {
	client *redis.Client
}

func (r *RedisCache) GetConv(ctx context.Context, sessionID, chatJID string) (int, int, bool) {
	key := fmt.Sprintf("cw:conv:%s:%s", sessionID, chatJID)
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return 0, 0, false
	}
	var entry convCacheEntry
	if err := json.Unmarshal(val, &entry); err != nil {
		return 0, 0, false
	}
	return entry.ConvID, entry.ContactID, true
}

func (r *RedisCache) SetConv(ctx context.Context, sessionID, chatJID string, convID, contactID int) {
	key := fmt.Sprintf("cw:conv:%s:%s", sessionID, chatJID)
	val, _ := json.Marshal(convCacheEntry{ConvID: convID, ContactID: contactID})
	_ = r.client.Set(ctx, key, val, convCacheTTL).Err()
}

func (r *RedisCache) DeleteConv(ctx context.Context, sessionID, chatJID string) {
	key := fmt.Sprintf("cw:conv:%s:%s", sessionID, chatJID)
	_ = r.client.Del(ctx, key).Err()
}

func (r *RedisCache) GetIdempotent(ctx context.Context, sessionID, sourceID string) bool {
	key := fmt.Sprintf("cw:idempotent:%s:%s", sessionID, sourceID)
	val, err := r.client.Get(ctx, key).Result()
	return err == nil && val == "1"
}

func (r *RedisCache) SetIdempotent(ctx context.Context, sessionID, sourceID string) {
	key := fmt.Sprintf("cw:idempotent:%s:%s", sessionID, sourceID)
	_ = r.client.Set(ctx, key, "1", idempotentTTL).Err()
}

type MemoryCache struct {
	mu          sync.RWMutex
	convs       map[string]convCacheEntry
	idempotents map[string]idempotentEntry
}

func newMemoryCache(ctx context.Context) *MemoryCache {
	if ctx == nil {
		ctx = context.Background()
	}

	cache := &MemoryCache{
		convs:       make(map[string]convCacheEntry),
		idempotents: make(map[string]idempotentEntry),
	}

	cache.startCleanup(ctx)
	return cache
}

func (m *MemoryCache) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.cleanupExpired(time.Now())
			}
		}
	}()
}

func (m *MemoryCache) cleanupExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.convs {
		if !entry.expiresAt.IsZero() && !entry.expiresAt.After(now) {
			delete(m.convs, key)
		}
	}
	for key, entry := range m.idempotents {
		if !entry.expiresAt.IsZero() && !entry.expiresAt.After(now) {
			delete(m.idempotents, key)
		}
	}
}

func (m *MemoryCache) GetConv(_ context.Context, sessionID, chatJID string) (int, int, bool) {
	key := sessionID + ":" + chatJID

	m.mu.RLock()
	entry, ok := m.convs[key]
	m.mu.RUnlock()
	if !ok {
		return 0, 0, false
	}
	if !entry.expiresAt.IsZero() && !entry.expiresAt.After(time.Now()) {
		m.mu.Lock()
		delete(m.convs, key)
		m.mu.Unlock()
		return 0, 0, false
	}

	return entry.ConvID, entry.ContactID, ok
}

func (m *MemoryCache) SetConv(_ context.Context, sessionID, chatJID string, convID, contactID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.convs[sessionID+":"+chatJID] = convCacheEntry{
		ConvID:    convID,
		ContactID: contactID,
		expiresAt: time.Now().Add(convCacheTTL),
	}
}

func (m *MemoryCache) DeleteConv(_ context.Context, sessionID, chatJID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.convs, sessionID+":"+chatJID)
}

func (m *MemoryCache) GetIdempotent(_ context.Context, sessionID, sourceID string) bool {
	key := sessionID + ":" + sourceID

	m.mu.RLock()
	entry, ok := m.idempotents[key]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	if !entry.expiresAt.IsZero() && !entry.expiresAt.After(time.Now()) {
		m.mu.Lock()
		delete(m.idempotents, key)
		m.mu.Unlock()
		return false
	}

	return true
}

func (m *MemoryCache) SetIdempotent(_ context.Context, sessionID, sourceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idempotents[sessionID+":"+sourceID] = idempotentEntry{
		expiresAt: time.Now().Add(idempotentTTL),
	}
}

func NewCache(ctx context.Context, redisURL string) Cache {
	if redisURL == "" {
		return newMemoryCache(ctx)
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("redisURL", redisURL).Msg("invalid Redis URL, using in-memory cache")
		return newMemoryCache(ctx)
	}
	client := redis.NewClient(opt)
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Redis unavailable, using in-memory cache")
		_ = client.Close()
		return newMemoryCache(ctx)
	}
	logger.Info().Str("component", "chatwoot").Str("addr", opt.Addr).Msg("Redis cache connected")
	return &RedisCache{client: client}
}
