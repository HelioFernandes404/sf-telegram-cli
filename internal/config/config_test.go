package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/config"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
telegram:
  api_id: 12345
  api_hash_env: MY_HASH_ENV

defaults:
  channel: "@alerts"
  format: "json"
  limit: 25
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Telegram.APIID != 12345 {
		t.Errorf("api_id: got %d want 12345", cfg.Telegram.APIID)
	}
	if cfg.Telegram.APIHashEnv != "MY_HASH_ENV" {
		t.Errorf("api_hash_env: got %q", cfg.Telegram.APIHashEnv)
	}
	if cfg.Defaults.Channel != "@alerts" {
		t.Errorf("channel: got %q", cfg.Defaults.Channel)
	}
	if cfg.Defaults.Limit != 25 {
		t.Errorf("limit: got %d", cfg.Defaults.Limit)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := config.LoadFrom("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestAPIHashFromEnv(t *testing.T) {
	t.Setenv("TEST_API_HASH", "secrethash")
	dir := t.TempDir()
	yaml := "telegram:\n  api_id: 1\n  api_hash_env: TEST_API_HASH\n"
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := cfg.APIHash()
	if hash != "secrethash" {
		t.Errorf("APIHash(): got %q want secrethash", hash)
	}
}
