package api

import (
	"regexp"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
)

// slashNameRegex: chỉ cho phép a-z, 0-9, -, _ và tối đa 32 ký tự
var slashNameRegex = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// handlePromptsList lấy danh sách prompts của user
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
	if p.IsSlash && !slashNameRegex.MatchString(p.SlashName) {
		return nil, &RPCError{Code: -32602, Message: "slash_name không hợp lệ: chỉ a-z 0-9 - _ và tối đa 32 ký tự"}
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
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.ID == 0 || p.Title == "" || p.Content == "" {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.IsSlash && !slashNameRegex.MatchString(p.SlashName) {
		return nil, &RPCError{Code: -32602, Message: "slash_name không hợp lệ"}
	}

	// Kiểm tra ownership — chống IDOR
	existing, err := r.prompts.FindByID(p.ID)
	if err != nil || existing == nil {
		return nil, &RPCError{Code: -32602, Message: "prompt không tồn tại"}
	}
	if existing.UserID != claims.UserID {
		return nil, &RPCError{Code: -32001, Message: "không có quyền truy cập prompt này"}
	}

	// Update trả về error, sau đó lấy prompt mới nhất bằng FindByID
	if err := r.prompts.Update(p.ID, repos.UpdatePromptInput{
		Title:     p.Title,
		Content:   p.Content,
		Category:  p.Category,
		Tags:      p.Tags,
		IsSlash:   p.IsSlash,
		SlashName: p.SlashName,
	}); err != nil {
		return nil, copyErr(ErrInternal)
	}
	prompt, err := r.prompts.FindByID(p.ID)
	if err != nil || prompt == nil {
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
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if p.ID == 0 {
		return nil, copyErr(ErrInvalidParams)
	}

	// Kiểm tra ownership — chống IDOR
	existing, err := r.prompts.FindByID(p.ID)
	if err != nil || existing == nil {
		return nil, &RPCError{Code: -32602, Message: "prompt không tồn tại"}
	}
	if existing.UserID != claims.UserID {
		return nil, &RPCError{Code: -32001, Message: "không có quyền xóa prompt này"}
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

	builder := engine.NewPromptBuilder()
	resolver := engine.NewCommandResolver(r.prompts, builder)

	result, err := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
	if err != nil {
		return nil, &RPCError{Code: -32002, Message: err.Error()}
	}
	if result.NeedsVars {
		return map[string]interface{}{
			"needs_vars":    true,
			"required_vars": result.RequiredVars,
			"raw_template":  result.RawTemplate,
		}, nil
	}
	return map[string]interface{}{"rendered": result.RenderedPrompt}, nil
}
