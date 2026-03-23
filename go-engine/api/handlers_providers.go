package api

import (
	"context"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

// requireAuth validate JWT token từ request params
func (r *Router) requireAuth(req *Request) (*auth.Claims, *RPCError) {
	var p struct {
		Token string `json:"token"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Token == "" {
		return nil, &RPCError{Code: ErrUnauthorized.Code, Message: "token bắt buộc"}
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, &RPCError{Code: ErrUnauthorized.Code, Message: "token không hợp lệ"}
	}
	return claims, nil
}

// handleProvidersList trả về danh sách providers và trạng thái của chúng
func (r *Router) handleProvidersList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	dbTokens, err := r.tokenRepo.GetByUser(claims.UserID)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("lỗi đọc DB: %v", err)}
	}

	connected := make(map[string]bool)
	for _, t := range dbTokens {
		connected[t.ProviderID] = true
	}

	reg := r.registry

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
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "params không hợp lệ"}
	}
	if params.ProviderID == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "provider_id không được để trống"}
	}

	if err := r.tokenManager.SaveToken(claims.UserID, params.ProviderID, params.APIKey); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("lưu token thất bại: %v", err)}
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
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "params không hợp lệ"}
	}

	var chain []repos.ModelPriority
	for _, item := range params.Chain {
		chain = append(chain, repos.ModelPriority{
			UserID:    claims.UserID,
			Provider:  item.Provider,
			Model:     item.Model,
			IsEnabled: item.IsEnabled,
		})
	}

	if err := r.priorityRepo.SetChain(claims.UserID, chain); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("lỗi cập nhật priority: %v", err)}
	}

	return map[string]interface{}{"ok": true}, nil
}

// handleProvidersAddGateway thêm custom gateway
func (r *Router) handleProvidersAddGateway(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token        string   `json:"token"`
		Name         string   `json:"name"`
		DisplayName  string   `json:"display_name"`
		BaseURL      string   `json:"base_url"`
		APIKey       string   `json:"api_key"`
		DefaultModel string   `json:"default_model"`
		Aliases      []string `json:"aliases"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" || p.BaseURL == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name và base_url bắt buộc"}
	}

	displayName := p.DisplayName
	if displayName == "" {
		displayName = p.Name
	}

	gw := providers.NewGatewayProvider(p.Name, displayName, p.BaseURL, p.APIKey, p.DefaultModel, p.Aliases)
	r.providerRegistry.Register(gw)

	_, err := r.server.db.Exec(
		`INSERT INTO custom_gateways (user_id, name, display_name, base_url, api_key, default_model, aliases)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		claims.UserID, p.Name, displayName, p.BaseURL, p.APIKey, p.DefaultModel, "[]",
	)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("save gateway: %v", err)}
	}

	return map[string]interface{}{"ok": true, "name": p.Name}, nil
}

// handleProvidersValidate kiểm tra provider có hoạt động không
func (r *Router) handleProvidersValidate(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name bắt buộc"}
	}

	prov, err := r.providerRegistry.Route(p.Name)
	if err != nil {
		return nil, &RPCError{Code: ErrProviderNotFound.Code, Message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	validateErr := prov.Validate(ctx)
	latency := time.Since(start).Milliseconds()

	result := map[string]interface{}{
		"name":       prov.Name(),
		"valid":      validateErr == nil,
		"latency_ms": latency,
	}
	if validateErr != nil {
		result["error"] = validateErr.Error()
	}
	if validateErr == nil {
		if models, mErr := prov.Models(ctx); mErr == nil {
			result["models"] = models
		}
	}

	return result, nil
}

// handleProvidersRemove xóa provider
func (r *Router) handleProvidersRemove(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name bắt buộc"}
	}

	_ = r.tokenRepo.Delete(claims.UserID, p.Name)
	_, _ = r.server.db.Exec("DELETE FROM custom_gateways WHERE user_id = ? AND name = ?", claims.UserID, p.Name)

	return map[string]interface{}{"ok": true}, nil
}
