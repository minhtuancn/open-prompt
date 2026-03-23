package api

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// handlePluginsList trả về danh sách plugins
func (r *Router) handlePluginsList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	plugins, err := r.plugins.List(claims.UserID)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("list: %v", err)}
	}
	if plugins == nil {
		plugins = []repos.Plugin{}
	}
	return plugins, nil
}

// handlePluginsInstall cài plugin mới
func (r *Router) handlePluginsInstall(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token      string `json:"token"`
		Name       string `json:"name"`
		Version    string `json:"version"`
		Type       string `json:"type"`
		ConfigJSON string `json:"config_json"`
		Source     string `json:"source"`
		SourcePath string `json:"source_path"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" || p.Type == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name và type bắt buộc"}
	}
	if p.Version == "" {
		p.Version = "1.0.0"
	}
	if p.ConfigJSON == "" {
		p.ConfigJSON = "{}"
	}

	id, err := r.plugins.Install(claims.UserID, p.Name, p.Version, p.Type, p.ConfigJSON, p.Source, p.SourcePath)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("install: %v", err)}
	}
	return map[string]interface{}{"id": id, "name": p.Name}, nil
}

// handlePluginsToggle bật/tắt plugin
func (r *Router) handlePluginsToggle(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token   string `json:"token"`
		ID      int64  `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ID == 0 {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "id bắt buộc"}
	}

	if err := r.plugins.Toggle(p.ID, claims.UserID, p.Enabled); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: err.Error()}
	}
	return map[string]bool{"ok": true}, nil
}

// handlePluginsUninstall xóa plugin
func (r *Router) handlePluginsUninstall(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		ID    int64  `json:"id"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ID == 0 {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "id bắt buộc"}
	}

	if err := r.plugins.Uninstall(p.ID, claims.UserID); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: err.Error()}
	}
	return map[string]bool{"ok": true}, nil
}
