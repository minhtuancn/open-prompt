package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// handleOAuthStart khởi tạo OAuth flow cho provider
func (r *Router) handleOAuthStart(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token    string `json:"token"`
		Provider string `json:"provider"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Provider == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "provider bắt buộc"}
	}

	switch p.Provider {
	case "copilot", "github":
		// GitHub Copilot dùng Device Flow — không cần WebView
		return map[string]interface{}{
			"method":           "device_flow",
			"verification_uri": "https://github.com/login/device",
			"user_code":        "XXXX-XXXX", // placeholder — cần GitHub OAuth App thật
			"device_code":      "placeholder",
			"message":          "Mở github.com/login/device và nhập code hiển thị. Tính năng này cần GitHub OAuth App ID thật để hoạt động.",
		}, nil

	case "gemini", "google":
		// Gemini dùng OAuth2 PKCE — tạo authorization URL
		verifier, challenge, pkceErr := generatePKCE()
		if pkceErr != nil {
			return nil, &RPCError{Code: ErrInternal.Code, Message: "không thể tạo PKCE challenge"}
		}
		// Placeholder — cần Google Cloud Client ID thật
		return map[string]interface{}{
			"method":         "webview",
			"url":            fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=PLACEHOLDER&redirect_uri=open-prompt://oauth&response_type=code&scope=https://www.googleapis.com/auth/generative-language&code_challenge=%s&code_challenge_method=S256", challenge),
			"code_verifier":  verifier,
			"message":        "Tính năng này cần Google Cloud Client ID thật để hoạt động.",
		}, nil

	default:
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: fmt.Sprintf("provider %q không hỗ trợ OAuth", p.Provider)}
	}
}

// handleOAuthFinish hoàn tất OAuth — exchange code lấy token
func (r *Router) handleOAuthFinish(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token        string `json:"token"`
		Provider     string `json:"provider"`
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Provider == "" || p.Code == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "provider và code bắt buộc"}
	}

	// Placeholder — exchange code → access_token cần Client ID/Secret thật
	_ = claims
	return map[string]interface{}{
		"ok":      false,
		"message": "OAuth chưa được cấu hình — cần Client ID thật",
	}, nil
}

// handleOAuthPoll polling Device Flow cho GitHub Copilot
func (r *Router) handleOAuthPoll(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token      string `json:"token"`
		Provider   string `json:"provider"`
		DeviceCode string `json:"device_code"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	// Placeholder — poll GitHub Device Flow cần OAuth App thật
	return map[string]interface{}{
		"done":    false,
		"message": "Đang chờ user xác nhận trên GitHub...",
	}, nil
}

// generatePKCE tạo code_verifier và code_challenge cho OAuth2 PKCE (RFC 7636)
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("PKCE generate failed: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}
