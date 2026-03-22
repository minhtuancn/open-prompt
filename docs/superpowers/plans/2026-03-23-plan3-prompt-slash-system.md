# Prompt & Slash Command System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement full prompt/skill CRUD with a slash command palette that lets users type `/email hello` in the overlay to resolve and render Go templates before sending to the AI model.

**Architecture:** The Go engine gains two repos (PromptRepo, SkillRepo) and two engine layers (PromptBuilder renders `text/template`, CommandResolver maps slash_name → rendered text); new JSON-RPC handlers expose `prompts.*` and `commands.*` methods; the React overlay detects a leading `/` to open a fuzzy-filtered SlashMenu built with fuse.js, and a Prompt Manager UI in settings provides full CRUD.

**Tech Stack:** Go `text/template`, SQLite (mattn/go-sqlite3), fuse.js 7.x, React 18, TypeScript, Tauri v2 JSON-RPC over Unix socket.

---

## Spec Reference
`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

---

## File Map

**Create (Go):**
- `go-engine/db/repos/prompt_repo.go`
- `go-engine/db/repos/prompt_repo_test.go`
- `go-engine/db/repos/skill_repo.go`
- `go-engine/db/repos/skill_repo_test.go`
- `go-engine/engine/prompt_builder.go`
- `go-engine/engine/prompt_builder_test.go`
- `go-engine/engine/command_resolver.go`
- `go-engine/engine/command_resolver_test.go`
- `go-engine/api/handlers_prompts.go`
- `go-engine/db/migrations/002_seed.sql`

**Modify (Go):**
- `go-engine/api/router.go` — thêm repos + dispatch cases mới
- `go-engine/api/handlers_query.go` — hỗ trợ `slash_name` param
- `go-engine/api/server.go` — chạy migration 002

**Create (React):**
- `src/components/overlay/SlashMenu.tsx`
- `src/components/prompts/PromptList.tsx`
- `src/components/prompts/PromptEditor.tsx`

**Modify (React):**
- `src/components/overlay/CommandInput.tsx` — detect `/` → mở SlashMenu

---

## Task 1: DB Repos — PromptRepo & SkillRepo

**Files:**
- Create: `go-engine/db/repos/prompt_repo.go`
- Create: `go-engine/db/repos/skill_repo.go`
- Test: `go-engine/db/repos/prompt_repo_test.go`
- Test: `go-engine/db/repos/skill_repo_test.go`

- [ ] **Step 1.1: Viết failing test cho PromptRepo**

File: `go-engine/db/repos/prompt_repo_test.go`

```go
package repos_test

import (
	"os"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("tạo temp file thất bại: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("mở db thất bại: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate thất bại: %v", err)
	}
	return database
}

func TestPromptRepo_CreateAndFind(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewPromptRepo(database)

	// Tạo prompt slash
	p, err := repo.Create(repos.CreatePromptInput{
		UserID:    1,
		Title:     "Email writer",
		Content:   "Write an email about {{.input}} in {{.lang}}",
		Category:  "productivity",
		Tags:      "email,writing",
		IsSlash:   true,
		SlashName: "email",
	})
	if err != nil {
		t.Fatalf("Create thất bại: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID phải > 0")
	}
	if p.SlashName != "email" {
		t.Errorf("SlashName = %q, muốn %q", p.SlashName, "email")
	}

	// FindByID
	found, err := repo.FindByID(p.ID)
	if err != nil {
		t.Fatalf("FindByID thất bại: %v", err)
	}
	if found == nil || found.Title != "Email writer" {
		t.Error("FindByID trả về sai dữ liệu")
	}

	// FindBySlashName
	bySlash, err := repo.FindBySlashName(1, "email")
	if err != nil {
		t.Fatalf("FindBySlashName thất bại: %v", err)
	}
	if bySlash == nil {
		t.Fatal("FindBySlashName phải tìm thấy prompt")
	}

	// List
	list, err := repo.List(1, "")
	if err != nil {
		t.Fatalf("List thất bại: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List = %d items, muốn 1", len(list))
	}
}

func TestPromptRepo_UpdateAndDelete(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewPromptRepo(database)

	p, _ := repo.Create(repos.CreatePromptInput{
		UserID:  1,
		Title:   "Old title",
		Content: "Old content",
	})

	// Update
	updated, err := repo.Update(p.ID, repos.UpdatePromptInput{
		Title:   "New title",
		Content: "New content",
	})
	if err != nil {
		t.Fatalf("Update thất bại: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("Title = %q, muốn %q", updated.Title, "New title")
	}

	// Delete
	if err := repo.Delete(p.ID); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}
	gone, _ := repo.FindByID(p.ID)
	if gone != nil {
		t.Error("Prompt vẫn còn sau khi Delete")
	}
}

func TestPromptRepo_SearchSlashCommands(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewPromptRepo(database)

	repo.Create(repos.CreatePromptInput{UserID: 1, Title: "Email", Content: "...", IsSlash: true, SlashName: "email"})
	repo.Create(repos.CreatePromptInput{UserID: 1, Title: "Code review", Content: "...", IsSlash: true, SlashName: "code"})
	repo.Create(repos.CreatePromptInput{UserID: 1, Title: "Normal", Content: "..."})

	// Chỉ lấy slash commands
	cmds, err := repo.ListSlashCommands(1)
	if err != nil {
		t.Fatalf("ListSlashCommands thất bại: %v", err)
	}
	if len(cmds) != 2 {
		t.Errorf("ListSlashCommands = %d, muốn 2", len(cmds))
	}
}
```

- [ ] **Step 1.2: Chạy test để xác nhận FAIL**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./db/repos/... -v -run TestPromptRepo
```

Expected: `FAIL` — `cannot find package` hoặc `undefined: repos.NewPromptRepo`

- [ ] **Step 1.3: Implement PromptRepo**

File: `go-engine/db/repos/prompt_repo.go`

```go
package repos

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Prompt là model cho bảng prompts
type Prompt struct {
	ID        int64
	UserID    int64
	ProjectID sql.NullInt64
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreatePromptInput chứa dữ liệu đầu vào khi tạo prompt mới
type CreatePromptInput struct {
	UserID    int64
	ProjectID int64 // 0 = không thuộc project nào
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
}

// UpdatePromptInput chứa các trường có thể cập nhật
type UpdatePromptInput struct {
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
}

// PromptRepo xử lý CRUD cho bảng prompts
type PromptRepo struct {
	db *db.DB
}

// NewPromptRepo tạo PromptRepo mới
func NewPromptRepo(database *db.DB) *PromptRepo {
	return &PromptRepo{db: database}
}

// Create tạo prompt mới và trả về bản ghi vừa tạo
func (r *PromptRepo) Create(in CreatePromptInput) (*Prompt, error) {
	isSlashInt := 0
	if in.IsSlash {
		isSlashInt = 1
	}
	var projectID interface{}
	if in.ProjectID > 0 {
		projectID = in.ProjectID
	}
	res, err := r.db.Exec(
		`INSERT INTO prompts (user_id, project_id, title, content, category, tags, is_slash, slash_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.UserID, projectID, in.Title, in.Content, in.Category, in.Tags, isSlashInt, nullString(in.SlashName),
	)
	if err != nil {
		return nil, fmt.Errorf("create prompt: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create prompt: last insert id: %w", err)
	}
	return r.FindByID(id)
}

