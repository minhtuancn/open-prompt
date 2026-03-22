package db_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func TestOpenAndMigrate(t *testing.T) {
	database, err := db.OpenInMemory()
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	tables := []string{
		"users", "projects", "prompts", "skills", "settings",
		"history", "provider_tokens", "model_priority", "usage_daily",
	}
	for _, table := range tables {
		var count int
		err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("table %q not found after migration", table)
		}
	}
}

func TestUserRepo(t *testing.T) {
	database, _ := openTestDB(t)
	repo := repos.NewUserRepo(database)

	// Create user
	user, err := repo.Create("alice", "hashed_password")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected alice, got %s", user.Username)
	}

	// Find by username
	found, err := repo.FindByUsername("alice")
	if err != nil || found == nil {
		t.Fatalf("find by username: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("wrong ID: expected %d, got %d", user.ID, found.ID)
	}

	// Count
	count, _ := repo.Count()
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Not found returns nil
	notFound, _ := repo.FindByUsername("nobody")
	if notFound != nil {
		t.Error("expected nil for unknown user")
	}
}

func TestSettingsRepo(t *testing.T) {
	database, _ := openTestDB(t)
	userRepo := repos.NewUserRepo(database)
	settingsRepo := repos.NewSettingsRepo(database)

	user, _ := userRepo.Create("bob", "hash")

	// Set and get
	if err := settingsRepo.Set(user.ID, "theme", "dark"); err != nil {
		t.Fatalf("set: %v", err)
	}
	val, err := settingsRepo.Get(user.ID, "theme")
	if err != nil || val != "dark" {
		t.Errorf("get: expected 'dark', got %q (err: %v)", val, err)
	}

	// Upsert
	_ = settingsRepo.Set(user.ID, "theme", "light")
	val, _ = settingsRepo.Get(user.ID, "theme")
	if val != "light" {
		t.Errorf("upsert: expected 'light', got %q", val)
	}

	// Missing key returns ""
	empty, _ := settingsRepo.Get(user.ID, "missing_key")
	if empty != "" {
		t.Errorf("missing key should return empty, got %q", empty)
	}
}

func openTestDB(t *testing.T) (*db.DB, error) {
	t.Helper()
	database, err := db.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database, nil
}
