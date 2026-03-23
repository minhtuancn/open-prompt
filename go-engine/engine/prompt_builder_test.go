package engine_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

func TestPromptBuilder_RenderSimple(t *testing.T) {
	pb := engine.NewPromptBuilder()

	result, err := pb.Render("Viết email về {{.input}}", map[string]string{
		"input": "meeting tomorrow",
	})
	if err != nil {
		t.Fatalf("Render thất bại: %v", err)
	}
	if result != "Viết email về meeting tomorrow" {
		t.Errorf("Render = %q, muốn %q", result, "Viết email về meeting tomorrow")
	}
}

func TestPromptBuilder_RenderMultiVar(t *testing.T) {
	pb := engine.NewPromptBuilder()

	result, err := pb.Render("Dịch {{.input}} sang {{.lang}}", map[string]string{
		"input": "Hello world",
		"lang":  "Vietnamese",
	})
	if err != nil {
		t.Fatalf("Render thất bại: %v", err)
	}
	expected := "Dịch Hello world sang Vietnamese"
	if result != expected {
		t.Errorf("Render = %q, muốn %q", result, expected)
	}
}

func TestPromptBuilder_ExtractVariables(t *testing.T) {
	pb := engine.NewPromptBuilder()

	// Phải trích xuất biến ngoài "input" và "context.*"
	vars := pb.ExtractVariables("Dịch {{.input}} sang {{.lang}} theo phong cách {{.style}}")
	if len(vars) != 2 {
		t.Fatalf("ExtractVariables = %v (len=%d), muốn 2 vars", vars, len(vars))
	}
	// Phải có "lang" và "style", không có "input"
	varSet := make(map[string]bool)
	for _, v := range vars {
		varSet[v] = true
	}
	if varSet["input"] {
		t.Error("input không được trong danh sách extra vars")
	}
	if !varSet["lang"] || !varSet["style"] {
		t.Errorf("Thiếu lang hoặc style trong %v", vars)
	}
}

func TestPromptBuilder_ExtractVariables_IgnoreContext(t *testing.T) {
	pb := engine.NewPromptBuilder()
	vars := pb.ExtractVariables("App: {{.context.app}} — Input: {{.input}} — Lang: {{.lang}}")
	if len(vars) != 1 || vars[0] != "lang" {
		t.Errorf("ExtractVariables = %v, muốn [lang]", vars)
	}
}

func TestPromptBuilder_RenderError(t *testing.T) {
	pb := engine.NewPromptBuilder()
	// Template sai cú pháp
	_, err := pb.Render("{{.input", map[string]string{"input": "test"})
	if err == nil {
		t.Error("Template sai cú pháp phải trả về error")
	}
}
