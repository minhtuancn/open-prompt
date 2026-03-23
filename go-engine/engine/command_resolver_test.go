package engine_test

import (
	"os"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

func setupResolverDB(t *testing.T) (*db.DB, *repos.PromptRepo) {
	t.Helper()
	f, _ := os.CreateTemp("", "resolver-*.db")
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	database, err := db.OpenPath(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// Tạo user để không vi phạm FOREIGN KEY
	userRepo := repos.NewUserRepo(database)
	_, err = userRepo.Create("testuser", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	return database, repos.NewPromptRepo(database)
}

func TestCommandResolver_Resolve(t *testing.T) {
	_, promptRepo := setupResolverDB(t)

	// Seed slash command
	_, err := promptRepo.Create(repos.CreatePromptInput{
		UserID:    1,
		Title:     "Email writer",
		Content:   "Viết email về {{.input}}",
		IsSlash:   true,
		SlashName: "email",
	})
	if err != nil {
		t.Fatalf("seed prompt: %v", err)
	}

	pb := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(promptRepo, pb)

	// Resolve "/email hello world"
	result, err := resolver.Resolve(1, "email", "hello world", nil)
	if err != nil {
		t.Fatalf("Resolve thất bại: %v", err)
	}
	if result.RenderedPrompt != "Viết email về hello world" {
		t.Errorf("RenderedPrompt = %q", result.RenderedPrompt)
	}
	if result.NeedsVars {
		t.Error("NeedsVars phải false khi chỉ có {{.input}}")
	}
}

func TestCommandResolver_NeedsVars(t *testing.T) {
	_, promptRepo := setupResolverDB(t)

	_, err := promptRepo.Create(repos.CreatePromptInput{
		UserID:    1,
		Title:     "Translator",
		Content:   "Dịch {{.input}} sang {{.lang}}",
		IsSlash:   true,
		SlashName: "translate",
	})
	if err != nil {
		t.Fatalf("seed prompt: %v", err)
	}

	pb := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(promptRepo, pb)

	// Resolve không có vars — phải báo NeedsVars
	result, err := resolver.Resolve(1, "translate", "Hello", nil)
	if err != nil {
		t.Fatalf("Resolve thất bại: %v", err)
	}
	if !result.NeedsVars {
		t.Error("NeedsVars phải true khi template có biến ngoài input")
	}
	if len(result.RequiredVars) == 0 {
		t.Error("RequiredVars phải có ít nhất 1 phần tử")
	}
}

func TestCommandResolver_NotFound(t *testing.T) {
	_, promptRepo := setupResolverDB(t)
	pb := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(promptRepo, pb)

	_, err := resolver.Resolve(1, "nonexistent", "input", nil)
	if err == nil {
		t.Error("Resolve slash_name không tồn tại phải trả về error")
	}
}
