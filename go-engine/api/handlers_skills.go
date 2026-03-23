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
	claims, rpcErr := r.requireAuth(req)
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
	// Kiểm tra quyền sở hữu: chỉ chủ sở hữu mới được cập nhật skill
	existing, err := r.skills.FindByID(p.ID)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if existing == nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if existing.UserID != claims.UserID {
		return nil, copyErr(ErrForbidden)
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
	claims, rpcErr := r.requireAuth(req)
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
	// Kiểm tra quyền sở hữu: chỉ chủ sở hữu mới được xóa skill
	existing, err := r.skills.FindByID(p.ID)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if existing == nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if existing.UserID != claims.UserID {
		return nil, copyErr(ErrForbidden)
	}
	if err := r.skills.Delete(p.ID); err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"ok": true}, nil
}
