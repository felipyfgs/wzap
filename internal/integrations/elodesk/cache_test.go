package elodesk

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_ConvRoundtrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newMemoryCache(ctx)

	c.SetConv(ctx, "sess1", "11988887777@s.whatsapp.net", 42, 100)
	convID, contactID, ok := c.GetConv(ctx, "sess1", "11988887777@s.whatsapp.net")
	if !ok {
		t.Fatal("expected hit")
	}
	if convID != 42 || contactID != 100 {
		t.Errorf("got %d/%d, want 42/100", convID, contactID)
	}

	c.DeleteConv(ctx, "sess1", "11988887777@s.whatsapp.net")
	if _, _, ok := c.GetConv(ctx, "sess1", "11988887777@s.whatsapp.net"); ok {
		t.Error("expected miss after delete")
	}
}

func TestMemoryCache_IdempotentRoundtrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newMemoryCache(ctx)

	if c.GetIdempotent(ctx, "sess", "WAID:x") {
		t.Error("expected miss before set")
	}
	c.SetIdempotent(ctx, "sess", "WAID:x")
	if !c.GetIdempotent(ctx, "sess", "WAID:x") {
		t.Error("expected hit after set")
	}
}

func TestMemoryCache_ExpiredEntry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newMemoryCache(ctx)

	c.convs["sess:chat"] = convCacheEntry{
		ConvID:    99,
		ContactID: 100,
		expiresAt: time.Now().Add(-time.Second),
	}
	if _, _, ok := c.GetConv(ctx, "sess", "chat"); ok {
		t.Error("expected expired entry to miss")
	}
}
