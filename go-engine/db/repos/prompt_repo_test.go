package repos_test

import (
	"os"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	// Dùng temp file thay vì :memory: để tránh concurrent test issues
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("tạo temp file thất bại: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	database, err := db.OpenPath(f.Name())
	if err != nil {
		t.Fatalf("mở db thất bại: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate thất bại: %v", err)
	}

	// Seed user_id=1 để thỏa foreign key constraint
	userRepo := repos.NewUserRepo(database)
	if _, err := userRepo.Create("testuser", "hashedpassword"); err != nil {
		t.Fatalf("tạo test user thất bại: %v", err)
	}

	t.Cleanup(func() { database.Close() })
	return database
}

func TestPromptRepo_CRUD(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewPromptRepo(database)

	// Create
	p, err := repo.Create(repos.CreatePromptInput{
		UserID:    1,
		Title:     "Email writer",
		Content:   "Write an email about {{.input}}",
		Category:  "productivity",
		IsSlash:   true,
		SlashName: "email",
	})
	if err != nil {
		t.Fatalf("Create thất bại: %v", err)
	}
	if p.ID == 0 || p.SlashName != "email" {
		t.Fatalf("Create trả về sai: %+v", p)
	}

	// FindByID
	found, err := repo.FindByID(p.ID)
	if err != nil || found == nil || found.Title != "Email writer" {
		t.Fatalf("FindByID thất bại: %v, %+v", err, found)
	}

	// FindBySlashName
	bySlash, err := repo.FindBySlashName(1, "email")
	if err != nil || bySlash == nil {
		t.Fatalf("FindBySlashName thất bại: %v", err)
	}

	// List
	list, err := repo.List(1, "")
	if err != nil || len(list) != 1 {
		t.Fatalf("List thất bại: %v, len=%d", err, len(list))
	}

	// Update
	err = repo.Update(p.ID, repos.UpdatePromptInput{Title: "Updated email"})
	if err != nil {
		t.Fatalf("Update thất bại: %v", err)
	}

	// Delete
	if err := repo.Delete(p.ID); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}
	deleted, _ := repo.FindByID(p.ID)
	if deleted != nil {
		t.Fatal("Delete phải xóa prompt")
	}
}
