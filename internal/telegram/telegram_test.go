package telegram_test

import (
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/telegram"
)

func TestStripAtPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"@canal-alertas", "canal-alertas"},
		{"canal-alertas", "canal-alertas"},
		{"@test", "test"},
	}
	for _, c := range cases {
		got := telegram.StripAt(c.in)
		if got != c.want {
			t.Errorf("StripAt(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
