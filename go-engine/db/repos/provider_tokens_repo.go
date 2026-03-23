package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// ProviderToken map với bảng provider_tokens trong DB
type ProviderToken struct {
	ID            int64
	UserID        int64
	ProviderID    string
	AuthType      string
	KeychainKey   string
	ExpiresAt     *time.Time
	DetectedAt    *time.Time
	LastRefreshed *time.Time
	IsActive      bool
}

// ProviderTokenRepo thao tác với bảng provider_tokens
type ProviderTokenRepo struct {
	db *db.DB
}

// NewProviderTokenRepo tạo repo mới
func NewProviderTokenRepo(database *db.DB) *ProviderTokenRepo {
	return &ProviderTokenRepo{db: database}
}

// Upsert thêm hoặc cập nhật provider token
func (r *ProviderTokenRepo) Upsert(t ProviderToken) error {
	now := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO provider_tokens (user_id, provider_id, auth_type, keychain_key, detected_at, is_active)
		VALUES (?, ?, ?, ?, ?, 1)
		ON CONFLICT(user_id, provider_id) DO UPDATE SET
			auth_type    = excluded.auth_type,
			keychain_key = excluded.keychain_key,
			detected_at  = excluded.detected_at,
			is_active    = 1
	`, t.UserID, t.ProviderID, t.AuthType, t.KeychainKey, now)
	if err != nil {
		return fmt.Errorf("upsert provider_token: %w", err)
	}
	return nil
}

// GetByUser trả về tất cả tokens của user
func (r *ProviderTokenRepo) GetByUser(userID int64) ([]ProviderToken, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, provider_id, auth_type, keychain_key, is_active
		FROM provider_tokens
		WHERE user_id = ? AND is_active = 1
		ORDER BY provider_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query provider_tokens: %w", err)
	}
	defer rows.Close()

	var tokens []ProviderToken
	for rows.Next() {
		var t ProviderToken
		var isActive int
		if err := rows.Scan(&t.ID, &t.UserID, &t.ProviderID, &t.AuthType, &t.KeychainKey, &isActive); err != nil {
			return nil, err
		}
		t.IsActive = isActive == 1
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// Delete đánh dấu token không còn active
func (r *ProviderTokenRepo) Delete(userID int64, providerID string) error {
	_, err := r.db.Exec(`
		UPDATE provider_tokens SET is_active = 0
		WHERE user_id = ? AND provider_id = ?
	`, userID, providerID)
	if err != nil {
		return fmt.Errorf("delete provider_token: %w", err)
	}
	return nil
}
