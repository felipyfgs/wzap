package chatwoot

import (
	"testing"
)

func TestChatwootConfigNewFields(t *testing.T) {
	cfg := &Config{
		ImportOnConnect: true,
		ImportPeriod:    "7d",
		TextTimeout:     10,
		MediaTimeout:    60,
		LargeTimeout:    300,
		RedisURL:        "redis://localhost:6379",
	}

	if !cfg.ImportOnConnect {
		t.Error("expected ImportOnConnect = true")
	}
	if cfg.ImportPeriod != "7d" {
		t.Errorf("expected ImportPeriod = 7d, got %s", cfg.ImportPeriod)
	}
	if cfg.TextTimeout != 10 {
		t.Errorf("expected TextTimeout = 10, got %d", cfg.TextTimeout)
	}
	if cfg.MediaTimeout != 60 {
		t.Errorf("expected MediaTimeout = 60, got %d", cfg.MediaTimeout)
	}
	if cfg.LargeTimeout != 300 {
		t.Errorf("expected LargeTimeout = 300, got %d", cfg.LargeTimeout)
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

func TestMaskURL_WithPassword(t *testing.T) {
	raw := "redis://:mysecretpass@redis.host:6379/0"
	got := maskURL(raw)
	want := "redis://***@redis.host:6379/0"
	if got != want {
		t.Errorf("maskURL(%q) = %q, want %q", raw, got, want)
	}
}

func TestMaskURL_WithoutPassword(t *testing.T) {
	raw := "redis://redis.host:6379/0"
	got := maskURL(raw)
	if got != raw {
		t.Errorf("maskURL(%q) = %q, want %q", raw, got, raw)
	}
}

func TestMaskURL_Empty(t *testing.T) {
	if got := maskURL(""); got != "" {
		t.Errorf("maskURL(empty) = %q, want empty", got)
	}
}
