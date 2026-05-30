package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type TelegramConfig struct {
	APIID      int    `yaml:"api_id"`
	APIHashEnv string `yaml:"api_hash_env"`
}

type Defaults struct {
	Channel string `yaml:"channel"`
	Format  string `yaml:"format"`
	Limit   int    `yaml:"limit"`
}

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Defaults Defaults       `yaml:"defaults"`
}

// APIHash resolves api_hash: env var first, then the local secrets file.
func (c *Config) APIHash() string {
	if c.Telegram.APIHashEnv != "" {
		if v := os.Getenv(c.Telegram.APIHashEnv); v != "" {
			return v
		}
	}
	// Fall back to secrets file (mode 0600).
	data, err := os.ReadFile(APIHashPath())
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

// ConfigPath returns the platform-appropriate config file path.
func ConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "tg-alerts", "config.yaml")
}

// SessionPath returns the platform-appropriate session file path.
func SessionPath() string {
	return filepath.Join(xdg.DataHome, "tg-alerts", "session.json")
}

// APIHashPath returns the path to the local api_hash secrets file.
func APIHashPath() string {
	return filepath.Join(xdg.DataHome, "tg-alerts", "api_hash")
}

// SaveAPIHash writes the api_hash to the local secrets file with mode 0600.
func SaveAPIHash(hash string) error {
	path := APIHashPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(hash)), 0600)
}

// LoadFrom reads and parses the config file at the given path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// Load reads from the default platform config path.
// Returns a zero-value Config (not an error) if the file does not yet exist.
func Load() (*Config, error) {
	path := ConfigPath()
	cfg, err := LoadFrom(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	return cfg, err
}