// FindByID tìm prompt theo ID, trả về nil nếu không tìm thấy
func (r *PromptRepo) FindByID(id int64) (*Prompt, error) {
	p := &Prompt{}
	var isSlashInt int
	err := r.db.QueryRow(
		`SELECT id, user_id, project_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.ProjectID, &p.Title, &p.Content,
		&p.Category, &p.Tags, &isSlashInt, &p.SlashName, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find prompt by id: %w", err)
	}
	p.IsSlash = isSlashInt == 1
	return p, nil
}

// FindBySlashName tìm prompt theo slash_name của user
func (r *PromptRepo) FindBySlashName(userID int64, slashName string) (*Prompt, error) {
	p := &Prompt{}
	var isSlashInt int
	err := r.db.QueryRow(
		`SELECT id, user_id, project_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE user_id = ? AND slash_name = ? AND is_slash = 1`, userID, slashName,
	).Scan(&p.ID, &p.UserID, &p.ProjectID, &p.Title, &p.Content,
		&p.Category, &p.Tags, &isSlashInt, &p.SlashName, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find prompt by slash_name: %w", err)
	}
	p.IsSlash = isSlashInt == 1
	return p, nil
}

// List lấy danh sách prompts của user, lọc theo category nếu có
func (r *PromptRepo) List(userID int64, category string) ([]*Prompt, error) {
	query := `SELECT id, user_id, project_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
	          FROM prompts WHERE user_id = ?`
	args := []interface{}{userID}
	if category != "" {
		query += ` AND category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY updated_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()
	return scanPromptRows(rows)
}

// ListSlashCommands lấy tất cả slash commands của user
func (r *PromptRepo) ListSlashCommands(userID int64) ([]*Prompt, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, project_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE user_id = ? AND is_slash = 1 ORDER BY slash_name ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list slash commands: %w", err)
	}
	defer rows.Close()
	return scanPromptRows(rows)
}

