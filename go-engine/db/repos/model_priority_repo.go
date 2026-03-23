package repos

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// ModelPriority map với bảng model_priority trong DB
type ModelPriority struct {
	ID        int64
	UserID    int64
	Priority  int
	Provider  string
	Model     string
	IsEnabled bool
}

// ModelPriorityRepo thao tác với bảng model_priority
type ModelPriorityRepo struct {
	db *db.DB
}

// NewModelPriorityRepo tạo repo mới
func NewModelPriorityRepo(database *db.DB) *ModelPriorityRepo {
	return &ModelPriorityRepo{db: database}
}

// GetByUser trả về priority chain theo thứ tự ưu tiên tăng dần
func (r *ModelPriorityRepo) GetByUser(userID int64) ([]ModelPriority, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, priority, provider, model, is_enabled
		FROM model_priority
		WHERE user_id = ?
		ORDER BY priority ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query model_priority: %w", err)
	}
	defer rows.Close()

	var items []ModelPriority
	for rows.Next() {
		var m ModelPriority
		var isEnabled int
		if err := rows.Scan(&m.ID, &m.UserID, &m.Priority, &m.Provider, &m.Model, &isEnabled); err != nil {
			return nil, err
		}
		m.IsEnabled = isEnabled == 1
		items = append(items, m)
	}
	return items, rows.Err()
}

// SetChain thay thế toàn bộ priority chain của user
func (r *ModelPriorityRepo) SetChain(userID int64, chain []ModelPriority) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM model_priority WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete old chain: %w", err)
	}

	for i, m := range chain {
		isEnabled := 0
		if m.IsEnabled {
			isEnabled = 1
		}
		if _, err := tx.Exec(`
			INSERT INTO model_priority (user_id, priority, provider, model, is_enabled)
			VALUES (?, ?, ?, ?, ?)
		`, userID, i+1, m.Provider, m.Model, isEnabled); err != nil {
			return fmt.Errorf("insert priority %d: %w", i+1, err)
		}
	}

	return tx.Commit()
}

// DefaultChain trả về chain mặc định nếu user chưa cấu hình
func DefaultChain(userID int64) []ModelPriority {
	return []ModelPriority{
		{UserID: userID, Priority: 1, Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", IsEnabled: true},
		{UserID: userID, Priority: 2, Provider: "openai", Model: "gpt-4o-mini", IsEnabled: true},
		{UserID: userID, Priority: 3, Provider: "ollama", Model: "llama3.2", IsEnabled: true},
	}
}
