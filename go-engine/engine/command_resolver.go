package engine

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// ResolveResult kết quả sau khi resolve slash command
type ResolveResult struct {
	RenderedPrompt string            // prompt đã render (nếu không cần thêm vars)
	NeedsVars      bool              // true nếu template cần thêm biến từ user
	RequiredVars   []string          // danh sách biến còn thiếu
	RawTemplate    string            // template gốc (để render sau khi có vars)
	ExtraVars      map[string]string // biến đã cung cấp
}

// CommandResolver map slash_name → rendered prompt
type CommandResolver struct {
	prompts *repos.PromptRepo
	builder *PromptBuilder
}

// NewCommandResolver tạo CommandResolver mới
func NewCommandResolver(prompts *repos.PromptRepo, builder *PromptBuilder) *CommandResolver {
	return &CommandResolver{prompts: prompts, builder: builder}
}

// Resolve tìm slash command theo tên và render template
// extraVars: biến bổ sung ngoài "input" (có thể nil)
func (r *CommandResolver) Resolve(userID int64, slashName, input string, extraVars map[string]string) (*ResolveResult, error) {
	prompt, err := r.prompts.FindBySlashName(userID, slashName)
	if err != nil {
		return nil, fmt.Errorf("find slash command: %w", err)
	}
	if prompt == nil {
		return nil, fmt.Errorf("slash command %q không tìm thấy", slashName)
	}

	// Kiểm tra template có cần biến bổ sung không
	requiredVars := r.builder.ExtractVariables(prompt.Content)

	// Xây dựng vars map
	vars := map[string]string{
		"input": input,
	}
	for k, v := range extraVars {
		vars[k] = v
	}

	// Kiểm tra biến còn thiếu
	var missingVars []string
	for _, v := range requiredVars {
		if _, provided := vars[v]; !provided {
			missingVars = append(missingVars, v)
		}
	}

	if len(missingVars) > 0 {
		return &ResolveResult{
			NeedsVars:    true,
			RequiredVars: missingVars,
			RawTemplate:  prompt.Content,
			ExtraVars:    vars,
		}, nil
	}

	// Render template
	rendered, err := r.builder.Render(prompt.Content, vars)
	if err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}

	return &ResolveResult{
		RenderedPrompt: rendered,
		NeedsVars:      false,
		RawTemplate:    prompt.Content,
	}, nil
}
