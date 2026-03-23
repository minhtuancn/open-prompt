package provider

import (
	"fmt"
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// TokenManager quản lý vòng đời của API tokens
type TokenManager struct {
	keychain  *Keychain
	tokenRepo *repos.ProviderTokenRepo
	registry  *Registry
}

// NewTokenManager tạo token manager (cho phép nil deps để test)
func NewTokenManager(kc *Keychain, repo *repos.ProviderTokenRepo, reg *Registry) *TokenManager {
	return &TokenManager{
		keychain:  kc,
		tokenRepo: repo,
		registry:  reg,
	}
}

// ValidateKeyFormat kiểm tra format API key trước khi lưu
func (tm *TokenManager) ValidateKeyFormat(providerID, key string) error {
	if providerID == "ollama" {
		return nil
	}
	if key == "" {
		return fmt.Errorf("API key không được để trống")
	}
	switch providerID {
	case "anthropic":
		if !strings.HasPrefix(key, "sk-ant-") {
			return fmt.Errorf("Anthropic API key phải bắt đầu bằng 'sk-ant-'")
		}
	case "openai":
		if !strings.HasPrefix(key, "sk-") {
			return fmt.Errorf("OpenAI API key phải bắt đầu bằng 'sk-'")
		}
	case "gemini":
		if len(key) < 10 {
			return fmt.Errorf("Gemini API key quá ngắn")
		}
	}
	return nil
}

// keychainKey tạo key duy nhất cho user+provider trong keychain
func keychainKey(userID int64, providerID string) string {
	return fmt.Sprintf("%s:user%d", providerID, userID)
}

// SaveToken validate format, lưu vào keychain và sync vào DB
func (tm *TokenManager) SaveToken(userID int64, providerID, token string) error {
	if err := tm.ValidateKeyFormat(providerID, token); err != nil {
		return fmt.Errorf("validate key: %w", err)
	}
	if tm.registry != nil {
		if _, exists := tm.registry.Get(providerID); !exists {
			return fmt.Errorf("provider %q không tồn tại trong registry", providerID)
		}
	}

	// Dùng composite key để hỗ trợ multi-user
	kcKey := keychainKey(userID, providerID)
	if tm.keychain != nil && token != "" {
		if err := tm.keychain.Set(kcKey, token); err != nil {
			return fmt.Errorf("lưu keychain: %w", err)
		}
	}

	if tm.tokenRepo != nil {
		authType := "api_key"
		if providerID == "ollama" {
			authType = "local"
		}
		if err := tm.tokenRepo.Upsert(repos.ProviderToken{
			UserID:      userID,
			ProviderID:  providerID,
			AuthType:    authType,
			KeychainKey: kcKey,
			IsActive:    true,
		}); err != nil {
			return fmt.Errorf("sync DB: %w", err)
		}
	}
	return nil
}

// GetToken đọc token từ keychain
func (tm *TokenManager) GetToken(userID int64, providerID string) (string, error) {
	if tm.keychain == nil {
		return "", fmt.Errorf("keychain chưa được khởi tạo")
	}
	kcKey := keychainKey(userID, providerID)
	return tm.keychain.Get(kcKey)
}

// DeleteToken xóa token khỏi keychain và đánh dấu inactive trong DB
func (tm *TokenManager) DeleteToken(userID int64, providerID string) error {
	if tm.keychain != nil {
		kcKey := keychainKey(userID, providerID)
		_ = tm.keychain.Delete(kcKey)
	}
	if tm.tokenRepo != nil {
		if err := tm.tokenRepo.Delete(userID, providerID); err != nil {
			return fmt.Errorf("xóa DB: %w", err)
		}
	}
	return nil
}
