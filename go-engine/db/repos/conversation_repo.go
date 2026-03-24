package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Conversation là một cuộc hội thoại multi-turn
type Conversation struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Message là một tin nhắn trong cuộc hội thoại
type Message struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	LatencyMs      int64  `json:"latency_ms"`
	CreatedAt      string `json:"created_at"`
}

// ConversationRepo xử lý CRUD cho conversations và messages
type ConversationRepo struct {
	db *db.DB
}

// NewConversationRepo tạo repo mới
func NewConversationRepo(database *db.DB) *ConversationRepo {
	return &ConversationRepo{db: database}
}

// Create tạo conversation mới, trả về ID
func (r *ConversationRepo) Create(userID int64, title string) (int64, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := r.db.Exec(
		`INSERT INTO conversations (user_id, title, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		userID, title, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("create conversation: %w", err)
	}
	return result.LastInsertId()
}

// List trả về conversations của user, mới nhất trước
func (r *ConversationRepo) List(userID int64, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, user_id, title, COALESCE(provider,''), COALESCE(model,''), created_at, updated_at
		 FROM conversations WHERE user_id = ? ORDER BY updated_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var result []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Provider, &c.Model, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// AddMessage thêm message vào conversation (có kiểm tra ownership)
func (r *ConversationRepo) AddMessage(convID, userID int64, role, content, provider, model string, latencyMs int64) error {
	var ownerID int64
	err := r.db.QueryRow(`SELECT user_id FROM conversations WHERE id = ?`, convID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("forbidden: conversation does not belong to user")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err = r.db.Exec(
		`INSERT INTO messages (conversation_id, role, content, provider, model, latency_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		convID, role, content, provider, model, latencyMs, now,
	)
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	// Cập nhật updated_at của conversation
	_, _ = r.db.Exec(`UPDATE conversations SET updated_at = ? WHERE id = ?`, now, convID)
	return nil
}

// GetMessages trả về tất cả messages của conversation (có kiểm tra ownership)
func (r *ConversationRepo) GetMessages(convID, userID int64) ([]Message, error) {
	var ownerID int64
	err := r.db.QueryRow(`SELECT user_id FROM conversations WHERE id = ?`, convID).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if ownerID != userID {
		return nil, fmt.Errorf("forbidden: conversation does not belong to user")
	}

	rows, err := r.db.Query(
		`SELECT id, conversation_id, role, content, COALESCE(provider,''), COALESCE(model,''),
		        COALESCE(latency_ms,0), created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`,
		convID,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var result []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Provider, &m.Model, &m.LatencyMs, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// Delete xóa conversation và messages
func (r *ConversationRepo) Delete(convID, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM conversations WHERE id = ? AND user_id = ?`, convID, userID)
	return err
}
