package api

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

// requireAuth validate JWT token từ request params
func (r *Router) requireAuth(req *Request) (*auth.Claims, *RPCError) {
	var p struct {
		Token string `json:"token"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Token == "" {
		return nil, &RPCError{Code: -32001, Message: "token bắt buộc"}
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: "token không hợp lệ"}
	}
	return claims, nil
}

// handleProvidersList trả về danh sách providers và trạng thái của chúng
func (r *Router) handleProvidersList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	tokenRepo := repos.NewProviderTokenRepo(r.server.db)
	dbTokens, err := tokenRepo.GetByUser(claims.UserID)
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lỗi đọc DB: %v", err)}
	}

	connected := make(map[string]bool)
	for _, t := range dbTokens {
		connected[t.ProviderID] = true
	}

	reg := provider.DefaultRegistry()

	type ModelStatus struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		MaxTokens      int     `json:"max_tokens"`
		InputCostPer1K float64 `json:"input_cost_per_1k"`
		OutputCostPer1K float64 `json:"output_cost_per_1k"`
		SupportsStream bool    `json:"supports_stream"`
	}

	type ProviderStatus struct {
		ID        string        `json:"id"`
		Name      string        `json:"name"`
		AuthType  string        `json:"auth_type"`
		Connected bool          `json:"connected"`
		Models    []ModelStatus `json:"models"`
	}

	var result []ProviderStatus
	for _, p := range reg.List() {
		var models []ModelStatus
		for _, m := range p.Models {
			models = append(models, ModelStatus{
				ID:              m.ID,
				Name:            m.Name,
				MaxTokens:       m.MaxTokens,
				InputCostPer1K:  m.InputCostPer1K,
				OutputCostPer1K: m.OutputCostPer1K,
				SupportsStream:  m.SupportsStream,
			})
		}
		result = append(result, ProviderStatus{
			ID:        p.ID,
			Name:      p.Name,
			AuthType:  p.AuthType,
			Connected: connected[p.ID],
			Models:    models,
		})
	}

	return result, nil
}

// handleProvidersDetect chạy auto-detect và trả về kết quả
func (r *Router) handleProvidersDetect(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	detector := provider.NewDetector(provider.DetectorConfig{ScanFiles: true})
	results := detector.Detect()

	type DetectResult struct {
		ProviderID string `json:"provider_id"`
		Source     string `json:"source"`
		HasToken   bool   `json:"has_token"`
		FilePath   string `json:"file_path,omitempty"`
	}

	var out []DetectResult
	for _, res := range results {
		out = append(out, DetectResult{
			ProviderID: res.ProviderID,
			Source:     res.Source,
			HasToken:   res.Token != "",
			FilePath:   res.FilePath,
		})
	}

	return out, nil
}

// handleProvidersConnect lưu API key thủ công
func (r *Router) handleProvidersConnect(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var params struct {
		Token      string `json:"token"`
		ProviderID string `json:"provider_id"`
		APIKey     string `json:"api_key"`
	}
	if err := decodeParams(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "params không hợp lệ"}
	}
	if params.ProviderID == "" {
		return nil, &RPCError{Code: -32602, Message: "provider_id không được để trống"}
	}

	kc := provider.NewKeychain("open-prompt")
	tokenRepo := repos.NewProviderTokenRepo(r.server.db)
	reg := provider.DefaultRegistry()
	tm := provider.NewTokenManager(kc, tokenRepo, reg)

	if err := tm.SaveToken(claims.UserID, params.ProviderID, params.APIKey); err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lưu token thất bại: %v", err)}
	}

	return map[string]interface{}{"ok": true, "provider_id": params.ProviderID}, nil
}

// handleProvidersSetPriority cập nhật model priority chain
func (r *Router) handleProvidersSetPriority(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var params struct {
		Token string `json:"token"`
		Chain []struct {
			Provider  string `json:"provider"`
			Model     string `json:"model"`
			IsEnabled bool   `json:"is_enabled"`
		} `json:"chain"`
	}
	if err := decodeParams(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "params không hợp lệ"}
	}

	prioRepo := repos.NewModelPriorityRepo(r.server.db)

	var chain []repos.ModelPriority
	for _, item := range params.Chain {
		chain = append(chain, repos.ModelPriority{
			UserID:    claims.UserID,
			Provider:  item.Provider,
			Model:     item.Model,
			IsEnabled: item.IsEnabled,
		})
	}

	if err := prioRepo.SetChain(claims.UserID, chain); err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lỗi cập nhật priority: %v", err)}
	}

	return map[string]interface{}{"ok": true}, nil
}
