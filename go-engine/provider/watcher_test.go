package provider_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestWatcherTriggerOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")
	os.WriteFile(configFile, []byte(`{"key":"v1"}`), 0644)

	changed := make(chan string, 1)
	w, err := provider.NewWatcher(func(path string) {
		select {
		case changed <- path:
		default:
		}
	})
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	if err := w.Watch(configFile); err != nil {
		t.Fatalf("Watch() error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	os.WriteFile(configFile, []byte(`{"key":"v2"}`), 0644)

	select {
	case path := <-changed:
		if path != configFile {
			t.Errorf("path = %q, want %q", path, configFile)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout: không nhận được event thay đổi file")
	}
}

func TestWatcherClose(t *testing.T) {
	w, err := provider.NewWatcher(func(path string) {})
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	w.Close()
	w.Close() // phải không panic
}
