package auth_test

import (
	"errors"
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
	svc, err := auth.NewService(userRepo, "test-jwt-secret-16chars")
	if err != nil {
		t.Fatal(err)
	}

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
	svc, err := auth.NewService(userRepo, "test-jwt-secret-16chars")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = svc.Register("bob", "correct-password")
	_, err = svc.Login("bob", "wrong")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc, err := auth.NewService(userRepo, "test-jwt-secret-16chars")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Login("nobody", "password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterPasswordValidation(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc, _ := auth.NewService(userRepo, "test-jwt-secret-16chars")

	// Password quá ngắn
	_, err := svc.Register("user1", "short")
	if !errors.Is(err, auth.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}

	// Password quá dài (> 72 bytes — bcrypt truncation risk)
	longPwd := string(make([]byte, 73))
	for i := range longPwd {
		longPwd = "a"
		_ = i
	}
	longPwd = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 73 'a'
	_, err = svc.Register("user2", longPwd)
	if !errors.Is(err, auth.ErrPasswordTooLong) {
		t.Errorf("expected ErrPasswordTooLong for 73-char password, got %v", err)
	}
}

func TestRegisterUsernameValidation(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc, _ := auth.NewService(userRepo, "test-jwt-secret-16chars")

	// Username quá dài (> 64 chars)
	longName := "usernamethatisveryveryveryveryveryveryveryveryveryverylongindeed12345"
	_, err := svc.Register(longName, "password123")
	if !errors.Is(err, auth.ErrUsernameTooLong) {
		t.Errorf("expected ErrUsernameTooLong, got %v", err)
	}
}

func TestFirstRun(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc, err := auth.NewService(userRepo, "test-jwt-secret-16chars")
	if err != nil {
		t.Fatal(err)
	}

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
