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

	// FindByID — tìm thấy
	found, err := repo.FindByID(s.ID)
	if err != nil {
		t.Fatalf("FindByID trả về lỗi: %v", err)
	}
	if found == nil {
		t.Fatal("FindByID phải trả về skill")
	}
	if found.Name != "Email skill" {
		t.Fatalf("FindByID tên sai: %q", found.Name)
	}
	if found.UserID != 1 {
		t.Fatalf("FindByID user_id sai: %d", found.UserID)
	}

	// FindByID — không tìm thấy
	missing, err := repo.FindByID(99999)
	if err != nil {
		t.Fatalf("FindByID (not found) trả về lỗi: %v", err)
	}
	if missing != nil {
		t.Fatal("FindByID ID không tồn tại phải trả về nil")
	}

	// List
	list, err := repo.List(1)
	if err != nil || len(list) != 1 {
		t.Fatalf("List thất bại: %v, len=%d", err, len(list))
	}

	// Update
	err = repo.Update(s.ID, repos.UpdateSkillInput{
		Name:       "Email skill v2",
		PromptText: "Send email: {{.input}}",
		Model:      "claude-3-5-sonnet",
		Provider:   "anthropic",
		Tags:       "email,productivity",
	})
	if err != nil {
		t.Fatalf("Update thất bại: %v", err)
	}

	// Xác nhận Update đã lưu đúng
	updated, err := repo.FindByID(s.ID)
	if err != nil || updated == nil {
		t.Fatalf("FindByID sau Update thất bại: %v", err)
	}
	if updated.Name != "Email skill v2" {
		t.Fatalf("Update tên sai: %q", updated.Name)
	}
	if updated.Model != "claude-3-5-sonnet" {
		t.Fatalf("Update model sai: %q", updated.Model)
	}
	if updated.Tags != "email,productivity" {
		t.Fatalf("Update tags sai: %q", updated.Tags)
	}

	// Delete
	if err := repo.Delete(s.ID); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}

	// Xác nhận đã xóa
	deletedSkill, _ := repo.FindByID(s.ID)
	if deletedSkill != nil {
		t.Fatal("Delete phải xóa skill")
	}
}
