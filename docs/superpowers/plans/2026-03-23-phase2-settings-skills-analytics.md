# Open Prompt — Phase 2: Settings, Skills & Analytics

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hoàn thiện Skills API, Analytics API, Settings UI (5 tabs + Analytics), Skills UI, Analytics panel và i18n để ứng dụng usable đầy đủ cho end-user.

**Architecture:** Go Engine bổ sung handlers cho `skills.*`, `analytics.*` và ghi history sau mỗi query. React thêm SettingsModal (6 tabs: Providers, Skills, Hotkey, Appearance, Language, Thống kê), SkillList/SkillEditor, UsageStats, và i18n với 2 locale (vi/en). Settings mở từ gear icon trong overlay. Bảng `history` đã có sẵn trong migration `001_init.sql` — không cần migration mới.

**Tech Stack:** Go 1.22+, modernc.org/sqlite, React 18 + Zustand + TailwindCSS, TypeScript, fuse.js

**Test pattern:** Dùng `setupServer(t)`, `callRPC(t, addr, secret, method, params)`, `registerAndLogin()` và `resultMap()` giống các test hiện tại trong `api/handlers_prompts_test.go`. Không có `NewTestServer`.

---

## Spec Reference

`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

---

## File Map

### Go Engine
| File | Action | Trách nhiệm |
|------|--------|-------------|
| `go-engine/db/repos/skill_repo.go` | Modify | Thêm `Update` method |
| `go-engine/db/repos/history_repo.go` | Create | Insert và query history (bảng `history` đã có trong migration) |
| `go-engine/api/handlers_skills.go` | Create | `skills.list/create/update/delete` handlers |
| `go-engine/api/handlers_skills_test.go` | Create | Tests cho skills handlers |
| `go-engine/api/handlers_analytics.go` | Create | `analytics.summary`, `analytics.by_provider` handlers |
| `go-engine/api/handlers_analytics_test.go` | Create | Tests cho analytics handlers |
| `go-engine/api/handlers_query.go` | Modify | Record history sau mỗi query (full rewrite của function) |
| `go-engine/api/router.go` | Modify | Thêm `skills`, `history` fields + 6 routes mới |

### React Frontend
| File | Action | Trách nhiệm |
|------|--------|-------------|
| `src/store/settingsStore.ts` | Create | Zustand store: locale, theme, fontSize |
| `src/i18n/locales/vi.json` | Create | Tiếng Việt strings |
| `src/i18n/locales/en.json` | Create | English strings |
| `src/i18n/index.ts` | Create | Lookup function `translate(locale, key)` |
| `src/hooks/useI18n.ts` | Create | React hook dùng settingsStore |
| `src/components/skills/SkillList.tsx` | Create | Danh sách skills + delete |
| `src/components/skills/SkillEditor.tsx` | Create | Form tạo/sửa skill |
| `src/components/analytics/UsageStats.tsx` | Create | Analytics summary + by_provider |
| `src/components/settings/SettingsLayout.tsx` | Create | Modal wrapper + tab bar (6 tabs) |
| `src/components/settings/ProvidersTab.tsx` | Create | API keys qua `providers.connect` |
| `src/components/settings/HotkeyTab.tsx` | Create | Xem hotkey hiện tại |
| `src/components/settings/AppearanceTab.tsx` | Create | Font size |
| `src/components/settings/LanguageTab.tsx` | Create | Chọn locale |
| `src/App.tsx` | Modify | Thêm AppState `'settings'` + gear icon |

---

## Task 1: Go — SkillRepo.Update + Skills API Handlers

**Files:**
- Modify: `go-engine/db/repos/skill_repo.go`
- Create: `go-engine/api/handlers_skills.go`
- Create: `go-engine/api/handlers_skills_test.go`
- Modify: `go-engine/api/router.go`

- [ ] **Step 1.1: Thêm Update vào SkillRepo**

Thêm vào cuối `go-engine/db/repos/skill_repo.go`:

```go
// UpdateSkillInput là input để cập nhật skill
type UpdateSkillInput struct {
	Name       string
	PromptID   int64
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
}
// Lưu ý: ConfigJSON được giữ trong UpdateSkillInput để nhất quán với CreateSkillInput và cột config_json trong DB.

// Update cập nhật skill theo ID
func (r *SkillRepo) Update(id int64, input UpdateSkillInput) error {
	var promptID interface{}
	if input.PromptID > 0 {
		promptID = input.PromptID
	}
	_, err := r.db.Exec(
		`UPDATE skills SET name=?, prompt_id=?, prompt_text=?, model=?, provider=?, config_json=?, tags=?
		 WHERE id=?`,
		input.Name, promptID, input.PromptText, input.Model, input.Provider, input.ConfigJSON, input.Tags, id,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	return nil
}
```

- [ ] **Step 1.2: Viết test cho skills handlers (failing)**

Tạo `go-engine/api/handlers_skills_test.go`. Dùng pattern `setupServer` + `callRPC` + `registerAndLogin` giống `handlers_prompts_test.go`:

```go
package api_test

import (
	"encoding/json"
	"testing"
)

func skillFromResult(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(m["skill"])
	if err != nil {
		t.Fatalf("marshal skill: %v", err)
	}
	var s map[string]interface{}
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal skill: %v", err)
	}
	return s
}

func TestSkillsListRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "skills.list", map[string]string{
		"token": "bad-token",
	})
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Errorf("expected -32001, got %v", resp.Error)
	}
}

