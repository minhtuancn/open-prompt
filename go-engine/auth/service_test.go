package auth_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestRegisterAndLogin(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	// Register
	user, err := svc.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", user.Username)
	}

	// Login
	token, err := svc.Login("alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Validate token
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, claims.UserID)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	_, _ = svc.Register("bob", "correct-password")
	_, err := svc.Login("bob", "wrong")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestFirstRun(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	isFirst, err := svc.IsFirstRun()
	if err != nil {
		t.Fatal(err)
	}
	if !isFirst {
		t.Error("expected first run before any users created")
	}

	_, _ = svc.Register("alice", "password123")

	isFirst, _ = svc.IsFirstRun()
	if isFirst {
		t.Error("expected not first run after user created")
	}
}
