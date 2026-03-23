package engine

import (
	"bytes"
	"regexp"
	"text/template"
)

// varPattern trích xuất biến {{.name}} từ template
var varPattern = regexp.MustCompile(`\{\{\.(\w+)\}\}`)

// PromptBuilder render Go text/template với các biến
type PromptBuilder struct{}

// NewPromptBuilder tạo PromptBuilder mới
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// Render render template với map biến
// Trả về chuỗi đã render hoặc error nếu template sai cú pháp
func (pb *PromptBuilder) Render(tmpl string, vars map[string]string) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}

	// Chuyển map[string]string sang map[string]interface{} để template xử lý
	// Hỗ trợ nested access {{.context.app}} bằng cách tạo context map
	data := make(map[string]interface{})
	for k, v := range vars {
		data[k] = v
	}

	// Tạo context object nếu chưa có
	if _, ok := data["context"]; !ok {
		data["context"] = map[string]string{
			"app":   "",
			"title": "",
		}
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExtractVariables trích xuất danh sách biến từ template
// Bỏ qua "input" và các biến bắt đầu bằng "context."
func (pb *PromptBuilder) ExtractVariables(tmpl string) []string {
	matches := varPattern.FindAllStringSubmatch(tmpl, -1)

	seen := make(map[string]bool)
	var result []string

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		// Bỏ qua "input" và "context"
		if name == "input" || name == "context" {
			continue
		}
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}
