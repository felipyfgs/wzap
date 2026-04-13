package chatwoot

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_ConvHitMiss(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()

	_, _, ok := c.GetConv(ctx, "sess", "jid@s.whatsapp.net")
	if ok {
		t.Error("expected cache miss on empty cache")
	}

	c.SetConv(ctx, "sess", "jid@s.whatsapp.net", 42, 7)
	convID, contactID, ok := c.GetConv(ctx, "sess", "jid@s.whatsapp.net")
	if !ok {
		t.Error("expected cache hit after SetConv")
	}
	if convID != 42 {
		t.Errorf("expected convID=42, got %d", convID)
	}
	if contactID != 7 {
		t.Errorf("expected contactID=7, got %d", contactID)
	}
}

func TestMemoryCache_ConvInvalidation(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()

	c.SetConv(ctx, "sess", "jid@s.whatsapp.net", 42, 7)
	c.DeleteConv(ctx, "sess", "jid@s.whatsapp.net")

	_, _, ok := c.GetConv(ctx, "sess", "jid@s.whatsapp.net")
	if ok {
		t.Error("expected cache miss after DeleteConv")
	}
}

func TestMemoryCache_IdempotencyHitMiss(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()

	if c.GetIdempotent(ctx, "sess", "WAID:abc123") {
		t.Error("expected false on empty idempotency cache")
	}

	c.SetIdempotent(ctx, "sess", "WAID:abc123")

	if !c.GetIdempotent(ctx, "sess", "WAID:abc123") {
		t.Error("expected true after SetIdempotent")
	}
}

func TestMemoryCache_IsolationBetweenSessions(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()

	c.SetConv(ctx, "sess-A", "jid@s.whatsapp.net", 10, 1)

	_, _, ok := c.GetConv(ctx, "sess-B", "jid@s.whatsapp.net")
	if ok {
		t.Error("expected isolation: sess-B should not see sess-A cache")
	}
}

func TestNewCache_FallbackToMemoryOnEmptyURL(t *testing.T) {
	c := NewCache(context.Background(), "")
	if _, ok := c.(*MemoryCache); !ok {
		t.Error("expected MemoryCache when redisURL is empty")
	}
}

func TestNewCache_FallbackToMemoryOnInvalidURL(t *testing.T) {
	c := NewCache(context.Background(), "not-a-valid-redis-url://!!!")
	if _, ok := c.(*MemoryCache); !ok {
		t.Errorf("expected MemoryCache on invalid redis URL, got %T", c)
	}
}

func TestMemoryCache_ConvTTLExpired(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()
	key := "sess:jid@s.whatsapp.net"

	c.mu.Lock()
	c.convs[key] = convCacheEntry{
		ConvID:    99,
		ContactID: 88,
		expiresAt: time.Now().Add(-time.Minute),
	}
	c.mu.Unlock()

	_, _, ok := c.GetConv(ctx, "sess", "jid@s.whatsapp.net")
	if ok {
		t.Fatal("expected expired conversation cache entry to miss")
	}

	c.mu.RLock()
	_, exists := c.convs[key]
	c.mu.RUnlock()
	if exists {
		t.Fatal("expected expired conversation cache entry to be removed")
	}
}

func TestMemoryCache_IdempotentTTLExpired(t *testing.T) {
	c := newMemoryCache(context.Background())
	ctx := context.Background()
	key := "sess:WAID:abc123"

	c.mu.Lock()
	c.idempotents[key] = idempotentEntry{expiresAt: time.Now().Add(-time.Minute)}
	c.mu.Unlock()

	if c.GetIdempotent(ctx, "sess", "WAID:abc123") {
		t.Fatal("expected expired idempotent cache entry to return false")
	}

	c.mu.RLock()
	_, exists := c.idempotents[key]
	c.mu.RUnlock()
	if exists {
		t.Fatal("expected expired idempotent cache entry to be removed")
	}
}
