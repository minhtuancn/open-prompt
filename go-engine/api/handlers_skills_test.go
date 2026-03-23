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
