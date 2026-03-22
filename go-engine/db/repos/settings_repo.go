package repos

import (
	"database/sql"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// SettingsRepo xử lý key-value settings per user
type SettingsRepo struct {
	db *db.DB
}

// NewSettingsRepo tạo SettingsRepo mới
func NewSettingsRepo(database *db.DB) *SettingsRepo {
	return &SettingsRepo{db: database}
}

// Get lấy giá trị setting, trả về "" nếu không tồn tại
func (r *SettingsRepo) Get(userID int64, key string) (string, error) {
	var value sql.NullString
	err := r.db.QueryRow(
		`SELECT value FROM settings WHERE user_id = ? AND key = ?`, userID, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value.String, nil
}

// Set lưu setting (upsert)
func (r *SettingsRepo) Set(userID int64, key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO settings (user_id, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value`,
		userID, key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