func TestSkillsList(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "skillsuser1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "skills.list", map[string]string{"token": token})
	if resp.Error != nil {
		t.Fatalf("skills.list error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["skills"]; !exists {
		t.Error("phải có field 'skills' trong response")
	}
}

func TestSkillsCreateAndList(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "skillsuser2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "skills.create", map[string]interface{}{
		"token":       token,
		"name":        "My Skill",
		"prompt_text": "Bạn là trợ lý",
		"provider":    "anthropic",
		"model":       "claude-3-5-sonnet-20241022",
	})
	if resp.Error != nil {
		t.Fatalf("skills.create error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	skill := skillFromResult(t, m)
	if skill["name"] != "My Skill" {
		t.Errorf("name = %v, want My Skill", skill["name"])
	}
	if skill["id"] == nil {
		t.Error("skill phải có id")
	}
}

func TestSkillsCreateRequiresName(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "skillsuser3", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "skills.create", map[string]interface{}{
		"token": token,
		"name":  "",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi name rỗng")
	}
}

func TestSkillsUpdate(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "skillsuser4", "pass1234")

	createResp := callRPC(t, addr, "test-secret-16chars", "skills.create", map[string]interface{}{
		"token": token, "name": "Old Name",
	})
	if createResp.Error != nil {
		t.Fatalf("create error: %v", createResp.Error)
	}
	skill := skillFromResult(t, resultMap(t, createResp))
	id := skill["id"].(float64)

	updateResp := callRPC(t, addr, "test-secret-16chars", "skills.update", map[string]interface{}{
		"token": token, "id": int64(id), "name": "New Name",
	})
	if updateResp.Error != nil {
		t.Fatalf("update error: %v", updateResp.Error)
	}
	updated := skillFromResult(t, resultMap(t, updateResp))
	if updated["name"] != "New Name" {
		t.Errorf("name sau update = %v, want New Name", updated["name"])
	}
}

func TestSkillsDelete(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "skillsuser5", "pass1234")

	createResp := callRPC(t, addr, "test-secret-16chars", "skills.create", map[string]interface{}{
		"token": token, "name": "To Delete",
	})
	skill := skillFromResult(t, resultMap(t, createResp))
	id := skill["id"].(float64)

	delResp := callRPC(t, addr, "test-secret-16chars", "skills.delete", map[string]interface{}{
		"token": token, "id": int64(id),
	})
	if delResp.Error != nil {
		t.Fatalf("delete error: %v", delResp.Error)
	}
	m := resultMap(t, delResp)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %v", m["ok"])
	}
}
```

- [ ] **Step 1.3: Chạy test — verify FAIL**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -run "TestSkills" -v 2>&1 | tail -10
```

Expected: compile lỗi hoặc FAIL vì `skills.*` routes chưa có trong router.

- [ ] **Step 1.4: Tạo handlers_skills.go**

Tạo `go-engine/api/handlers_skills.go`:

```go
package api

import (
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// handleSkillsList trả về danh sách skills của user
func (r *Router) handleSkillsList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	list, err := r.skills.List(claims.UserID)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	type skillItem struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		PromptText string `json:"prompt_text"`
		Model      string `json:"model"`
		Provider   string `json:"provider"`
		Tags       string `json:"tags"`
	}
	items := make([]skillItem, 0, len(list))
	for _, s := range list {
		items = append(items, skillItem{
			ID: s.ID, Name: s.Name, PromptText: s.PromptText,
			Model: s.Model, Provider: s.Provider, Tags: s.Tags,
		})
	}
	return map[string]interface{}{"skills": items}, nil
}

// handleSkillsCreate tạo skill mới
func (r *Router) handleSkillsCreate(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Token      string `json:"token"`
		Name       string `json:"name"`
		PromptText string `json:"prompt_text"`
		Model      string `json:"model"`
		Provider   string `json:"provider"`
		Tags       string `json:"tags"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, copyErr(ErrInvalidParams)
	}
	skill, err := r.skills.Create(repos.CreateSkillInput{
		UserID: claims.UserID, Name: p.Name, PromptText: p.PromptText,
		Model: p.Model, Provider: p.Provider, Tags: p.Tags,
	})
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"skill": map[string]interface{}{
		"id": skill.ID, "name": skill.Name, "prompt_text": skill.PromptText,
		"model": skill.Model, "provider": skill.Provider, "tags": skill.Tags,
	}}, nil
}

// handleSkillsUpdate cập nhật skill
func (r *Router) handleSkillsUpdate(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Token      string `json:"token"`
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		PromptText string `json:"prompt_text"`
		Model      string `json:"model"`
		Provider   string `json:"provider"`
		Tags       string `json:"tags"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ID == 0 || p.Name == "" {
		return nil, copyErr(ErrInvalidParams)
	}
	if err := r.skills.Update(p.ID, repos.UpdateSkillInput{
		Name: p.Name, PromptText: p.PromptText,
		Model: p.Model, Provider: p.Provider, Tags: p.Tags,
	}); err != nil {
		return nil, copyErr(ErrInternal)
	}
	skill, err := r.skills.FindByID(p.ID)
	if err != nil || skill == nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"skill": map[string]interface{}{
		"id": skill.ID, "name": skill.Name, "prompt_text": skill.PromptText,
		"model": skill.Model, "provider": skill.Provider, "tags": skill.Tags,
	}}, nil
}

// handleSkillsDelete xóa skill
func (r *Router) handleSkillsDelete(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Token string `json:"token"`
		ID    int64  `json:"id"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ID == 0 {
		return nil, copyErr(ErrInvalidParams)
	}
	if err := r.skills.Delete(p.ID); err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"ok": true}, nil
}
```

- [ ] **Step 1.5: Cập nhật router.go — thêm skills field + routes**

Trong `go-engine/api/router.go`, thực hiện 3 thay đổi:

**1. Thêm field vào struct Router** (sau field `prompts`):
```go
skills        *repos.SkillRepo
```

**2. Trong `newRouter()`, sau dòng `prompts := repos.NewPromptRepo(s.db)`**:
```go
skills := repos.NewSkillRepo(s.db)
```

**3. Trong `return &Router{...}`, thêm**:
```go
skills: skills,
```

**4. Trong `dispatch()`, thêm 4 case trước `default`**:
```go
case "skills.list":
    return r.handleSkillsList(req)
case "skills.create":
    return r.handleSkillsCreate(req)
case "skills.update":
    return r.handleSkillsUpdate(req)
case "skills.delete":
    return r.handleSkillsDelete(req)
```

- [ ] **Step 1.6: Chạy test — verify PASS**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -run "TestSkills" -v 2>&1 | tail -15
```

Expected: tất cả TestSkills* PASS.

- [ ] **Step 1.7: Chạy toàn bộ test**

```bash
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: tất cả `ok`.

- [ ] **Step 1.8: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/db/repos/skill_repo.go go-engine/api/handlers_skills.go go-engine/api/handlers_skills_test.go go-engine/api/router.go
git commit -m "feat: thêm Skills API handlers (skills.list/create/update/delete)"
```

---

## Task 2: Go — History Repo + Analytics Handlers + Record Query

