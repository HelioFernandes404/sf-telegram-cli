package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/version"
)

func TestVersionCommand(t *testing.T) {
	version.Version = "v0.1.0"
	version.Commit = "abc1234"
	version.Date = "2026-06-01T00:00:00Z"
	rootCmd.SetVersionTemplate(versionJSON())

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	var got map[string]string
	if err := json.NewDecoder(buf).Decode(&got); err != nil {
		t.Fatalf("decode version JSON: %v", err)
	}
	if got["version"] != "v0.1.0" || got["commit"] != "abc1234" || got["date"] != "2026-06-01T00:00:00Z" {
		t.Errorf("unexpected version JSON: %#v", got)
	}
}
