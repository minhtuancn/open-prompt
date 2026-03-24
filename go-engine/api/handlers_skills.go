package api

import (
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// assertSkillOwner verifica quyền sở hữu skill; trả về skill nếu hợp lệ
func (r *Router) assertSkillOwner(skillID, userID int64) (*repos.Skill, *RPCError) {
	skill, err := r.skills.FindByID(skillID)
	if err != nil || skill == nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if skill.UserID != userID {
		return nil, copyErr(ErrForbidden)
	}
	return skill, nil
}

// skillResponse chuyển đổi Skill thành map response
func skillResponse(s *repos.Skill) map[string]interface{} {
	return map[string]interface{}{
		"id": s.ID, "name": s.Name, "prompt_text": s.PromptText,
		"model": s.Model, "provider": s.Provider, "tags": s.Tags,
	}
}

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
		Name       string `json:"name"`
		PromptText string `json:"prompt_text"`
		Model      string `json:"model"`
		Provider   string `json:"provider"`
		Tags       string `json:"tags"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, copyErr(ErrInvalidParams)
	}
	p.Name = truncateString(p.Name, MaxNameLen)
	p.PromptText = truncateString(p.PromptText, MaxContentLen)
	p.Tags = truncateString(p.Tags, MaxTagsLen)
	skill, err := r.skills.Create(repos.CreateSkillInput{
		UserID: claims.UserID, Name: p.Name, PromptText: p.PromptText,
		Model: p.Model, Provider: p.Provider, Tags: p.Tags,
	})
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"skill": skillResponse(skill)}, nil
}

// handleSkillsUpdate cập nhật skill
func (r *Router) handleSkillsUpdate(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
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
	if _, rpcErr := r.assertSkillOwner(p.ID, claims.UserID); rpcErr != nil {
		return nil, rpcErr
	}
	if err := r.skills.Update(p.ID, repos.UpdateSkillInput{
		Name: p.Name, PromptText: p.PromptText,
		Model: p.Model, Provider: p.Provider, Tags: p.Tags,
	}); err != nil {
		return nil, copyErr(ErrInternal)
	}
	updated := &repos.Skill{
		ID: p.ID, Name: p.Name, PromptText: p.PromptText,
		Model: p.Model, Provider: p.Provider, Tags: p.Tags,
	}
	return map[string]interface{}{"skill": skillResponse(updated)}, nil
}

// handleSkillsDelete xóa skill
func (r *Router) handleSkillsDelete(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		ID int64 `json:"id"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ID == 0 {
		return nil, copyErr(ErrInvalidParams)
	}
	// Kiểm tra quyền sở hữu: chỉ chủ sở hữu mới được xóa skill
	if _, rpcErr := r.assertSkillOwner(p.ID, claims.UserID); rpcErr != nil {
		return nil, rpcErr
	}
	if err := r.skills.Delete(p.ID); err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]interface{}{"ok": true}, nil
}
