package auth_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
)

func TestFileSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)

	ctx := context.Background()

	// Load from non-existent file returns nil, no error
	data, err := s.LoadSession(ctx)
	if err != nil {
		t.Fatalf("LoadSession on missing file: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for missing file, got %q", data)
	}

	// Store and reload
	want := []byte(`{"test":"data"}`)
	if err := s.StoreSession(ctx, want); err != nil {
		t.Fatalf("StoreSession: %v", err)
	}

	got, err := s.LoadSession(ctx)
	if err != nil {
		t.Fatalf("LoadSession after store: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFileSessionPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission test not meaningful as root")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)
	_ = s.StoreSession(context.Background(), []byte("x"))

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o want 0600", perm)
	}
}

func TestSessionExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)

	if s.Exists() {
		t.Error("Exists() should be false before any store")
	}
	_ = s.StoreSession(context.Background(), []byte("x"))
	if !s.Exists() {
		t.Error("Exists() should be true after store")
	}
}

func TestSessionDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := auth.NewFileSession(path)
	_ = s.StoreSession(context.Background(), []byte("x"))

	if err := s.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Exists() {
		t.Error("Exists() should be false after Delete")
	}
}

func TestLogoutRemovesSession(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	os.WriteFile(sessionPath, []byte("data"), 0600)

	svc := auth.NewService(auth.ServiceConfig{SessionPath: sessionPath})
	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("session file should be gone after Logout")
	}
}

func TestStatusNoSession(t *testing.T) {
	dir := t.TempDir()
	svc := auth.NewService(auth.ServiceConfig{
		SessionPath: filepath.Join(dir, "session.json"),
	})
	status := svc.LocalStatus()
	if status.Authenticated {
		t.Error("should not be authenticated with no session")
	}
	if status.Session.Exists {
		t.Error("session should not exist")
	}
}
