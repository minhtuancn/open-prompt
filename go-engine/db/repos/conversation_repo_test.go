package repos_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func TestAddMessage_RejectsWrongUser(t *testing.T) {
	database := newTestDB(t)
	convRepo := repos.NewConversationRepo(database)
	userRepo := repos.NewUserRepo(database)

	// user 1 is seeded by newTestDB (id=1)
	// Create user 2
	u2, err := userRepo.Create("user2", "hashedpw")
	if err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	// user 1 creates a conversation
	convID, err := convRepo.Create(1, "User1 conversation")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	// user 2 tries to add message to user 1's conversation → should fail
	err = convRepo.AddMessage(convID, u2.ID, "user", "hello", "", "", 0)
	if err == nil {
		t.Fatal("expected error when user 2 adds message to user 1's conversation, got nil")
	}
}

func TestGetMessages_RejectsWrongUser(t *testing.T) {
	database := newTestDB(t)
	convRepo := repos.NewConversationRepo(database)
	userRepo := repos.NewUserRepo(database)

	// Create user 2
	u2, err := userRepo.Create("user2", "hashedpw")
	if err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	// user 1 creates a conversation and adds a message
	convID, err := convRepo.Create(1, "User1 conversation")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}
	if err := convRepo.AddMessage(convID, 1, "user", "secret message", "", "", 0); err != nil {
		t.Fatalf("add message failed: %v", err)
	}

	// user 2 tries to read user 1's conversation → should fail
	_, err = convRepo.GetMessages(convID, u2.ID)
	if err == nil {
		t.Fatal("expected error when user 2 reads user 1's conversation, got nil")
	}
}

func TestGetMessages_AllowsCorrectUser(t *testing.T) {
	database := newTestDB(t)
	convRepo := repos.NewConversationRepo(database)

	// user 1 creates a conversation and adds a message
	convID, err := convRepo.Create(1, "My conversation")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}
	if err := convRepo.AddMessage(convID, 1, "user", "hello", "", "", 0); err != nil {
		t.Fatalf("add message failed: %v", err)
	}

	// user 1 reads own conversation → should succeed
	msgs, err := convRepo.GetMessages(convID, 1)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("expected content 'hello', got '%s'", msgs[0].Content)
	}
}
