package auth

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// FileSession implements telegram.SessionStorage (session.Storage) and provides
// lifecycle helpers for the auth service.
type FileSession struct {
	path string
	mu   sync.Mutex
}

// NewFileSession creates a FileSession for the given path.
func NewFileSession(path string) *FileSession {
	return &FileSession{path: path}
}

// Path returns the session file path.
func (f *FileSession) Path() string { return f.path }

// LoadSession satisfies session.Storage. Returns nil, nil when file is absent.
func (f *FileSession) LoadSession(_ context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// StoreSession satisfies session.Storage. Writes with mode 0600.
func (f *FileSession) StoreSession(_ context.Context, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(f.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0600)
}

// Exists reports whether the session file is present.
func (f *FileSession) Exists() bool {
	_, err := os.Stat(f.path)
	return err == nil
}

// Delete removes the session file.
func (f *FileSession) Delete() error {
	err := os.Remove(f.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
