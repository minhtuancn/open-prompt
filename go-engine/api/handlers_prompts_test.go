package api_test

import (
	"encoding/json"
	"testing"
)

func promptFromResult(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(m["prompt"])
	if err != nil {
		t.Fatalf("marshal prompt: %v", err)
	}
	var p map[string]interface{}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal prompt: %v", err)
	}
	return p
}

func TestPromptsListRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.list", map[string]string{
		"token": "bad-token",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("error code = %d, want -32001", resp.Error.Code)
	}
}

func TestPromptsList(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "prompts.list", map[string]string{
		"token": token,
	})
	if resp.Error != nil {
		t.Fatalf("prompts.list error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	// Danh sách rỗng → "prompts" field tồn tại nhưng có thể là null
	if _, exists := m["prompts"]; !exists {
		t.Error("phải có field 'prompts' trong response")
	}
}

func TestPromptsCreate(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":   token,
		"title":   "Test Prompt",
		"content": "Trả lời câu hỏi sau: {{.question}}",
	})
	if resp.Error != nil {
		t.Fatalf("prompts.create error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	promptData, ok := m["prompt"].(map[string]interface{})
	if !ok {
		t.Fatalf("phải có field 'prompt' trong response, got %T", m["prompt"])
	}
	// Prompt struct không có json tags → field names viết hoa
	if promptData["Title"] != "Test Prompt" {
		t.Errorf("Title = %q, want %q", promptData["Title"], "Test Prompt")
	}
	if promptData["ID"] == nil {
		t.Error("prompt phải có ID")
	}
}

func TestPromptsCreateValidation(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts3", "pass1234")

	// Thiếu title
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":   token,
		"title":   "",
		"content": "some content",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi title rỗng")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestPromptsCreateSlashValidation(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts4", "pass1234")

	// is_slash=true nhưng slash_name không hợp lệ (có chữ hoa)
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":      token,
		"title":      "My Command",
		"content":    "do something",
		"is_slash":   true,
		"slash_name": "MyCommand",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi slash_name không hợp lệ")
	}
}

func TestPromptsUpdate(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts5", "pass1234")

	// Tạo prompt
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":   token,
		"title":   "Original",
		"content": "original content",
	})
	if resp.Error != nil {
		t.Fatalf("create error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	// Prompt struct không có json tags → field names viết hoa
	created := promptFromResult(t, m)
	promptID := created["ID"].(float64)

	// Cập nhật prompt
	resp = callRPC(t, addr, "test-secret-16chars", "prompts.update", map[string]interface{}{
		"token":   token,
		"id":      int64(promptID),
		"title":   "Updated",
		"content": "updated content",
	})
	if resp.Error != nil {
		t.Fatalf("update error: %v", resp.Error)
	}
	m = resultMap(t, resp)
	updated := promptFromResult(t, m)
	if updated["Title"] != "Updated" {
		t.Errorf("Title sau update = %q, want %q", updated["Title"], "Updated")
	}
}

func TestPromptsDelete(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "prompts6", "pass1234")

	// Tạo prompt
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":   token,
		"title":   "To delete",
		"content": "will be deleted",
	})
	if resp.Error != nil {
		t.Fatalf("create error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	created := promptFromResult(t, m)
	promptID := created["ID"].(float64)

	// Xoá prompt
	resp = callRPC(t, addr, "test-secret-16chars", "prompts.delete", map[string]interface{}{
		"token": token,
		"id":    int64(promptID),
	})
	if resp.Error != nil {
		t.Fatalf("delete error: %v", resp.Error)
	}
	m = resultMap(t, resp)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %v", m["ok"])
	}

	// Kiểm tra list trống
	resp = callRPC(t, addr, "test-secret-16chars", "prompts.list", map[string]string{
		"token": token,
	})
	if resp.Error != nil {
		t.Fatalf("list error: %v", resp.Error)
	}
	m = resultMap(t, resp)
	data, err := json.Marshal(m["prompts"])
	if err != nil {
		t.Fatalf("marshal prompts: %v", err)
	}
	var prompts []interface{}
	if err := json.Unmarshal(data, &prompts); err != nil {
		t.Fatalf("unmarshal prompts: %v", err)
	}
	if len(prompts) != 0 {
		t.Errorf("len(prompts) = %d, want 0 sau khi xoá", len(prompts))
	}
}

func TestCommandsListRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "commands.list", map[string]string{
		"token": "bad-token",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
}

func TestCommandsList(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "cmds1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "commands.list", map[string]string{
		"token": token,
	})
	if resp.Error != nil {
		t.Fatalf("commands.list error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["commands"]; !exists {
		t.Error("phải có field 'commands' trong response")
	}
}

func TestCommandsResolveRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "commands.resolve", map[string]interface{}{
		"token":      "bad-token",
		"slash_name": "email",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
}

func TestCommandsResolveMissingSlashName(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "cmds2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "commands.resolve", map[string]interface{}{
		"token":      token,
		"slash_name": "",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi slash_name rỗng")
	}
}

func TestCommandsResolveNotFound(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "cmds3", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "commands.resolve", map[string]interface{}{
		"token":      token,
		"slash_name": "nonexistent-command",
		"input":      "some input",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi command không tồn tại")
	}
}

func TestCommandsResolveWithSlashPrompt(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "cmds4", "pass1234")

	// Tạo slash command
	resp := callRPC(t, addr, "test-secret-16chars", "prompts.create", map[string]interface{}{
		"token":      token,
		"title":      "Email template",
		"content":    "Viết email {{.input}}",
		"is_slash":   true,
		"slash_name": "myemail",
	})
	if resp.Error != nil {
		t.Fatalf("create slash command error: %v", resp.Error)
	}

	// Resolve command với input
	resp = callRPC(t, addr, "test-secret-16chars", "commands.resolve", map[string]interface{}{
		"token":      token,
		"slash_name": "myemail",
		"input":      "chuyên nghiệp",
	})
	if resp.Error != nil {
		t.Fatalf("commands.resolve error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if m["rendered"] == nil {
		t.Error("phải có field 'rendered' trong response")
	}
	if m["rendered"].(string) == "" {
		t.Error("rendered prompt không được rỗng")
	}
}
