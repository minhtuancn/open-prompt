package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSQL string

// DB wraps sql.DB
type DB struct {
	*sql.DB
}

// Open mở SQLite database tại ~/.open-prompt/open-prompt.db
func Open() (*DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	return openPath(filepath.Join(dir, config.DBFileName))
}

// OpenInMemory mở SQLite in-memory (dùng cho test)
func OpenInMemory() (*DB, error) {
	return openPath(":memory:")
}

func openPath(path string) (*DB, error) {
	dsn := path
	if path != ":memory:" {
		dsn = path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	} else {
		dsn = path + "?_pragma=foreign_keys(1)"
	}
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	raw.SetMaxOpenConns(1)
	return &DB{raw}, nil
}

// Migrate chạy migration SQL (idempotent — dùng CREATE TABLE IF NOT EXISTS)
func Migrate(db *DB) error {
	_, err := db.Exec(initSQL)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

// dataDir trả về ~/.open-prompt/
func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".open-prompt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}