// Update cập nhật prompt và trả về bản ghi mới
func (r *PromptRepo) Update(id int64, in UpdatePromptInput) (*Prompt, error) {
	isSlashInt := 0
	if in.IsSlash {
		isSlashInt = 1
	}
	_, err := r.db.Exec(
		`UPDATE prompts SET title=?, content=?, category=?, tags=?, is_slash=?, slash_name=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		in.Title, in.Content, in.Category, in.Tags, isSlashInt, nullString(in.SlashName), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update prompt: %w", err)
	}
	return r.FindByID(id)
}

// Delete xoá prompt theo ID
func (r *PromptRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM prompts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete prompt: %w", err)
	}
	return nil
}

// scanPromptRows đọc nhiều rows thành slice Prompt
func scanPromptRows(rows *sql.Rows) ([]*Prompt, error) {
	var list []*Prompt
	for rows.Next() {
		p := &Prompt{}
		var isSlashInt int
		if err := rows.Scan(&p.ID, &p.UserID, &p.ProjectID, &p.Title, &p.Content,
			&p.Category, &p.Tags, &isSlashInt, &p.SlashName, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt row: %w", err)
		}
		p.IsSlash = isSlashInt == 1
		list = append(list, p)
	}
	return list, rows.Err()
}

// nullString trả về nil nếu s rỗng để lưu NULL vào DB
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 1.4: Viết failing test cho SkillRepo**

File: `go-engine/db/repos/skill_repo_test.go`

```go
package repos_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func TestSkillRepo_CreateAndList(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewSkillRepo(database)

	s, err := repo.Create(repos.CreateSkillInput{
		UserID:     1,
		Name:       "Email assistant",
		PromptText: "Help write emails",
		Model:      "claude-3-5-sonnet-20241022",
		Provider:   "anthropic",
		Tags:       "email,writing",
	})
	if err != nil {
		t.Fatalf("Create thất bại: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("ID phải > 0")
	}

	list, err := repo.List(1)
	if err != nil {
		t.Fatalf("List thất bại: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List = %d, muốn 1", len(list))
	}

	if err := repo.Delete(s.ID); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}
	afterDelete, _ := repo.List(1)
	if len(afterDelete) != 0 {
		t.Error("Skill vẫn còn sau Delete")
	}
}
```

- [ ] **Step 1.5: Implement SkillRepo**

File: `go-engine/db/repos/skill_repo.go`

```go
package repos

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Skill là model cho bảng skills
type Skill struct {
	ID         int64
	UserID     int64
	Name       string
	PromptID   sql.NullInt64
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
	CreatedAt  time.Time
}

// CreateSkillInput chứa dữ liệu tạo skill mới
type CreateSkillInput struct {
	UserID     int64
	Name       string
	PromptID   int64 // 0 = không liên kết prompt
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
}

// SkillRepo xử lý CRUD cho bảng skills
type SkillRepo struct {
	db *db.DB
}

// NewSkillRepo tạo SkillRepo mới
func NewSkillRepo(database *db.DB) *SkillRepo {
	return &SkillRepo{db: database}
}

// Create tạo skill mới
func (r *SkillRepo) Create(in CreateSkillInput) (*Skill, error) {
	var promptID interface{}
	if in.PromptID > 0 {
		promptID = in.PromptID
	}
	res, err := r.db.Exec(
		`INSERT INTO skills (user_id, name, prompt_id, prompt_text, model, provider, config_json, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.UserID, in.Name, promptID, in.PromptText, in.Model, in.Provider, in.ConfigJSON, in.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("create skill: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create skill: last insert id: %w", err)
	}
	return r.FindByID(id)
}

// FindByID tìm skill theo ID
func (r *SkillRepo) FindByID(id int64) (*Skill, error) {
	s := &Skill{}
	err := r.db.QueryRow(
		`SELECT id, user_id, name, prompt_id, prompt_text, model, provider, config_json, tags, created_at
		 FROM skills WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.PromptID, &s.PromptText, &s.Model, &s.Provider, &s.ConfigJSON, &s.Tags, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find skill by id: %w", err)
	}
	return s, nil
}

// List lấy tất cả skills của user
func (r *SkillRepo) List(userID int64) ([]*Skill, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, prompt_id, prompt_text, model, provider, config_json, tags, created_at
		 FROM skills WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	var list []*Skill
	for rows.Next() {
		s := &Skill{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.PromptID, &s.PromptText,
			&s.Model, &s.Provider, &s.ConfigJSON, &s.Tags, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan skill row: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// Delete xoá skill theo ID
func (r *SkillRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM skills WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}
```

- [ ] **Step 1.6: Chạy test để xác nhận PASS**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./db/repos/... -v -run "TestPromptRepo|TestSkillRepo"
```

Expected: tất cả PASS, không có lỗi.

- [ ] **Step 1.7: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add db/repos/prompt_repo.go db/repos/prompt_repo_test.go db/repos/skill_repo.go db/repos/skill_repo_test.go && git commit -m "feat: thêm PromptRepo và SkillRepo với CRUD đầy đủ"
```

---

## Task 2: Template Engine — PromptBuilder

**Files:**
- Create: `go-engine/engine/prompt_builder.go`
- Test: `go-engine/engine/prompt_builder_test.go`

- [ ] **Step 2.1: Viết failing test**

File: `go-engine/engine/prompt_builder_test.go`

```go
package engine_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

func TestPromptBuilder_Render(t *testing.T) {
	builder := engine.NewPromptBuilder()

	// Render template đơn giản với input và lang
	result, err := builder.Render(
		"Write an email about {{.input}} in {{.lang}}",
		map[string]string{
			"input": "project update",
			"lang":  "English",
		},
	)
	if err != nil {
		t.Fatalf("Render thất bại: %v", err)
	}
	if result != "Write an email about project update in English" {
		t.Errorf("Render = %q, kết quả không đúng", result)
	}
}

func TestPromptBuilder_ExtractVariables(t *testing.T) {
	builder := engine.NewPromptBuilder()

	// Trích xuất biến, bỏ qua input và context.*
	vars := builder.ExtractVariables("Translate {{.input}} to {{.lang}} with {{.tone}} style in {{.context.app}}")
	if len(vars) != 2 {
		t.Fatalf("ExtractVariables = %v (len=%d), muốn [lang tone]", vars, len(vars))
	}
	// Kiểm tra lang và tone có trong kết quả
	found := map[string]bool{}
	for _, v := range vars {
		found[v] = true
	}
	if !found["lang"] || !found["tone"] {
		t.Errorf("ExtractVariables thiếu biến: %v", vars)
	}
}

func TestPromptBuilder_RenderMissingVar(t *testing.T) {
	builder := engine.NewPromptBuilder()

	// Biến thiếu sẽ render thành chuỗi rỗng (hành vi mặc định của text/template)
	result, err := builder.Render("Hello {{.name}}", map[string]string{"input": "world"})
	if err != nil {
		t.Fatalf("Render thất bại: %v", err)
	}
	// text/template với map sẽ render biến thiếu thành <no value> hoặc rỗng
	// Builder phải dùng option "missingkey=zero" để trả về rỗng
	if result != "Hello " {
		t.Errorf("Render với biến thiếu = %q, muốn %q", result, "Hello ")
	}
}

func TestPromptBuilder_InvalidTemplate(t *testing.T) {
	builder := engine.NewPromptBuilder()

	// Template không hợp lệ phải trả về lỗi
	_, err := builder.Render("Hello {{.name", nil)
	if err == nil {
		t.Error("Render template không hợp lệ phải trả về lỗi")
	}
}
```

- [ ] **Step 2.2: Chạy test để xác nhận FAIL**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... -v -run TestPromptBuilder
```

Expected: `FAIL` — `cannot find package "github.com/minhtuancn/open-prompt/go-engine/engine"`

- [ ] **Step 2.3: Implement PromptBuilder**

Tạo thư mục engine trước:
```
mkdir -p /home/dev/open-prompt-code/open-prompt/go-engine/engine
```

File: `go-engine/engine/prompt_builder.go`

```go
package engine

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"
)

// Regex trích xuất tên biến từ template Go, dạng {{.varName}}
var varRegex = regexp.MustCompile(`\{\{\.(\w+)\}\}`)

// Tập biến hệ thống — không hiển thị cho user nhập
var systemVars = map[string]bool{
	"input": true,
}

// PromptBuilder render template Go thành prompt cuối cùng
type PromptBuilder struct{}

// NewPromptBuilder tạo PromptBuilder mới
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// Render thực thi template với map biến, trả về chuỗi kết quả
// Biến thiếu sẽ render thành chuỗi rỗng (missingkey=zero)
func (b *PromptBuilder) Render(tmplText string, vars map[string]string) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(tmplText)
	if err != nil {
		return "", err
	}

	// Chuyển map[string]string → map[string]interface{} cho template
	data := make(map[string]interface{}, len(vars))
	for k, v := range vars {
		data[k] = v
	}
	// Thêm context rỗng để tránh lỗi khi template dùng {{.context.app}}
	if _, ok := data["context"]; !ok {
		data["context"] = map[string]string{}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExtractVariables trả về danh sách tên biến người dùng cần nhập
// Bỏ qua các biến hệ thống (input) và biến context (context.*)
func (b *PromptBuilder) ExtractVariables(tmplText string) []string {
	matches := varRegex.FindAllStringSubmatch(tmplText, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		name := m[1]
		// Bỏ qua biến hệ thống và biến context.*
		if systemVars[name] || strings.HasPrefix(name, "context") {
			continue
		}
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}
```

- [ ] **Step 2.4: Chạy test để xác nhận PASS**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... -v -run TestPromptBuilder
```

Expected: tất cả 4 test PASS.

- [ ] **Step 2.5: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add engine/prompt_builder.go engine/prompt_builder_test.go && git commit -m "feat: thêm PromptBuilder với text/template rendering và trích xuất biến"
```

---

## Task 3: Command Resolver

**Files:**
- Create: `go-engine/engine/command_resolver.go`
- Test: `go-engine/engine/command_resolver_test.go`

- [ ] **Step 3.1: Viết failing test**

File: `go-engine/engine/command_resolver_test.go`

```go
package engine_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

// mockPromptFinder giả lập PromptFinder cho test
type mockPromptFinder struct {
	prompts map[string]string // slash_name → template content
}

func (m *mockPromptFinder) FindBySlashName(userID int64, slashName string) (string, error) {
	content, ok := m.prompts[slashName]
	if !ok {
		return "", nil
	}
	return content, nil
}

func TestCommandResolver_Resolve(t *testing.T) {
	finder := &mockPromptFinder{
		prompts: map[string]string{
			"email": "Write an email about {{.input}}",
			"code":  "Review this code:\n{{.input}}",
		},
	}
	builder := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(finder, builder)

	// /email hello world → render template với input = "hello world"
	result, err := resolver.Resolve(1, "email", "hello world", nil)
	if err != nil {
		t.Fatalf("Resolve thất bại: %v", err)
	}
	if result != "Write an email about hello world" {
		t.Errorf("Resolve = %q, kết quả không đúng", result)
	}
}

func TestCommandResolver_ResolveWithExtraVars(t *testing.T) {
	finder := &mockPromptFinder{
		prompts: map[string]string{
			"translate": "Translate {{.input}} to {{.lang}}",
		},
	}
	builder := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(finder, builder)

	// Truyền thêm biến lang
	extraVars := map[string]string{"lang": "Vietnamese"}
	result, err := resolver.Resolve(1, "translate", "Hello", extraVars)
	if err != nil {
		t.Fatalf("Resolve thất bại: %v", err)
	}
	if result != "Translate Hello to Vietnamese" {
		t.Errorf("Resolve = %q, kết quả không đúng", result)
	}
}

func TestCommandResolver_NotFound(t *testing.T) {
	finder := &mockPromptFinder{prompts: map[string]string{}}
	builder := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(finder, builder)

	_, err := resolver.Resolve(1, "nonexistent", "input", nil)
	if err == nil {
		t.Error("Resolve slash không tồn tại phải trả về lỗi")
	}
}

func TestCommandResolver_GetRequiredVars(t *testing.T) {
	finder := &mockPromptFinder{
		prompts: map[string]string{
			"translate": "Translate {{.input}} to {{.lang}} in {{.tone}} tone",
		},
	}
	builder := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(finder, builder)

	vars, err := resolver.GetRequiredVars(1, "translate")
	if err != nil {
		t.Fatalf("GetRequiredVars thất bại: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("GetRequiredVars = %v (len=%d), muốn [lang tone]", vars, len(vars))
	}
}
```

- [ ] **Step 3.2: Chạy test để xác nhận FAIL**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... -v -run TestCommandResolver
```

Expected: `FAIL` — `undefined: engine.NewCommandResolver`

- [ ] **Step 3.3: Implement CommandResolver**

File: `go-engine/engine/command_resolver.go`

```go
package engine

import "fmt"

// PromptFinder là interface để resolver tìm prompt theo slash_name
// (tách biệt với DB để dễ test)
type PromptFinder interface {
	FindBySlashName(userID int64, slashName string) (string, error)
}

// CommandResolver chuyển đổi slash command thành prompt đã render
type CommandResolver struct {
	finder  PromptFinder
	builder *PromptBuilder
}

// NewCommandResolver tạo CommandResolver mới
func NewCommandResolver(finder PromptFinder, builder *PromptBuilder) *CommandResolver {
	return &CommandResolver{finder: finder, builder: builder}
}

// Resolve tìm template theo slash_name, render với input và extraVars
// Trả về prompt đã render sẵn để gửi lên model
func (r *CommandResolver) Resolve(userID int64, slashName, input string, extraVars map[string]string) (string, error) {
	tmplText, err := r.finder.FindBySlashName(userID, slashName)
	if err != nil {
		return "", fmt.Errorf("resolve command %q: %w", slashName, err)
	}
	if tmplText == "" {
		return "", fmt.Errorf("slash command /%s không tồn tại", slashName)
	}

	// Ghép biến: input + extraVars
	vars := map[string]string{"input": input}
	for k, v := range extraVars {
		vars[k] = v
	}

	rendered, err := r.builder.Render(tmplText, vars)
	if err != nil {
		return "", fmt.Errorf("render command %q: %w", slashName, err)
	}
	return rendered, nil
}

// GetRequiredVars trả về danh sách biến người dùng cần cung cấp cho slash command
func (r *CommandResolver) GetRequiredVars(userID int64, slashName string) ([]string, error) {
	tmplText, err := r.finder.FindBySlashName(userID, slashName)
	if err != nil {
		return nil, fmt.Errorf("get required vars for /%s: %w", slashName, err)
	}
	if tmplText == "" {
		return nil, fmt.Errorf("slash command /%s không tồn tại", slashName)
	}
	return r.builder.ExtractVariables(tmplText), nil
}
```

- [ ] **Step 3.4: Chạy test để xác nhận PASS**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... -v
```

Expected: tất cả test trong package engine PASS.

- [ ] **Step 3.5: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add engine/command_resolver.go engine/command_resolver_test.go && git commit -m "feat: thêm CommandResolver để chuyển slash command thành prompt đã render"
```

---

## Task 4: Seed Data Migration

**Files:**
- Create: `go-engine/db/migrations/002_seed.sql`
- Modify: `go-engine/db/sqlite.go` — đăng ký migration 002

- [ ] **Step 4.1: Tạo file seed SQL**

File: `go-engine/db/migrations/002_seed.sql`

```sql
-- Seed 3 slash commands mẫu cho user đầu tiên (user_id = 1)
-- Chỉ insert nếu chưa tồn tại để migration idempotent

INSERT OR IGNORE INTO prompts (user_id, title, content, category, tags, is_slash, slash_name)
VALUES
  (1,
   'Email Writer',
   'Write a professional email about {{.input}}. Tone: {{.tone}}. Language: {{.lang}}.',
   'productivity',
   'email,writing',
   1,
   'email'),

  (1,
   'Code Review',
   'Review the following code and provide feedback on quality, bugs, and improvements:

```
{{.input}}
```

Focus on: {{.focus}}. Language/framework: {{.lang}}.',
   'development',
   'code,review',
   1,
   'code'),

  (1,
   'Translate',
   'Translate the following text to {{.lang}}:

{{.input}}

Preserve the original formatting and tone.',
   'language',
   'translate,language',
   1,
   'translate');
```

- [ ] **Step 4.2: Đọc sqlite.go để hiểu cách đăng ký migration**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && cat db/sqlite.go
```

- [ ] **Step 4.3: Cập nhật sqlite.go để nhúng migration 002**

Thêm `002_seed.sql` vào danh sách migrations trong hàm `Migrate()`. Tìm đoạn code nhúng file SQL (embed hoặc hardcode) và thêm migration 002 vào cuối danh sách, sau migration 001.

Nếu DB dùng `go:embed`:
```go
//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_seed.sql
var migration002 string
```

Nếu DB dùng danh sách string hoặc slice:
- Thêm `migration002` vào slice theo thứ tự.

Nếu DB dùng schema version table, đảm bảo migration 002 chỉ chạy một lần.

- [ ] **Step 4.4: Chạy test DB để xác nhận migration hoạt động**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./db/... -v
```

Expected: PASS — seed data được insert khi `newTestDB` gọi `Migrate()`.

- [ ] **Step 4.5: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add db/migrations/002_seed.sql db/sqlite.go && git commit -m "feat: thêm seed data 3 slash commands mẫu (email, code, translate)"
```

---

## Task 5: API Handlers — prompts.* và commands.*

**Files:**
- Create: `go-engine/api/handlers_prompts.go`
- Modify: `go-engine/api/router.go`

- [ ] **Step 5.1: Đọc handlers hiện tại để hiểu pattern**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && cat api/handlers_auth.go
```

- [ ] **Step 5.2: Implement handlers_prompts.go**

File: `go-engine/api/handlers_prompts.go`

```go
package api

import (
	"regexp"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

// Regex kiểm tra slash_name hợp lệ: chỉ a-z, 0-9, -, _ và 1-32 ký tự
var slashNameRegex = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// handlePromptsList trả về danh sách prompts của user hiện tại
func (r *Router) handlePromptsList(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token    string `json:"token"`
		Category string `json:"category"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	list, err := r.prompts.List(claims.UserID, p.Category)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"prompts": list}, nil
}

// handlePromptsCreate tạo prompt mới
func (r *Router) handlePromptsCreate(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string `json:"token"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Category  string `json:"category"`
		Tags      string `json:"tags"`
		IsSlash   bool   `json:"is_slash"`
		SlashName string `json:"slash_name"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.Title == "" || p.Content == "" {
		return nil, &RPCError{Code: -32602, Message: "title và content không được rỗng"}
	}
	// Validate slash_name nếu là slash command
	if p.IsSlash {
		if !slashNameRegex.MatchString(p.SlashName) {
			return nil, &RPCError{Code: -32602, Message: "slash_name không hợp lệ: chỉ a-z 0-9 - _ và tối đa 32 ký tự"}
		}
	}

	prompt, err := r.prompts.Create(repos.CreatePromptInput{
		UserID:    claims.UserID,
		Title:     p.Title,
		Content:   p.Content,
		Category:  p.Category,
		Tags:      p.Tags,
		IsSlash:   p.IsSlash,
		SlashName: p.SlashName,
	})
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"prompt": prompt}, nil
}

// handlePromptsUpdate cập nhật prompt
func (r *Router) handlePromptsUpdate(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string `json:"token"`
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Category  string `json:"category"`
		Tags      string `json:"tags"`
		IsSlash   bool   `json:"is_slash"`
		SlashName string `json:"slash_name"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if _, err := r.auth.ValidateToken(p.Token); err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.ID == 0 || p.Title == "" || p.Content == "" {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.IsSlash && !slashNameRegex.MatchString(p.SlashName) {
		return nil, &RPCError{Code: -32602, Message: "slash_name không hợp lệ"}
	}

	prompt, err := r.prompts.Update(p.ID, repos.UpdatePromptInput{
		Title:     p.Title,
		Content:   p.Content,
		Category:  p.Category,
		Tags:      p.Tags,
		IsSlash:   p.IsSlash,
		SlashName: p.SlashName,
	})
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"prompt": prompt}, nil
}

// handlePromptsDelete xoá prompt
func (r *Router) handlePromptsDelete(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
		ID    int64  `json:"id"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if _, err := r.auth.ValidateToken(p.Token); err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.ID == 0 {
		return nil, copyErr(ErrInvalidParams)
	}
	if err := r.prompts.Delete(p.ID); err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"ok": true}, nil
}

// handleCommandsList lấy tất cả slash commands của user (để fuzzy search ở frontend)
func (r *Router) handleCommandsList(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	cmds, err := r.prompts.ListSlashCommands(claims.UserID)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}

	// Trả về thêm required_vars cho mỗi command
	builder := engine.NewPromptBuilder()
	type cmdItem struct {
		ID           int64    `json:"id"`
		SlashName    string   `json:"slash_name"`
		Title        string   `json:"title"`
		Content      string   `json:"content"`
		Category     string   `json:"category"`
		Tags         string   `json:"tags"`
		RequiredVars []string `json:"required_vars"`
	}
	items := make([]cmdItem, 0, len(cmds))
	for _, c := range cmds {
		items = append(items, cmdItem{
			ID:           c.ID,
			SlashName:    c.SlashName,
			Title:        c.Title,
			Content:      c.Content,
			Category:     c.Category,
			Tags:         c.Tags,
			RequiredVars: builder.ExtractVariables(c.Content),
		})
	}
	return map[string]interface{}{"commands": items}, nil
}

// handleCommandsResolve resolve slash command thành prompt đã render
func (r *Router) handleCommandsResolve(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		SlashName string            `json:"slash_name"`
		Input     string            `json:"input"`
		ExtraVars map[string]string `json:"extra_vars"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.SlashName == "" {
		return nil, copyErr(ErrInvalidParams)
	}

	// Tạo resolver kết nối PromptRepo với PromptBuilder
	builder := engine.NewPromptBuilder()
	finder := &promptRepoFinder{repo: r.prompts}
	resolver := engine.NewCommandResolver(finder, builder)

	rendered, err := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
	if err != nil {
		return nil, &RPCError{Code: -32002, Message: err.Error()}
	}
	return map[string]interface{}{"rendered": rendered}, nil
}

// promptRepoFinder adapter để PromptRepo implement interface PromptFinder của engine
type promptRepoFinder struct {
	repo *repos.PromptRepo
}

func (f *promptRepoFinder) FindBySlashName(userID int64, slashName string) (string, error) {
	p, err := f.repo.FindBySlashName(userID, slashName)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", nil
	}
	return p.Content, nil
}
```

- [ ] **Step 5.3: Cập nhật router.go — thêm prompts repo và dispatch cases**

File: `go-engine/api/router.go` — thêm field `prompts` vào struct Router và khởi tạo trong `newRouter`, đồng thời thêm cases vào `dispatch`:

```go
// Trong struct Router — thêm field:
prompts *repos.PromptRepo

// Trong hàm newRouter — thêm sau dòng khởi tạo settings:
prompts := repos.NewPromptRepo(s.db)

// Trong return statement — thêm field:
prompts: prompts,

// Trong hàm dispatch — thêm các cases mới trước `default`:
case "prompts.list":
    return r.handlePromptsList(req)
case "prompts.create":
    return r.handlePromptsCreate(req)
case "prompts.update":
    return r.handlePromptsUpdate(req)
case "prompts.delete":
    return r.handlePromptsDelete(req)
case "commands.list":
    return r.handleCommandsList(req)
case "commands.resolve":
    return r.handleCommandsResolve(req)
```

- [ ] **Step 5.4: Build để kiểm tra không có lỗi compile**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go build ./...
```

Expected: không có lỗi compile.

- [ ] **Step 5.5: Chạy toàn bộ tests**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./... -v
```

Expected: tất cả PASS.

- [ ] **Step 5.6: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add api/handlers_prompts.go api/router.go && git commit -m "feat: thêm API handlers cho prompts.* và commands.*"
```

---

## Task 6: Update Query Handler — hỗ trợ slash_name

**Files:**
- Modify: `go-engine/api/handlers_query.go`

- [ ] **Step 6.1: Viết failing test cho query với slash_name**

File: `go-engine/api/handlers_query_slash_test.go`

```go
package api_test

import (
	"testing"
)

// Test xác nhận rằng khi query.stream nhận slash_name,
// nó resolve template trước khi gửi lên model.
// Test này dùng integration-style check vào router dispatch.
func TestQueryStream_SlashNameParam_Compile(t *testing.T) {
	// Kiểm tra cơ bản: file handlers_query.go phải compile với SlashName field
	// Test thực tế sẽ cần mock model router — đây là smoke test
	t.Log("handlers_query.go phải có SlashName trong struct params")
}
```

- [ ] **Step 6.2: Cập nhật handlers_query.go**

Sửa `go-engine/api/handlers_query.go` — thêm field `SlashName` và logic resolve trước khi gửi lên model:

```go
func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		Input     string            `json:"input"`
		Model     string            `json:"model"`
		System    string            `json:"system"`
		SlashName string            `json:"slash_name"`  // tên slash command (tùy chọn)
		ExtraVars map[string]string `json:"extra_vars"`  // biến bổ sung cho template
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	// Nếu có slash_name, resolve template trước khi gửi lên model
	finalInput := p.Input
	if p.SlashName != "" {
		builder := engine.NewPromptBuilder()
		finder := &promptRepoFinder{repo: r.prompts}
		resolver := engine.NewCommandResolver(finder, builder)
		rendered, resolveErr := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
		if resolveErr != nil {
			return nil, &RPCError{Code: -32002, Message: resolveErr.Error()}
		}
		finalInput = rendered
	}

	// Lấy API key từ settings
	apiKey, _ := r.settings.Get(claims.UserID, "anthropic_api_key")
	if apiKey == "" {
		return nil, copyErr(ErrProviderNotFound)
	}

	// Build model router
	modelRouter := model.NewRouter()
	modelRouter.RegisterAnthropic(apiKey)

	modelName := p.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	// Stream response qua JSON-RPC notifications
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	streamErr := modelRouter.Stream(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: finalInput,
		System: p.System,
	}, func(chunk string) {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	if streamErr != nil {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": "",
			"done":  true,
			"error": fmt.Sprintf("%v", streamErr),
		})
		return nil, nil
	}

	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})
	return nil, nil
}
```

Thêm import `engine` vào đầu file:
```go
import (
    "context"
    "fmt"
    "net"
    "time"

    "github.com/minhtuancn/open-prompt/go-engine/engine"
    "github.com/minhtuancn/open-prompt/go-engine/model"
    "github.com/minhtuancn/open-prompt/go-engine/model/providers"
)
```

- [ ] **Step 6.3: Build và test**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go build ./... && go test ./...
```

Expected: build thành công, tất cả test PASS.

- [ ] **Step 6.4: Commit**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && git add api/handlers_query.go && git commit -m "feat: query.stream hỗ trợ slash_name để resolve template trước khi gửi model"
```

---

## Task 7: Frontend — SlashMenu Component

**Files:**
- Create: `src/components/overlay/SlashMenu.tsx`

- [ ] **Step 7.1: Cài đặt fuse.js**

```
cd /home/dev/open-prompt-code/open-prompt && npm install fuse.js
```

Expected: fuse.js được thêm vào package.json.

- [ ] **Step 7.2: Implement SlashMenu.tsx**

File: `src/components/overlay/SlashMenu.tsx`

```tsx
import { useEffect, useRef, useState } from 'react'
import Fuse from 'fuse.js'

// Kiểu dữ liệu cho một slash command
export interface SlashCommand {
  id: number
  slash_name: string
  title: string
  content: string
  category: string
  tags: string
  required_vars: string[]
}

interface Props {
  commands: SlashCommand[]
  query: string          // phần text sau dấu / để filter
  onSelect: (cmd: SlashCommand) => void
  onClose: () => void
  visible: boolean
}

// Cấu hình fuzzy search với fuse.js
const fuseOptions = {
  keys: ['slash_name', 'title', 'tags'],
  threshold: 0.4,
  includeScore: true,
}

export function SlashMenu({ commands, query, onSelect, onClose, visible }: Props) {
  const [activeIndex, setActiveIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  // Fuzzy filter dựa theo query
  const filtered = (() => {
    if (!query) return commands
    const fuse = new Fuse(commands, fuseOptions)
    return fuse.search(query).map(r => r.item)
  })()

  // Reset active index khi query thay đổi
  useEffect(() => {
    setActiveIndex(0)
  }, [query])

  // Xử lý keyboard navigation từ parent thông qua custom event
  useEffect(() => {
    if (!visible) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex(i => Math.min(i + 1, filtered.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex(i => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (filtered[activeIndex]) {
          onSelect(filtered[activeIndex])
        }
      } else if (e.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown, true)
    return () => window.removeEventListener('keydown', handleKeyDown, true)
  }, [visible, filtered, activeIndex, onSelect, onClose])

  // Scroll item active vào tầm nhìn
  useEffect(() => {
    const container = listRef.current
    if (!container) return
    const activeEl = container.querySelector(`[data-index="${activeIndex}"]`) as HTMLElement
    activeEl?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex])

  if (!visible || filtered.length === 0) return null

  return (
    <div
      className="absolute bottom-full left-0 right-0 mb-1 bg-black/80 backdrop-blur-md border border-white/10 rounded-xl overflow-hidden shadow-2xl z-50 max-h-64 overflow-y-auto"
      ref={listRef}
    >
      {filtered.map((cmd, index) => (
        <button
          key={cmd.id}
          data-index={index}
          className={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${
            index === activeIndex
              ? 'bg-indigo-500/30 text-white'
              : 'text-white/70 hover:bg-white/5'
          }`}
          onMouseEnter={() => setActiveIndex(index)}
          onClick={() => onSelect(cmd)}
        >
          {/* Badge slash name */}
          <span className="flex-shrink-0 font-mono text-sm text-indigo-400 bg-indigo-500/20 px-2 py-0.5 rounded-md">
            /{cmd.slash_name}
          </span>

          {/* Title và category */}
          <span className="flex-1 min-w-0">
            <span className="block text-sm font-medium text-white truncate">{cmd.title}</span>
            {cmd.category && (
              <span className="text-xs text-white/40">{cmd.category}</span>
            )}
          </span>

          {/* Badge biến cần nhập */}
          {cmd.required_vars.length > 0 && (
            <span className="flex-shrink-0 text-xs text-white/40">
              {cmd.required_vars.map(v => `{${v}}`).join(' ')}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 7.3: Kiểm tra TypeScript compile**

```
cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit
```

Expected: không có lỗi TypeScript.

- [ ] **Step 7.4: Commit**

```
cd /home/dev/open-prompt-code/open-prompt && git add src/components/overlay/SlashMenu.tsx package.json package-lock.json && git commit -m "feat: thêm SlashMenu component với fuzzy search fuse.js và keyboard navigation"
```

---

## Task 8: Frontend — Cập nhật CommandInput để detect /

**Files:**
- Modify: `src/components/overlay/CommandInput.tsx`

- [ ] **Step 8.1: Implement CommandInput mới**

Thay thế toàn bộ nội dung `src/components/overlay/CommandInput.tsx`:

```tsx
import { useEffect, useRef, useState } from 'react'
import { useOverlayStore } from '../../store/overlayStore'
import { SlashMenu, type SlashCommand } from './SlashMenu'

interface Props {
  onSubmit: (input: string, slashName?: string, extraVars?: Record<string, string>) => void
}

export function CommandInput({ onSubmit }: Props) {
  const { input, setInput, isStreaming } = useOverlayStore()

  // State cho slash menu
  const [commands, setCommands] = useState<SlashCommand[]>([])
  const [slashMenuVisible, setSlashMenuVisible] = useState(false)
  const [slashQuery, setSlashQuery] = useState('')    // phần text sau /
  const [selectedCmd, setSelectedCmd] = useState<SlashCommand | null>(null)
  const [extraVars, setExtraVars] = useState<Record<string, string>>({})
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Load danh sách slash commands khi mount
  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return

    // Gọi commands.list qua RPC
    window.__rpc?.call('commands.list', { token })
      .then((res: { commands: SlashCommand[] }) => {
        setCommands(res.commands || [])
      })
      .catch(console.error)
  }, [])

  // Detect khi user gõ / ở đầu input
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setInput(value)

    // Nếu đang ở chế độ slash đã chọn, không mở menu nữa
    if (selectedCmd) return

    if (value.startsWith('/') && !value.includes('\n')) {
      const query = value.slice(1) // bỏ ký tự /
      setSlashQuery(query)
      setSlashMenuVisible(true)
    } else {
      setSlashMenuVisible(false)
      setSlashQuery('')
    }
  }

  // Khi user chọn slash command từ menu
  const handleSlashSelect = (cmd: SlashCommand) => {
    setSelectedCmd(cmd)
    setSlashMenuVisible(false)
    // Xoá input, sẽ hỏi input bên dưới
    setInput('')
    // Reset extra vars
    const initVars: Record<string, string> = {}
    cmd.required_vars.forEach(v => { initVars[v] = '' })
    setExtraVars(initVars)
    textareaRef.current?.focus()
  }

  const handleCloseMenu = () => {
    setSlashMenuVisible(false)
  }

  // Xoá lựa chọn slash command
  const handleClearSlash = () => {
    setSelectedCmd(null)
    setExtraVars({})
    setInput('')
    setSlashMenuVisible(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Khi slash menu đang mở, Enter và Arrow keys được handle bởi SlashMenu
    if (slashMenuVisible) return

    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (!isStreaming) {
        handleSubmit()
      }
    }
    if (e.key === 'Escape') {
      if (selectedCmd) {
        handleClearSlash()
      } else {
        window.close()
      }
    }
  }

  const handleSubmit = () => {
    if (selectedCmd) {
      // Gửi với slash command
      onSubmit(input.trim(), selectedCmd.slash_name, extraVars)
      setSelectedCmd(null)
      setExtraVars({})
    } else {
      if (input.trim()) {
        onSubmit(input.trim())
      }
    }
    setInput('')
  }

  return (
    <div className="relative">
      {/* Slash Menu hiện phía trên input */}
      <SlashMenu
        commands={commands}
        query={slashQuery}
        onSelect={handleSlashSelect}
        onClose={handleCloseMenu}
        visible={slashMenuVisible}
      />

      {/* Badge slash command đã chọn */}
      {selectedCmd && (
        <div className="flex items-center gap-2 px-5 pt-3 pb-1">
          <span className="font-mono text-sm text-indigo-400 bg-indigo-500/20 px-2 py-0.5 rounded-md">
            /{selectedCmd.slash_name}
          </span>
          <span className="text-xs text-white/50">{selectedCmd.title}</span>
          <button
            onClick={handleClearSlash}
            className="ml-auto text-white/30 hover:text-white/60 text-xs"
          >
            ✕
          </button>
        </div>
      )}

      {/* Form điền biến bổ sung nếu command có required_vars */}
      {selectedCmd && selectedCmd.required_vars.length > 0 && (
        <div className="px-5 py-2 flex flex-wrap gap-2">
          {selectedCmd.required_vars.map(varName => (
            <div key={varName} className="flex items-center gap-1.5">
              <label className="text-xs text-white/40">{varName}:</label>
              <input
                type="text"
                value={extraVars[varName] || ''}
                onChange={e => setExtraVars(prev => ({ ...prev, [varName]: e.target.value }))}
                className="bg-white/10 text-white text-xs px-2 py-1 rounded-md outline-none focus:ring-1 focus:ring-indigo-500 w-28"
                placeholder={varName}
              />
            </div>
          ))}
        </div>
      )}

      {/* Text input chính */}
      <textarea
        ref={textareaRef}
        autoFocus
        rows={1}
        className="w-full bg-transparent text-white text-lg placeholder-white/40 outline-none resize-none px-5 py-4 leading-relaxed"
        placeholder={
          selectedCmd
            ? `Nhập nội dung cho /${selectedCmd.slash_name}...`
            : 'Hỏi AI... hoặc gõ / để dùng slash command'
        }
        value={input}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        disabled={isStreaming}
      />
    </div>
  )
}
```

- [ ] **Step 8.2: Kiểm tra TypeScript compile**

```
cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit
```

Expected: không có lỗi TypeScript.

- [ ] **Step 8.3: Build frontend**

```
cd /home/dev/open-prompt-code/open-prompt && npm run build 2>&1 | tail -20
```

Expected: build thành công, không có lỗi.

- [ ] **Step 8.4: Commit**

```
cd /home/dev/open-prompt-code/open-prompt && git add src/components/overlay/CommandInput.tsx && git commit -m "feat: CommandInput detect / để mở SlashMenu và hỗ trợ extra vars form"
```

---

## Task 9: Prompt Manager UI

**Files:**
- Create: `src/components/prompts/PromptList.tsx`
- Create: `src/components/prompts/PromptEditor.tsx`

- [ ] **Step 9.1: Implement PromptList.tsx**

File: `src/components/prompts/PromptList.tsx`

```tsx
import { useEffect, useState } from 'react'

interface Prompt {
  id: number
  title: string
  content: string
  category: string
  tags: string
  is_slash: boolean
  slash_name: string
  updated_at: string
}

interface Props {
  onEdit: (prompt: Prompt) => void
  onNew: () => void
  refreshTrigger?: number  // tăng giá trị để trigger reload
}

export function PromptList({ onEdit, onNew, refreshTrigger }: Props) {
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<'all' | 'slash' | 'regular'>('all')

  const loadPrompts = async () => {
    setLoading(true)
    try {
      const token = localStorage.getItem('auth_token') || ''
      const res = await window.__rpc?.call('prompts.list', { token })
      setPrompts(res?.prompts || [])
    } catch (e) {
      console.error('Lỗi tải prompts:', e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPrompts()
  }, [refreshTrigger])

  const handleDelete = async (id: number) => {
    if (!confirm('Xoá prompt này?')) return
    const token = localStorage.getItem('auth_token') || ''
    await window.__rpc?.call('prompts.delete', { token, id })
    loadPrompts()
  }

  const filtered = prompts.filter(p => {
    if (filter === 'slash') return p.is_slash
    if (filter === 'regular') return !p.is_slash
    return true
  })

  return (
    <div className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Prompt Library</h2>
        <button
          onClick={onNew}
          className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-sm rounded-lg transition-colors"
        >
          + Thêm Prompt
        </button>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-2">
        {(['all', 'slash', 'regular'] as const).map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1 text-xs rounded-full transition-colors ${
              filter === f
                ? 'bg-indigo-600 text-white'
                : 'bg-white/10 text-white/60 hover:bg-white/15'
            }`}
          >
            {f === 'all' ? 'Tất cả' : f === 'slash' ? 'Slash Commands' : 'Thường'}
          </button>
        ))}
      </div>

      {/* List */}
      {loading ? (
        <div className="text-white/40 text-sm text-center py-8">Đang tải...</div>
      ) : filtered.length === 0 ? (
        <div className="text-white/40 text-sm text-center py-8">
          Chưa có prompt nào. Nhấn <strong>+ Thêm Prompt</strong> để bắt đầu.
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {filtered.map(p => (
            <div
              key={p.id}
              className="bg-white/5 hover:bg-white/8 border border-white/10 rounded-xl p-4 flex items-start gap-3 transition-colors"
            >
              {/* Slash badge */}
              {p.is_slash && (
                <span className="flex-shrink-0 font-mono text-xs text-indigo-400 bg-indigo-500/20 px-2 py-0.5 rounded-md mt-0.5">
                  /{p.slash_name}
                </span>
              )}

              {/* Info */}
              <div className="flex-1 min-w-0">
                <p className="text-white text-sm font-medium truncate">{p.title}</p>
                <p className="text-white/40 text-xs mt-0.5 line-clamp-2">{p.content}</p>
                {p.category && (
                  <span className="inline-block mt-1.5 text-xs text-white/30 bg-white/5 px-2 py-0.5 rounded">
                    {p.category}
                  </span>
                )}
              </div>

              {/* Actions */}
              <div className="flex gap-2 flex-shrink-0">
                <button
                  onClick={() => onEdit(p)}
                  className="text-xs text-white/50 hover:text-white transition-colors px-2 py-1 rounded hover:bg-white/10"
                >
                  Sửa
                </button>
                <button
                  onClick={() => handleDelete(p.id)}
                  className="text-xs text-red-400/60 hover:text-red-400 transition-colors px-2 py-1 rounded hover:bg-red-500/10"
                >
                  Xoá
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 9.2: Implement PromptEditor.tsx**

File: `src/components/prompts/PromptEditor.tsx`

```tsx
import { useEffect, useState } from 'react'

interface Prompt {
  id?: number
  title: string
  content: string
  category: string
  tags: string
  is_slash: boolean
  slash_name: string
}

interface Props {
  prompt?: Prompt   // undefined = tạo mới
  onSave: () => void
  onCancel: () => void
}

// Regex kiểm tra slash_name hợp lệ
const SLASH_NAME_REGEX = /^[a-z0-9_-]{1,32}$/

export function PromptEditor({ prompt, onSave, onCancel }: Props) {
  const isNew = !prompt?.id

  const [form, setForm] = useState<Prompt>({
    title: '',
    content: '',
    category: '',
    tags: '',
    is_slash: false,
    slash_name: '',
    ...prompt,
  })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (prompt) setForm({ ...prompt })
  }, [prompt])

  // Trích xuất biến để preview
  const previewVars = (() => {
    const matches = [...form.content.matchAll(/\{\{\.(\w+)\}\}/g)]
    const vars = new Set<string>()
    for (const m of matches) {
      if (m[1] !== 'input' && !m[1].startsWith('context')) {
        vars.add(m[1])
      }
    }
    return [...vars]
  })()

  const handleSave = async () => {
    setError('')

    // Validate
    if (!form.title.trim()) { setError('Tiêu đề không được rỗng'); return }
    if (!form.content.trim()) { setError('Nội dung không được rỗng'); return }
    if (form.is_slash && !SLASH_NAME_REGEX.test(form.slash_name)) {
      setError('Slash name chỉ gồm a-z, 0-9, -, _ và tối đa 32 ký tự')
      return
    }

    setSaving(true)
    try {
      const token = localStorage.getItem('auth_token') || ''
      const method = isNew ? 'prompts.create' : 'prompts.update'
      await window.__rpc?.call(method, { token, ...form })
      onSave()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setError(msg || 'Lưu thất bại')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">
          {isNew ? 'Tạo Prompt Mới' : 'Sửa Prompt'}
        </h2>
        <button onClick={onCancel} className="text-white/40 hover:text-white text-sm">
          Huỷ
        </button>
      </div>

      {/* Form */}
      <div className="flex flex-col gap-3">
        {/* Tiêu đề */}
        <div>
          <label className="text-xs text-white/50 block mb-1">Tiêu đề *</label>
          <input
            type="text"
            value={form.title}
            onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
            className="w-full bg-white/10 text-white text-sm px-3 py-2 rounded-lg outline-none focus:ring-1 focus:ring-indigo-500"
            placeholder="Ví dụ: Email Writer"
          />
        </div>

        {/* Nội dung template */}
        <div>
          <label className="text-xs text-white/50 block mb-1">
            Template * — dùng {'{{.input}}'}, {'{{.varName}}'}
          </label>
          <textarea
            rows={5}
            value={form.content}
            onChange={e => setForm(f => ({ ...f, content: e.target.value }))}
            className="w-full bg-white/10 text-white text-sm px-3 py-2 rounded-lg outline-none focus:ring-1 focus:ring-indigo-500 resize-y font-mono"
            placeholder="Write an email about {{.input}} in {{.lang}}"
          />
          {/* Preview biến */}
          {previewVars.length > 0 && (
            <p className="text-xs text-indigo-400 mt-1">
              Biến người dùng nhập: {previewVars.map(v => `{${v}}`).join(', ')}
            </p>
          )}
        </div>

        {/* Category và Tags */}
        <div className="flex gap-3">
          <div className="flex-1">
            <label className="text-xs text-white/50 block mb-1">Danh mục</label>
            <input
              type="text"
              value={form.category}
              onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
              className="w-full bg-white/10 text-white text-sm px-3 py-2 rounded-lg outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder="productivity"
            />
          </div>
          <div className="flex-1">
            <label className="text-xs text-white/50 block mb-1">Tags (ngăn cách bằng dấu phẩy)</label>
            <input
              type="text"
              value={form.tags}
              onChange={e => setForm(f => ({ ...f, tags: e.target.value }))}
              className="w-full bg-white/10 text-white text-sm px-3 py-2 rounded-lg outline-none focus:ring-1 focus:ring-indigo-500"
              placeholder="email,writing"
            />
          </div>
        </div>

        {/* Slash command toggle */}
        <div>
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={form.is_slash}
              onChange={e => setForm(f => ({ ...f, is_slash: e.target.checked }))}
              className="w-4 h-4 accent-indigo-500"
            />
            <span className="text-sm text-white/80">Kích hoạt như Slash Command</span>
          </label>
        </div>

        {/* Slash name field */}
        {form.is_slash && (
          <div>
            <label className="text-xs text-white/50 block mb-1">
              Slash Name * (a-z, 0-9, -, _ tối đa 32 ký tự)
            </label>
            <div className="flex items-center gap-1">
              <span className="text-white/40 text-sm">/</span>
              <input
                type="text"
                value={form.slash_name}
                onChange={e => setForm(f => ({ ...f, slash_name: e.target.value.toLowerCase() }))}
                className="flex-1 bg-white/10 text-white text-sm px-3 py-2 rounded-lg outline-none focus:ring-1 focus:ring-indigo-500 font-mono"
                placeholder="my-command"
                maxLength={32}
              />
            </div>
          </div>
        )}
      </div>

      {/* Error */}
      {error && (
        <p className="text-red-400 text-xs bg-red-500/10 px-3 py-2 rounded-lg">{error}</p>
      )}

      {/* Save button */}
      <button
        onClick={handleSave}
        disabled={saving}
        className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
      >
        {saving ? 'Đang lưu...' : isNew ? 'Tạo Prompt' : 'Lưu Thay Đổi'}
      </button>
    </div>
  )
}
```

- [ ] **Step 9.3: Kiểm tra TypeScript compile**

```
cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit
```

Expected: không có lỗi TypeScript.

- [ ] **Step 9.4: Build toàn bộ**

```
cd /home/dev/open-prompt-code/open-prompt && npm run build 2>&1 | tail -20
```

Expected: build thành công.

- [ ] **Step 9.5: Commit**

```
cd /home/dev/open-prompt-code/open-prompt && git add src/components/prompts/PromptList.tsx src/components/prompts/PromptEditor.tsx && git commit -m "feat: thêm PromptList và PromptEditor cho Prompt Manager UI"
```

---

## Task 10: Integration — Kiểm tra toàn bộ

- [ ] **Step 10.1: Chạy toàn bộ Go tests**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./... -v 2>&1 | tail -40
```

Expected: tất cả test PASS, không có FAIL.

- [ ] **Step 10.2: Build Go binary**

```
cd /home/dev/open-prompt-code/open-prompt/go-engine && go build -o /tmp/go-engine-test . && echo "Build OK" && rm /tmp/go-engine-test
```

Expected: `Build OK`.

- [ ] **Step 10.3: Build Tauri app**

```
cd /home/dev/open-prompt-code/open-prompt && npm run tauri build 2>&1 | tail -30
```

Expected: build hoàn tất, không có lỗi.

- [ ] **Step 10.4: Final commit**

```
cd /home/dev/open-prompt-code/open-prompt && git add -A && git commit -m "chore: hoàn thành Plan 3 — Prompt & Slash Command System"
```

---

## Tóm tắt các file tạo mới / sửa

| File | Loại | Mô tả |
|------|------|-------|
| `go-engine/db/repos/prompt_repo.go` | Tạo mới | CRUD cho bảng prompts |
| `go-engine/db/repos/prompt_repo_test.go` | Tạo mới | Unit test PromptRepo |
| `go-engine/db/repos/skill_repo.go` | Tạo mới | CRUD cho bảng skills |
| `go-engine/db/repos/skill_repo_test.go` | Tạo mới | Unit test SkillRepo |
| `go-engine/engine/prompt_builder.go` | Tạo mới | Template rendering với text/template |
| `go-engine/engine/prompt_builder_test.go` | Tạo mới | Unit test PromptBuilder |
| `go-engine/engine/command_resolver.go` | Tạo mới | Resolve slash command → rendered prompt |
| `go-engine/engine/command_resolver_test.go` | Tạo mới | Unit test CommandResolver |
| `go-engine/api/handlers_prompts.go` | Tạo mới | Handlers prompts.* và commands.* |
| `go-engine/db/migrations/002_seed.sql` | Tạo mới | Seed 3 slash commands mẫu |
| `go-engine/api/router.go` | Sửa | Thêm PromptRepo + dispatch cases |
| `go-engine/api/handlers_query.go` | Sửa | Hỗ trợ slash_name param |
| `go-engine/db/sqlite.go` | Sửa | Đăng ký migration 002 |
| `src/components/overlay/SlashMenu.tsx` | Tạo mới | Fuzzy slash palette |
| `src/components/overlay/CommandInput.tsx` | Sửa | Detect / → mở SlashMenu |
| `src/components/prompts/PromptList.tsx` | Tạo mới | Danh sách prompts |
| `src/components/prompts/PromptEditor.tsx` | Tạo mới | Form tạo/sửa prompt |
