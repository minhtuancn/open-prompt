package repos_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func TestSkillRepo_CRUD(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewSkillRepo(database)

	// Create
	s, err := repo.Create(repos.CreateSkillInput{
		UserID:     1,
		Name:       "Email skill",
		PromptText: "Write email: {{.input}}",
	})
	if err != nil {
		t.Fatalf("Create thất bại: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("ID phải > 0")
	}

	// List
	list, err := repo.List(1)
	if err != nil || len(list) != 1 {
		t.Fatalf("List thất bại: %v, len=%d", err, len(list))
	}

	// Delete
	if err := repo.Delete(s.ID); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}
}
