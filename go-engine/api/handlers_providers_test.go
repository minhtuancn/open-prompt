package api_test

import (
	"encoding/json"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/api"
)

// registerAndLogin đăng ký + đăng nhập, trả về JWT token
func registerAndLogin(t *testing.T, addr, username, password string) string {
	t.Helper()
	resp := callRPC(t, addr, "test-secret-16chars", "auth.register", map[string]string{
		"username": username,
		"password": password,
	})
	if resp.Error != nil {
		t.Fatalf("register failed: %v", resp.Error)
	}
	resp = callRPC(t, addr, "test-secret-16chars", "auth.login", map[string]string{
		"username": username,
		"password": password,
	})
	if resp.Error != nil {
		t.Fatalf("login failed: %v", resp.Error)
	}
	result := resp.Result.(map[string]interface{})
	return result["token"].(string)
}

// resultMap chuyển resp.Result sang map[string]interface{}
func resultMap(t *testing.T, resp api.Response) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return m
}

func TestProvidersListRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "providers.list", map[string]string{
		"token": "invalid-token",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("error code = %d, want -32001", resp.Error.Code)
	}
}

func TestProvidersList(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "user1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.list", map[string]string{
		"token": token,
	})
	if resp.Error != nil {
		t.Fatalf("providers.list error: %v", resp.Error)
	}

	// Result là array of provider objects
	data, _ := json.Marshal(resp.Result)
	var providers []map[string]interface{}
	if err := json.Unmarshal(data, &providers); err != nil {
		t.Fatalf("unmarshal providers: %v", err)
	}
	if len(providers) == 0 {
		t.Error("phải có ít nhất một provider")
	}

	// Kiểm tra các field bắt buộc
	found := false
	for _, p := range providers {
		if p["id"] == "anthropic" {
			found = true
			if p["name"] == "" {
				t.Error("anthropic provider phải có name")
			}
			if p["connected"] == nil {
				t.Error("anthropic provider phải có connected field")
			}
		}
	}
	if !found {
		t.Error("phải có anthropic trong danh sách providers")
	}
}

func TestProvidersDetectRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "providers.detect", map[string]string{
		"token": "",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi không có token")
	}
}

func TestProvidersDetect(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "user2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.detect", map[string]string{
		"token": token,
	})
	if resp.Error != nil {
		t.Fatalf("providers.detect error: %v", resp.Error)
	}
	// Kết quả là array (có thể rỗng nếu không có env vars)
	data, _ := json.Marshal(resp.Result)
	var results []interface{}
	if err := json.Unmarshal(data, &results); err != nil {
		// Có thể là nil nếu không phát hiện được gì — OK
		t.Logf("detect result (may be empty): %s", data)
	}
}

func TestProvidersConnectRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "providers.connect", map[string]string{
		"token":       "bad-token",
		"provider_id": "anthropic",
		"api_key":     "sk-ant-abc",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("error code = %d, want -32001", resp.Error.Code)
	}
}

func TestProvidersConnectMissingProviderID(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "user3", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.connect", map[string]string{
		"token":       token,
		"provider_id": "",
		"api_key":     "sk-ant-abc",
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi provider_id rỗng")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestProvidersConnectUnknownProvider(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "user4", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.connect", map[string]interface{}{
		"token":       token,
		"provider_id": "nonexistent-provider",
		"api_key":     "some-key",
	})
	// SaveToken phải fail khi provider không có trong registry
	if resp.Error == nil {
		t.Error("phải trả về error khi provider không tồn tại")
	}
}

func TestProvidersSetPriorityRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "providers.set_priority", map[string]interface{}{
		"token": "bad-token",
		"chain": []interface{}{},
	})
	if resp.Error == nil {
		t.Error("phải trả về error khi token không hợp lệ")
	}
}

func TestProvidersSetPriority(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "user5", "pass1234")

	chain := []map[string]interface{}{
		{"provider": "anthropic", "model": "claude-3-5-haiku-20241022", "is_enabled": true},
		{"provider": "openai", "model": "gpt-4o", "is_enabled": false},
	}
	resp := callRPC(t, addr, "test-secret-16chars", "providers.set_priority", map[string]interface{}{
		"token": token,
		"chain": chain,
	})
	if resp.Error != nil {
		t.Fatalf("providers.set_priority error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %v", m["ok"])
	}
}

func TestProvidersAddGateway(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "gwuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.add_gateway", map[string]interface{}{
		"token":         token,
		"name":          "my-ollama",
		"display_name":  "My Ollama",
		"base_url":      "http://localhost:11434/v1",
		"default_model": "llama3",
	})
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["ok"] != true {
		t.Errorf("ok=%v, want true", result["ok"])
	}
}

func TestProvidersAddGatewayMissingName(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "gwuser2", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.add_gateway", map[string]interface{}{
		"token":    token,
		"base_url": "http://localhost:11434/v1",
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Errorf("expected error -32602, got %v", resp.Error)
	}
}

func TestProvidersValidateNotFound(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "valuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.validate", map[string]interface{}{
		"token": token,
		"name":  "nonexistent",
	})
	if resp.Error == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestProvidersRemove(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "rmuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.remove", map[string]interface{}{
		"token": token,
		"name":  "anthropic",
	})
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["ok"] != true {
		t.Errorf("ok=%v, want true", result["ok"])
	}
}