**Files:**
- Create: `go-engine/db/repos/history_repo.go`
- Create: `go-engine/api/handlers_analytics.go`
- Create: `go-engine/api/handlers_analytics_test.go`
- Modify: `go-engine/api/handlers_query.go` (full rewrite của `handleQueryStream`)
- Modify: `go-engine/api/router.go`

**Lưu ý:** Bảng `history` và `usage_daily` đã có sẵn trong `go-engine/db/migrations/001_init.sql` — không cần migration mới.

- [ ] **Step 2.1: Tạo history_repo.go**

Tạo `go-engine/db/repos/history_repo.go`:

```go
package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// HistoryRepo xử lý ghi và đọc history queries
type HistoryRepo struct {
	db *db.DB
}

// NewHistoryRepo tạo HistoryRepo mới
func NewHistoryRepo(database *db.DB) *HistoryRepo {
	return &HistoryRepo{db: database}
}

// InsertHistoryInput là dữ liệu ghi vào bảng history
type InsertHistoryInput struct {
	UserID    int64
	Query     string
	Response  string
	Provider  string
	Model     string
	LatencyMs int64
	Status    string // "success" | "error"
}

// Insert ghi một history record
func (r *HistoryRepo) Insert(input InsertHistoryInput) error {
	status := input.Status
	if status == "" {
		status = "success"
	}
	_, err := r.db.Exec(
		`INSERT INTO history (user_id, query, response, provider, model, latency_ms, status, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Query, input.Response, input.Provider, input.Model,
		input.LatencyMs, status, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

// DailySummary là kết quả aggregate theo ngày
type DailySummary struct {
	Date         string `json:"date"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Requests     int    `json:"requests"`
	Errors       int    `json:"errors"`
	AvgLatencyMs int    `json:"avg_latency_ms"`
}

// SummaryByPeriod trả về summary theo khoảng thời gian (số ngày)
func (r *HistoryRepo) SummaryByPeriod(userID int64, days int) ([]DailySummary, error) {
	rows, err := r.db.Query(
		`SELECT date(timestamp) as date, provider, model,
		        COUNT(*) as requests,
		        SUM(CASE WHEN status='error' THEN 1 ELSE 0 END) as errors,
		        CAST(AVG(latency_ms) AS INTEGER) as avg_latency_ms
		 FROM history
		 WHERE user_id = ? AND timestamp >= datetime('now', '-' || ? || ' days')
		 GROUP BY date(timestamp), provider, model
		 ORDER BY date DESC, requests DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("summary by period: %w", err)
	}
	defer rows.Close()

	var result []DailySummary
	for rows.Next() {
		var s DailySummary
		if err := rows.Scan(&s.Date, &s.Provider, &s.Model, &s.Requests, &s.Errors, &s.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ProviderTotals là tổng theo provider
type ProviderTotals struct {
	Provider    string  `json:"provider"`
	Requests    int     `json:"requests"`
	Errors      int     `json:"errors"`
	SuccessRate float64 `json:"success_rate"`
}

// TotalsByProvider trả về tổng requests/errors theo provider
func (r *HistoryRepo) TotalsByProvider(userID int64, days int) ([]ProviderTotals, error) {
	rows, err := r.db.Query(
		`SELECT provider,
		        COUNT(*) as requests,
		        SUM(CASE WHEN status='error' THEN 1 ELSE 0 END) as errors
		 FROM history
		 WHERE user_id = ? AND timestamp >= datetime('now', '-' || ? || ' days')
		 GROUP BY provider
		 ORDER BY requests DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("totals by provider: %w", err)
	}
	defer rows.Close()

	var result []ProviderTotals
	for rows.Next() {
		var p ProviderTotals
		if err := rows.Scan(&p.Provider, &p.Requests, &p.Errors); err != nil {
			return nil, fmt.Errorf("scan provider totals: %w", err)
		}
		if p.Requests > 0 {
			p.SuccessRate = float64(p.Requests-p.Errors) / float64(p.Requests) * 100
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
```

- [ ] **Step 2.2: Viết test analytics (failing)**

Tạo `go-engine/api/handlers_analytics_test.go`:

```go
package api_test

import (
	"testing"
)

func TestAnalyticsSummaryRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": "bad-token", "period": "7d",
	})
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Errorf("expected -32001, got %v", resp.Error)
	}
}

func TestAnalyticsSummaryEmpty(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": token, "period": "7d",
	})
	if resp.Error != nil {
		t.Fatalf("analytics.summary error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["summary"]; !exists {
		t.Error("phải có field 'summary'")
	}
}

func TestAnalyticsByProviderEmpty(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "analytics.by_provider", map[string]interface{}{
		"token": token, "period": "30d",
	})
	if resp.Error != nil {
		t.Fatalf("analytics.by_provider error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["providers"]; !exists {
		t.Error("phải có field 'providers'")
	}
}

func TestAnalyticsPeriodDefault(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser3", "pass1234")

	// Period không hợp lệ → dùng default 7d, không return error
	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": token, "period": "invalid",
	})
	if resp.Error != nil {
		t.Fatalf("period không hợp lệ phải dùng default, got error: %v", resp.Error)
	}
}
```

- [ ] **Step 2.3: Chạy test — verify FAIL**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -run "TestAnalytics" -v 2>&1 | tail -5
```

Expected: compile error hoặc FAIL — `analytics.summary` và `analytics.by_provider` chưa có trong router dispatch, `r.history` field chưa tồn tại trong Router struct.

- [ ] **Step 2.4: Tạo handlers_analytics.go**

Tạo `go-engine/api/handlers_analytics.go`:

```go
package api

import (
	"strconv"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// parsePeriodDays chuyển "7d", "30d", "90d" thành số ngày; mặc định 7
func parsePeriodDays(period string) int {
	switch period {
	case "30d":
		return 30
	case "90d":
		return 90
	default:
		return 7
	}
}

// handleAnalyticsSummary trả về daily summary theo period
func (r *Router) handleAnalyticsSummary(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Token  string `json:"token"`
		Period string `json:"period"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	days := parsePeriodDays(p.Period)
	summary, err := r.history.SummaryByPeriod(claims.UserID, days)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if summary == nil {
		summary = []repos.DailySummary{}
	}
	return map[string]interface{}{
		"summary": summary,
		"period":  strconv.Itoa(days) + "d",
	}, nil
}

// handleAnalyticsByProvider trả về totals theo provider
func (r *Router) handleAnalyticsByProvider(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Token  string `json:"token"`
		Period string `json:"period"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	days := parsePeriodDays(p.Period)
	providers, err := r.history.TotalsByProvider(claims.UserID, days)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if providers == nil {
		providers = []repos.ProviderTotals{}
	}
	return map[string]interface{}{
		"providers": providers,
		"period":    strconv.Itoa(days) + "d",
	}, nil
}
```

**Import đã được tích hợp sẵn trong code snippet trên.** Không cần thêm gì thêm.

- [ ] **Step 2.5: Cập nhật router.go — thêm history + analytics**

Trong `go-engine/api/router.go`:

**1. Thêm field vào struct Router** (sau `skills`):
```go
history *repos.HistoryRepo
```

**2. Trong `newRouter()`, sau `skills := ...`**:
```go
history := repos.NewHistoryRepo(s.db)
```

**3. Trong `return &Router{...}`**:
```go
history: history,
```

**4. Trong `dispatch()`, thêm 2 case trước `default`**:
```go
case "analytics.summary":
    return r.handleAnalyticsSummary(req)
case "analytics.by_provider":
    return r.handleAnalyticsByProvider(req)
```

- [ ] **Step 2.6: Chạy test — verify PASS**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -run "TestAnalytics" -v 2>&1 | tail -15
```

Expected: tất cả TestAnalytics* PASS.

- [ ] **Step 2.7: Rewrite handleQueryStream để ghi history**

Thay toàn bộ nội dung file `go-engine/api/handlers_query.go` bằng version dưới đây (bao gồm cả package declaration và imports — `"strings"` và `"repos"` là imports mới, `"time"` đã có sẵn):

```go
package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
	"github.com/minhtuancn/open-prompt/go-engine/model"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		Input     string            `json:"input"`
		Model     string            `json:"model"`
		System    string            `json:"system"`
		SlashName string            `json:"slash_name"`
		ExtraVars map[string]string `json:"extra_vars"`
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
		resolver := engine.NewCommandResolver(r.prompts, builder)
		resolved, resolveErr := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
		if resolveErr != nil {
			return nil, &RPCError{Code: -32002, Message: resolveErr.Error()}
		}
		if resolved.NeedsVars {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("slash command cần thêm biến: %v", resolved.RequiredVars)}
		}
		finalInput = resolved.RenderedPrompt
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

	// Tích lũy response và đo latency
	start := time.Now()
	var chunks []string

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	streamErr := modelRouter.Stream(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: finalInput,
		System: p.System,
	}, func(chunk string) {
		chunks = append(chunks, chunk)
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	latency := time.Since(start).Milliseconds()

	if streamErr != nil {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": "",
			"done":  true,
			"error": fmt.Sprintf("%v", streamErr),
		})
		// Ghi history lỗi (non-blocking, ignore error)
		_ = r.history.Insert(repos.InsertHistoryInput{
			UserID:    claims.UserID,
			Query:     finalInput,
			Provider:  "anthropic",
			Model:     modelName,
			LatencyMs: latency,
			Status:    "error",
		})
		return nil, nil
	}

	// Gửi notification done
	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	// Ghi history thành công (non-blocking, ignore error)
	_ = r.history.Insert(repos.InsertHistoryInput{
		UserID:    claims.UserID,
		Query:     finalInput,
		Response:  strings.Join(chunks, ""),
		Provider:  "anthropic",
		Model:     modelName,
		LatencyMs: latency,
		Status:    "success",
	})

	return nil, nil
}
```

- [ ] **Step 2.8: Chạy toàn bộ test**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: tất cả `ok`.

- [ ] **Step 2.9: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/db/repos/history_repo.go go-engine/api/handlers_analytics.go go-engine/api/handlers_analytics_test.go go-engine/api/handlers_query.go go-engine/api/router.go
git commit -m "feat: thêm history recording và Analytics API (analytics.summary/by_provider)"
```

---

## Task 3: React — settingsStore + i18n

**Files:**
- Create: `src/store/settingsStore.ts`
- Create: `src/i18n/locales/vi.json`
- Create: `src/i18n/locales/en.json`
- Create: `src/i18n/index.ts`
- Create: `src/hooks/useI18n.ts`

- [ ] **Step 3.1: Tạo settingsStore.ts**

Tạo `src/store/settingsStore.ts`:

```ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Locale = 'vi' | 'en'
type FontSize = 'sm' | 'base' | 'lg'

interface SettingsState {
  locale: Locale
  fontSize: FontSize
  setLocale: (locale: Locale) => void
  setFontSize: (size: FontSize) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      locale: 'vi',
      fontSize: 'base',
      setLocale: (locale) => set({ locale }),
      setFontSize: (fontSize) => set({ fontSize }),
    }),
    { name: 'open-prompt-settings' }
  )
)
```

- [ ] **Step 3.2: Tạo locale files**

Tạo `src/i18n/locales/vi.json`:

```json
{
  "overlay.placeholder": "Hỏi AI bất cứ điều gì... (/ để slash command)",
  "overlay.send": "Gửi",
  "overlay.streaming": "Đang xử lý...",
  "overlay.hint": "Enter gửi • Shift+Enter xuống dòng • / slash command",
  "overlay.insert": "Chèn vào app",
  "settings.title": "Cài đặt",
  "settings.providers": "Providers",
  "settings.skills": "Skills",
  "settings.hotkey": "Phím tắt",
  "settings.appearance": "Giao diện",
  "settings.language": "Ngôn ngữ",
  "settings.analytics": "Thống kê",
  "skills.new": "Tạo mới",
  "skills.edit": "Sửa",
  "skills.delete": "Xóa",
  "skills.save": "Lưu",
  "skills.cancel": "Hủy",
  "analytics.requests": "Yêu cầu",
  "analytics.errors": "Lỗi",
  "analytics.successRate": "Tỷ lệ thành công",
  "analytics.avgLatency": "Thời gian TB",
  "providers.connected": "Đã kết nối",
  "providers.disconnected": "Chưa kết nối",
  "providers.saveKey": "Lưu API Key",
  "language.vi": "Tiếng Việt",
  "language.en": "English"
}
```

Tạo `src/i18n/locales/en.json`:

```json
{
  "overlay.placeholder": "Ask AI anything... (/ for slash commands)",
  "overlay.send": "Send",
  "overlay.streaming": "Processing...",
  "overlay.hint": "Enter to send • Shift+Enter newline • / slash command",
  "overlay.insert": "Insert into app",
  "settings.title": "Settings",
  "settings.providers": "Providers",
  "settings.skills": "Skills",
  "settings.hotkey": "Hotkey",
  "settings.appearance": "Appearance",
  "settings.language": "Language",
  "settings.analytics": "Analytics",
  "skills.new": "New",
  "skills.edit": "Edit",
  "skills.delete": "Delete",
  "skills.save": "Save",
  "skills.cancel": "Cancel",
  "analytics.requests": "Requests",
  "analytics.errors": "Errors",
  "analytics.successRate": "Success rate",
  "analytics.avgLatency": "Avg latency",
  "providers.connected": "Connected",
  "providers.disconnected": "Not connected",
  "providers.saveKey": "Save API Key",
  "language.vi": "Tiếng Việt",
  "language.en": "English"
}
```

- [ ] **Step 3.3: Tạo i18n/index.ts**

Tạo `src/i18n/index.ts`:

```ts
import vi from './locales/vi.json'
import en from './locales/en.json'

type Locale = 'vi' | 'en'

const locales: Record<Locale, Record<string, string>> = { vi, en }

export function translate(locale: Locale, key: string, fallback = key): string {
  return locales[locale]?.[key] ?? locales.vi?.[key] ?? fallback
}
```

- [ ] **Step 3.4: Tạo hooks/useI18n.ts**

Tạo `src/hooks/useI18n.ts`:

```ts
import { useSettingsStore } from '../store/settingsStore'
import { translate } from '../i18n'

export function useI18n() {
  const locale = useSettingsStore((s) => s.locale)
  const t = (key: string, fallback?: string) => translate(locale, key, fallback)
  return { t, locale }
}
```

- [ ] **Step 3.5: Verify TypeScript build**

```bash
cd /home/dev/open-prompt-code/open-prompt
npx tsc --noEmit 2>&1 | head -20
```

Expected: không có lỗi.

- [ ] **Step 3.6: Commit**

```bash
git add src/store/settingsStore.ts src/i18n/ src/hooks/useI18n.ts
git commit -m "feat: thêm settingsStore, i18n (vi/en) và useI18n hook"
```

---

## Task 4: React — Skills UI

**Files:**
- Create: `src/components/skills/SkillList.tsx`
- Create: `src/components/skills/SkillEditor.tsx`

- [ ] **Step 4.1: Tạo SkillList.tsx**

Tạo `src/components/skills/SkillList.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'

interface Skill {
  id: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  onEdit: (skill: Skill) => void
  onNew: () => void
  refreshSignal: number
}

export function SkillList({ onEdit, onNew, refreshSignal }: Props) {
  const [skills, setSkills] = useState<Skill[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return
    setLoading(true)
    callEngine<{ skills: Skill[] }>('skills.list', { token })
      .then((res) => setSkills(res.skills ?? []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [refreshSignal])

  const handleDelete = async (id: number) => {
    const token = localStorage.getItem('auth_token')
    if (!token || !confirm('Xóa skill này?')) return
    await callEngine('skills.delete', { token, id })
    setSkills((prev) => prev.filter((s) => s.id !== id))
  }

  if (loading) {
    return <div className="text-white/40 text-sm text-center py-8">Đang tải...</div>
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-white/40">{skills.length} skill</span>
        <button
          onClick={onNew}
          className="text-xs px-3 py-1.5 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors"
        >
          + Tạo mới
        </button>
      </div>

      {skills.length === 0 && (
        <div className="text-white/30 text-sm text-center py-6">Chưa có skill. Tạo skill đầu tiên!</div>
      )}

      {skills.map((skill) => (
        <div
          key={skill.id}
          className="bg-white/5 border border-white/10 rounded-xl p-3 flex items-start justify-between gap-2"
        >
          <div className="flex-1 min-w-0">
            <div className="font-medium text-sm text-white truncate">{skill.name}</div>
            {skill.prompt_text && (
              <div className="text-xs text-white/40 mt-0.5 line-clamp-2">{skill.prompt_text}</div>
            )}
            <div className="flex gap-1.5 mt-1 flex-wrap">
              {skill.provider && (
                <span className="text-xs text-indigo-400/70 bg-indigo-500/10 px-1.5 py-0.5 rounded">
                  {skill.provider}
                </span>
              )}
              {skill.tags && skill.tags.split(',').map((tag) => tag.trim()).filter(Boolean).map((tag) => (
                <span key={tag} className="text-xs text-white/30 bg-white/5 px-1.5 py-0.5 rounded">{tag}</span>
              ))}
            </div>
          </div>
          <div className="flex gap-1 shrink-0">
            <button onClick={() => onEdit(skill)} className="text-xs px-2 py-1 text-white/50 hover:text-white transition-colors">Sửa</button>
            <button onClick={() => handleDelete(skill.id)} className="text-xs px-2 py-1 text-red-400/60 hover:text-red-400 transition-colors">Xóa</button>
          </div>
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 4.2: Tạo SkillEditor.tsx**

Tạo `src/components/skills/SkillEditor.tsx`:

```tsx
import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'

interface Skill {
  id?: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  skill?: Skill
  onSave: () => void
  onCancel: () => void
}

const PROVIDERS = ['anthropic', 'openai', 'ollama']
const MODELS: Record<string, string[]> = {
  anthropic: ['claude-3-5-sonnet-20241022', 'claude-3-haiku-20240307'],
  openai: ['gpt-4o', 'gpt-4o-mini'],
  ollama: ['llama3.2', 'mistral'],
}

export function SkillEditor({ skill, onSave, onCancel }: Props) {
  const [name, setName] = useState(skill?.name ?? '')
  const [promptText, setPromptText] = useState(skill?.prompt_text ?? '')
  const [provider, setProvider] = useState(skill?.provider ?? 'anthropic')
  const [model, setModel] = useState(skill?.model ?? '')
  const [tags, setTags] = useState(skill?.tags ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleSave = async () => {
    if (!name.trim()) { setError('Tên skill không được rỗng'); return }
    const token = localStorage.getItem('auth_token')
    if (!token) return
    setSaving(true)
    setError('')
    try {
      const payload = { token, name: name.trim(), prompt_text: promptText, provider, model, tags }
      if (skill?.id) {
        await callEngine('skills.update', { ...payload, id: skill.id })
      } else {
        await callEngine('skills.create', payload)
      }
      onSave()
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold text-white">{skill?.id ? 'Sửa skill' : 'Tạo skill mới'}</h3>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Tên skill *</label>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="VD: Dịch thuật..." className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
      </div>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Nội dung prompt</label>
        <textarea value={promptText} onChange={(e) => setPromptText(e.target.value)} placeholder="Bạn là trợ lý {{role}}..." rows={4} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 resize-none font-mono" />
        <p className="text-xs text-white/30 mt-1">Dùng {`{{variable}}`} để tạo biến động</p>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-white/50 mb-1 block">Provider</label>
          <select value={provider} onChange={(e) => { setProvider(e.target.value); setModel('') }} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white outline-none focus:border-indigo-500/50">
            {PROVIDERS.map((p) => <option key={p} value={p} className="bg-gray-900">{p}</option>)}
          </select>
        </div>
        <div>
          <label className="text-xs text-white/50 mb-1 block">Model</label>
          <select value={model} onChange={(e) => setModel(e.target.value)} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white outline-none focus:border-indigo-500/50">
            <option value="" className="bg-gray-900">Mặc định</option>
            {(MODELS[provider] ?? []).map((m) => <option key={m} value={m} className="bg-gray-900">{m}</option>)}
          </select>
        </div>
      </div>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Tags (phân cách bằng dấu phẩy)</label>
        <input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="dịch thuật, code, viết lách" className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
      </div>

      {error && <p className="text-xs text-red-400">{error}</p>}

      <div className="flex gap-2 justify-end">
        <button onClick={onCancel} className="text-sm px-4 py-2 text-white/50 hover:text-white transition-colors">Hủy</button>
        <button onClick={handleSave} disabled={saving} className="text-sm px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-50">
          {saving ? 'Đang lưu...' : 'Lưu'}
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 4.3: Verify TypeScript**

```bash
cd /home/dev/open-prompt-code/open-prompt
npx tsc --noEmit 2>&1 | head -20
```

Expected: không có lỗi.

- [ ] **Step 4.4: Commit**

```bash
git add src/components/skills/
git commit -m "feat: thêm SkillList và SkillEditor components"
```

---

## Task 5: React — Analytics Panel (tạo trước SettingsLayout)

**Files:**
- Create: `src/components/analytics/UsageStats.tsx`

**Lý do thứ tự:** `SettingsLayout` (Task 6) import `UsageStats`, nên Task 5 phải chạy trước.

- [ ] **Step 5.1: Tạo UsageStats.tsx**

Tạo `src/components/analytics/UsageStats.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'

type Period = '7d' | '30d' | '90d'

interface ProviderTotal {
  provider: string
  requests: number
  errors: number
  success_rate: number
}

interface DailySummary {
  date: string
  provider: string
  model: string
  requests: number
  errors: number
  avg_latency_ms: number
}

const PERIOD_LABELS: Record<Period, string> = { '7d': '7 ngày', '30d': '30 ngày', '90d': '90 ngày' }

export function UsageStats() {
  const [period, setPeriod] = useState<Period>('7d')
  const [providers, setProviders] = useState<ProviderTotal[]>([])
  const [summary, setSummary] = useState<DailySummary[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return
    setLoading(true)
    Promise.all([
      callEngine<{ providers: ProviderTotal[] }>('analytics.by_provider', { token, period }),
      callEngine<{ summary: DailySummary[] }>('analytics.summary', { token, period }),
    ])
      .then(([byProvider, bySummary]) => {
        setProviders(byProvider.providers ?? [])
        setSummary(bySummary.summary ?? [])
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [period])

  const totalRequests = providers.reduce((sum, p) => sum + p.requests, 0)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-1">
        {(['7d', '30d', '90d'] as Period[]).map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className={`text-xs px-3 py-1.5 rounded-lg transition-colors ${period === p ? 'bg-indigo-500/20 text-indigo-300' : 'text-white/40 hover:text-white/60'}`}
          >
            {PERIOD_LABELS[p]}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="text-white/30 text-sm text-center py-8">Đang tải...</div>
      ) : (
        <>
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <div className="text-xs text-white/40 mb-1">Tổng yêu cầu ({PERIOD_LABELS[period]})</div>
            <div className="text-3xl font-bold text-white">{totalRequests}</div>
          </div>

          {providers.length > 0 && (
            <div className="flex flex-col gap-2">
              <div className="text-xs text-white/40">Theo provider</div>
              {providers.map((p) => (
                <div key={p.provider} className="bg-white/5 border border-white/10 rounded-xl p-3 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-white capitalize">{p.provider}</div>
                    <div className="text-xs text-white/40 mt-0.5">{p.errors} lỗi • {p.success_rate.toFixed(1)}% thành công</div>
                  </div>
                  <div className="text-right">
                    <div className="text-lg font-bold text-white">{p.requests}</div>
                    <div className="text-xs text-white/30">yêu cầu</div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {summary.slice(0, 10).length > 0 && (
            <div className="flex flex-col gap-1">
              <div className="text-xs text-white/40 mb-1">Theo ngày</div>
              {summary.slice(0, 10).map((s, i) => (
                <div key={i} className="flex items-center justify-between py-1.5 border-b border-white/5 last:border-0">
                  <div className="text-xs text-white/50 shrink-0">{s.date}</div>
                  <div className="text-xs text-white/30 mx-2 flex-1 truncate">{s.provider}/{s.model}</div>
                  <div className="text-xs text-white font-medium shrink-0">{s.requests} req</div>
                  <div className="text-xs text-white/30 ml-2 shrink-0">{s.avg_latency_ms}ms</div>
                </div>
              ))}
            </div>
          )}

          {totalRequests === 0 && (
            <div className="text-white/30 text-sm text-center py-6">Chưa có dữ liệu trong {PERIOD_LABELS[period]} qua.</div>
          )}
        </>
      )}
    </div>
  )
}
```

- [ ] **Step 5.2: Verify TypeScript**

```bash
cd /home/dev/open-prompt-code/open-prompt
npx tsc --noEmit 2>&1 | head -20
```

Expected: không có lỗi.

- [ ] **Step 5.3: Commit**

```bash
git add src/components/analytics/
git commit -m "feat: thêm UsageStats analytics panel"
```

---

## Task 6: React — Settings Modal (6 tabs)

**Files:**
- Create: `src/components/settings/ProvidersTab.tsx`
- Create: `src/components/settings/HotkeyTab.tsx`
- Create: `src/components/settings/AppearanceTab.tsx`
- Create: `src/components/settings/LanguageTab.tsx`
- Create: `src/components/settings/SettingsLayout.tsx`

**Lưu ý:** `ProvidersTab` gọi `providers.connect` (không phải `providers.connect_oauth`) vì router hiện tại đăng ký route `"providers.connect"` tại `router.go:81`.

- [ ] **Step 6.1: Tạo ProvidersTab.tsx**

Tạo `src/components/settings/ProvidersTab.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'

interface Provider {
  id: string
  name: string
  auth_type: string
  connected: boolean
}

export function ProvidersTab() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState<Record<string, boolean>>({})
  const [saved, setSaved] = useState<Record<string, boolean>>({})

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return
    callEngine<Provider[]>('providers.list', { token })
      .then((list) => setProviders(list ?? []))
      .catch(console.error)
  }, [])

  const handleSaveKey = async (providerId: string) => {
    const token = localStorage.getItem('auth_token')
    if (!token || !apiKeys[providerId]) return
    setSaving((p) => ({ ...p, [providerId]: true }))
    try {
      await callEngine('providers.connect', { token, provider_id: providerId, api_key: apiKeys[providerId] })
      setSaved((p) => ({ ...p, [providerId]: true }))
      setProviders((prev) => prev.map((p) => p.id === providerId ? { ...p, connected: true } : p))
      setTimeout(() => setSaved((p) => ({ ...p, [providerId]: false })), 2000)
    } catch (e) {
      console.error(e)
    } finally {
      setSaving((p) => ({ ...p, [providerId]: false }))
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Nhập API key để kết nối AI provider.</p>
      {providers.map((provider) => (
        <div key={provider.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
          <div className="flex items-center justify-between mb-3">
            <div>
              <div className="text-sm font-medium text-white">{provider.name}</div>
              <div className="text-xs text-white/40">{provider.id}</div>
            </div>
            <span className={`text-xs px-2 py-0.5 rounded-full ${provider.connected ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/30'}`}>
              {provider.connected ? 'Đã kết nối' : 'Chưa kết nối'}
            </span>
          </div>
          {provider.auth_type === 'api_key' && (
            <div className="flex gap-2">
              <input
                type="password"
                placeholder={`${provider.name} API Key`}
                value={apiKeys[provider.id] ?? ''}
                onChange={(e) => setApiKeys((prev) => ({ ...prev, [provider.id]: e.target.value }))}
                className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono"
              />
              <button
                onClick={() => handleSaveKey(provider.id)}
                disabled={!apiKeys[provider.id] || saving[provider.id]}
                className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-40 shrink-0"
              >
                {saved[provider.id] ? '✓' : saving[provider.id] ? '...' : 'Lưu'}
              </button>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 6.2: Tạo HotkeyTab.tsx**

Tạo `src/components/settings/HotkeyTab.tsx`:

```tsx
export function HotkeyTab() {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Hotkey được cấu hình trong Tauri và áp dụng khi khởi động ứng dụng.</p>
      <div className="bg-white/5 border border-white/10 rounded-xl p-4">
        <div className="text-xs text-white/40 mb-2">Hotkey hiện tại</div>
        <div className="flex items-center gap-2">
          <kbd className="px-3 py-1.5 bg-white/10 border border-white/20 rounded-lg text-sm text-white font-mono">Ctrl</kbd>
          <span className="text-white/40">+</span>
          <kbd className="px-3 py-1.5 bg-white/10 border border-white/20 rounded-lg text-sm text-white font-mono">Space</kbd>
        </div>
        <p className="text-xs text-white/30 mt-3">Để thay đổi, sửa <code className="text-indigo-400">tauri.conf.json</code> và rebuild.</p>
      </div>
    </div>
  )
}
```

- [ ] **Step 6.3: Tạo AppearanceTab.tsx**

Tạo `src/components/settings/AppearanceTab.tsx`:

```tsx
import { useSettingsStore } from '../../store/settingsStore'

const FONT_SIZES = [
  { value: 'sm' as const, label: 'Nhỏ (13px)' },
  { value: 'base' as const, label: 'Vừa (14px)' },
  { value: 'lg' as const, label: 'Lớn (16px)' },
]

export function AppearanceTab() {
  const { fontSize, setFontSize } = useSettingsStore()

  return (
    <div className="flex flex-col gap-4">
      <div>
        <label className="text-xs text-white/50 mb-2 block">Cỡ chữ</label>
        <div className="flex gap-2">
          {FONT_SIZES.map((size) => (
            <button
              key={size.value}
              onClick={() => setFontSize(size.value)}
              className={`flex-1 text-xs py-2 px-3 rounded-lg border transition-colors ${fontSize === size.value ? 'border-indigo-500 bg-indigo-500/20 text-indigo-300' : 'border-white/10 bg-white/5 text-white/50 hover:bg-white/10'}`}
            >
              {size.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 6.4: Tạo LanguageTab.tsx**

Tạo `src/components/settings/LanguageTab.tsx`:

```tsx
import { useSettingsStore } from '../../store/settingsStore'

const LOCALES = [
  { value: 'vi' as const, label: 'Tiếng Việt', flag: '🇻🇳' },
  { value: 'en' as const, label: 'English', flag: '🇬🇧' },
]

export function LanguageTab() {
  const { locale, setLocale } = useSettingsStore()

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-white/40 mb-2">Chọn ngôn ngữ hiển thị của ứng dụng.</p>
      {LOCALES.map((l) => (
        <button
          key={l.value}
          onClick={() => setLocale(l.value)}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl border text-left transition-colors ${locale === l.value ? 'border-indigo-500 bg-indigo-500/15 text-white' : 'border-white/10 bg-white/5 text-white/60 hover:bg-white/10'}`}
        >
          <span className="text-xl">{l.flag}</span>
          <span className="text-sm font-medium">{l.label}</span>
          {locale === l.value && <span className="ml-auto text-indigo-400 text-xs">✓ Đang dùng</span>}
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 6.5: Tạo SettingsLayout.tsx**

Tạo `src/components/settings/SettingsLayout.tsx`:

```tsx
import { useState } from 'react'
import { ProvidersTab } from './ProvidersTab'
import { HotkeyTab } from './HotkeyTab'
import { AppearanceTab } from './AppearanceTab'
import { LanguageTab } from './LanguageTab'
import { SkillList } from '../skills/SkillList'
import { SkillEditor } from '../skills/SkillEditor'
import { UsageStats } from '../analytics/UsageStats'

type Tab = 'providers' | 'skills' | 'hotkey' | 'appearance' | 'language' | 'analytics'

interface SkillData {
  id?: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  onClose: () => void
}

const TABS: { id: Tab; label: string }[] = [
  { id: 'providers', label: 'Providers' },
  { id: 'skills', label: 'Skills' },
  { id: 'hotkey', label: 'Phím tắt' },
  { id: 'appearance', label: 'Giao diện' },
  { id: 'language', label: 'Ngôn ngữ' },
  { id: 'analytics', label: 'Thống kê' },
]

export function SettingsLayout({ onClose }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>('providers')
  const [editingSkill, setEditingSkill] = useState<SkillData | undefined>()
  const [isNewSkill, setIsNewSkill] = useState(false)
  const [skillRefresh, setSkillRefresh] = useState(0)

  const handleSkillSave = () => {
    setEditingSkill(undefined)
    setIsNewSkill(false)
    setSkillRefresh((n) => n + 1)
  }

  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab)
    setEditingSkill(undefined)
    setIsNewSkill(false)
  }

  return (
    <div className="flex flex-col" style={{ maxHeight: '600px' }}>
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-3 border-b border-white/10 shrink-0">
        <span className="text-sm font-semibold text-white">Cài đặt</span>
        <button onClick={onClose} className="text-white/40 hover:text-white transition-colors text-xl leading-none">×</button>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 px-4 py-2 border-b border-white/10 overflow-x-auto shrink-0">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => handleTabChange(tab.id)}
            className={`text-xs px-3 py-1.5 rounded-lg whitespace-nowrap transition-colors ${activeTab === tab.id ? 'bg-indigo-500/20 text-indigo-300' : 'text-white/40 hover:text-white/70'}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-5 py-4">
        {activeTab === 'providers' && <ProvidersTab />}
        {activeTab === 'skills' && (
          (editingSkill || isNewSkill) ? (
            <SkillEditor
              skill={editingSkill}
              onSave={handleSkillSave}
              onCancel={() => { setEditingSkill(undefined); setIsNewSkill(false) }}
            />
          ) : (
            <SkillList
              onEdit={(skill) => setEditingSkill(skill as SkillData)}
              onNew={() => setIsNewSkill(true)}
              refreshSignal={skillRefresh}
            />
          )
        )}
        {activeTab === 'hotkey' && <HotkeyTab />}
        {activeTab === 'appearance' && <AppearanceTab />}
        {activeTab === 'language' && <LanguageTab />}
        {activeTab === 'analytics' && <UsageStats />}
      </div>
    </div>
  )
}
```

- [ ] **Step 6.6: Verify TypeScript**

```bash
cd /home/dev/open-prompt-code/open-prompt
npx tsc --noEmit 2>&1 | head -20
```

Expected: không có lỗi TypeScript (UsageStats đã tồn tại từ Task 5).

- [ ] **Step 6.7: Commit**

```bash
git add src/components/settings/
git commit -m "feat: thêm Settings modal 6 tabs (Providers, Skills, Hotkey, Appearance, Language, Thống kê)"
```

---

## Task 7: React — Wire Settings vào App

**Files:**
- Modify: `src/App.tsx`

- [ ] **Step 7.1: Sửa App.tsx**

Mở `src/App.tsx`. Thực hiện 3 thay đổi:

**1. Thêm import** (sau các import hiện có):
```tsx
import { SettingsLayout } from './components/settings/SettingsLayout'
```

**2. Sửa type AppState** (thêm `'settings'`):
```tsx
type AppState = 'loading' | 'first-run' | 'login' | 'api-setup' | 'overlay' | 'settings'
```

**3. Thêm render cho settings** (trước `return` cuối cùng, tức là trước overlay return):
```tsx
if (state === 'settings') {
  return (
    <div className="bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden">
      <SettingsLayout onClose={() => setState('overlay')} />
    </div>
  )
}
```

**4. Thay thế overlay return** bằng version có gear icon:
```tsx
return (
  <div className="bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden min-h-16">
    <div className="flex items-start">
      <div className="flex-1 min-w-0">
        <CommandInput onSubmit={handleQuery} />
      </div>
      <button
        onClick={() => setState('settings')}
        title="Cài đặt"
        className="p-3 mt-2 mr-2 text-white/25 hover:text-white/60 transition-colors text-base shrink-0"
      >
        ⚙
      </button>
    </div>
    <ResponsePanel />
  </div>
)
```

- [ ] **Step 7.2: Verify TypeScript + build**

```bash
cd /home/dev/open-prompt-code/open-prompt
npx tsc --noEmit 2>&1 | head -20
npm run build 2>&1 | tail -10
```

Expected: không có lỗi TypeScript, build thành công.

- [ ] **Step 7.3: Commit**

```bash
git add src/App.tsx
git commit -m "feat: tích hợp Settings vào overlay — gear icon ⚙ mở Settings modal"
```

---

## Task 8: Kiểm tra cuối cùng

- [ ] **Step 8.1: Toàn bộ Go tests**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|--- PASS|--- FAIL)"
```

Expected: tất cả `ok`, không có `FAIL`.

- [ ] **Step 8.2: Go build**

```bash
go build ./...
```

Expected: không có lỗi.

- [ ] **Step 8.3: React build**

```bash
cd /home/dev/open-prompt-code/open-prompt
npm run build 2>&1 | tail -5
```

Expected: `✓ built in` không có error.

- [ ] **Step 8.4: Commit docs**

```bash
git add docs/superpowers/plans/2026-03-23-phase2-settings-skills-analytics.md
git commit -m "docs: thêm implementation plan Phase 2 Settings/Skills/Analytics"
```
