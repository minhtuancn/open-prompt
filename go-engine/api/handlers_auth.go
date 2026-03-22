package api

import (
	"encoding/json"
	"fmt"
)

type registerParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *Router) handleRegister(req *Request) (interface{}, *RPCError) {
	var p registerParams
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	user, err := r.auth.Register(p.Username, p.Password)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: err.Error()}
	}
	return map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	}, nil
}

func (r *Router) handleLogin(req *Request) (interface{}, *RPCError) {
	var p loginParams
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	token, err := r.auth.Login(p.Username, p.Password)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: err.Error()}
	}
	return map[string]string{"token": token}, nil
}

func (r *Router) handleMe(req *Request) (interface{}, *RPCError) {
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
	return map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
	}, nil
}

func (r *Router) handleIsFirstRun(req *Request) (interface{}, *RPCError) {
	isFirst, err := r.auth.IsFirstRun()
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]bool{"is_first_run": isFirst}, nil
}

func (r *Router) handleSettingsGet(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
		Key   string `json:"key"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	value, err := r.settings.Get(claims.UserID, p.Key)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]string{"value": value}, nil
}

func (r *Router) handleSettingsSet(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}
	if err := r.settings.Set(claims.UserID, p.Key, p.Value); err != nil {
		return nil, copyErr(ErrInternal)
	}
	return map[string]bool{"ok": true}, nil
}

// decodeParams decode params từ interface{} sang struct
func decodeParams(params interface{}, dst interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	return json.Unmarshal(data, dst)
}
