package chatwoot

import (
	"testing"
)

func TestChatwootConfigNewFields(t *testing.T) {
	cfg := &Config{
		SessionID:           "sess-1",
		URL:                 "https://cw.example.com",
		AccountID:           1,
		Token:               "token",
		InboxID:             1,
		InboxName:           "wzap",
		ImportOnConnect:     true,
		ImportPeriod:        "7d",
		TimeoutTextSeconds:  10,
		TimeoutMediaSeconds: 60,
		TimeoutLargeSeconds: 300,
		RedisURL:            "redis://localhost:6379",
	}

	if !cfg.ImportOnConnect {
		t.Error("expected ImportOnConnect = true")
	}
	if cfg.ImportPeriod != "7d" {
		t.Errorf("expected ImportPeriod = 7d, got %s", cfg.ImportPeriod)
	}
	if cfg.TimeoutTextSeconds != 10 {
		t.Errorf("expected TimeoutTextSeconds = 10, got %d", cfg.TimeoutTextSeconds)
	}
	if cfg.TimeoutMediaSeconds != 60 {
		t.Errorf("expected TimeoutMediaSeconds = 60, got %d", cfg.TimeoutMediaSeconds)
	}
	if cfg.TimeoutLargeSeconds != 300 {
		t.Errorf("expected TimeoutLargeSeconds = 300, got %d", cfg.TimeoutLargeSeconds)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("expected RedisURL = redis://localhost:6379, got %s", cfg.RedisURL)
	}
}

func TestImportPeriodToDays(t *testing.T) {
	cases := []struct {
		period     string
		customDays int
		expected   int
	}{
		{"24h", 0, 1},
		{"7d", 0, 7},
		{"30d", 0, 30},
		{"custom", 15, 15},
		{"custom", 0, 0},
		{"invalid", 0, 0},
	}
	for _, tc := range cases {
		got := importPeriodToDays(tc.period, tc.customDays)
		if got != tc.expected {
			t.Errorf("importPeriodToDays(%q, %d) = %d, want %d", tc.period, tc.customDays, got, tc.expected)
		}
	}
}

func TestMaskRedisURL_WithPassword(t *testing.T) {
	raw := "redis://:mysecretpass@redis.host:6379/0"
	got := maskRedisURL(raw)
	want := "redis://***@redis.host:6379/0"
	if got != want {
		t.Errorf("maskRedisURL(%q) = %q, want %q", raw, got, want)
	}
}

func TestMaskRedisURL_WithoutPassword(t *testing.T) {
	raw := "redis://redis.host:6379/0"
	got := maskRedisURL(raw)
	if got != raw {
		t.Errorf("maskRedisURL(%q) = %q, want %q", raw, got, raw)
	}
}

func TestMaskRedisURL_Empty(t *testing.T) {
	if got := maskRedisURL(""); got != "" {
		t.Errorf("maskRedisURL(empty) = %q, want empty", got)
	}
}
