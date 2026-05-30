package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "commit:") || !strings.Contains(got, "built:") {
		t.Errorf("unexpected output: %q", got)
	}
}
