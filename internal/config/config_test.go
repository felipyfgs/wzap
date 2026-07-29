package config_test

import (
	"testing"

	"wzap/internal/config"
)

func TestLoadWAVersion(t *testing.T) {
	t.Setenv("WA_VERSION", "2.3000.1044062641")

	cfg := config.Load()
	if cfg.WAVersion != "2.3000.1044062641" {
		t.Fatalf("expected WA_VERSION to be loaded, got %q", cfg.WAVersion)
	}
}
