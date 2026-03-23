package repos

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Prompt là model cho bảng prompts
type Prompt struct {
	ID        int64
	UserID    int64
	ProjectID sql.NullInt64
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreatePromptInput là input để tạo prompt mới
type CreatePromptInput struct {
	UserID    int64
	ProjectID int64
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
}

// UpdatePromptInput là input để cập nhật prompt
type UpdatePromptInput struct {
	Title     string
	Content   string
	Category  string
	Tags      string
	IsSlash   bool
	SlashName string
}

// PromptRepo xử lý CRUD cho bảng prompts
type PromptRepo struct {
	db *db.DB
}

// NewPromptRepo tạo PromptRepo mới
func NewPromptRepo(database *db.DB) *PromptRepo {
	return &PromptRepo{db: database}
}

// Create tạo prompt mới
func (r *PromptRepo) Create(input CreatePromptInput) (*Prompt, error) {
	isSlash := 0
	if input.IsSlash {
		isSlash = 1
	}
	res, err := r.db.Exec(
		`INSERT INTO prompts (user_id, title, content, category, tags, is_slash, slash_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Title, input.Content, input.Category, input.Tags, isSlash, input.SlashName,
	)
	if err != nil {
		return nil, fmt.Errorf("create prompt: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create prompt: get last insert id: %w", err)
	}
	return r.FindByID(id)
}

// FindByID tìm prompt theo ID (trả về nil nếu không tìm thấy)
func (r *PromptRepo) FindByID(id int64) (*Prompt, error) {
	p := &Prompt{}
	var isSlash int
	err := r.db.QueryRow(
		`SELECT id, user_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Category, &p.Tags, &isSlash, &p.SlashName, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find prompt by id: %w", err)
	}
	p.IsSlash = isSlash == 1
	return p, nil
}

// FindBySlashName tìm prompt theo slash_name và user_id
func (r *PromptRepo) FindBySlashName(userID int64, slashName string) (*Prompt, error) {
	p := &Prompt{}
	var isSlash int
	err := r.db.QueryRow(
		`SELECT id, user_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE user_id = ? AND slash_name = ? AND is_slash = 1`, userID, slashName,
	).Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Category, &p.Tags, &isSlash, &p.SlashName, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find prompt by slash name: %w", err)
	}
	p.IsSlash = true
	return p, nil
}

// List trả về danh sách prompts của user, filter theo category nếu không rỗng
func (r *PromptRepo) List(userID int64, category string) ([]*Prompt, error) {
	var rows *sql.Rows
	var err error
	if category == "" {
		rows, err = r.db.Query(
			`SELECT id, user_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
			 FROM prompts WHERE user_id = ? ORDER BY updated_at DESC`, userID,
		)
	} else {
		rows, err = r.db.Query(
			`SELECT id, user_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
			 FROM prompts WHERE user_id = ? AND category = ? ORDER BY updated_at DESC`, userID, category,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()

	var result []*Prompt
	for rows.Next() {
		p := &Prompt{}
		var isSlash int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Category, &p.Tags, &isSlash, &p.SlashName, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt: %w", err)
		}
		p.IsSlash = isSlash == 1
		result = append(result, p)
	}
	return result, rows.Err()
}

// ListSlashCommands trả về tất cả slash commands của user
func (r *PromptRepo) ListSlashCommands(userID int64) ([]*Prompt, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, title, content, category, tags, is_slash, slash_name, created_at, updated_at
		 FROM prompts WHERE user_id = ? AND is_slash = 1 ORDER BY slash_name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list slash commands: %w", err)
	}
	defer rows.Close()

	var result []*Prompt
	for rows.Next() {
		p := &Prompt{}
		var isSlash int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Category, &p.Tags, &isSlash, &p.SlashName, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt: %w", err)
		}
		p.IsSlash = true
		result = append(result, p)
	}
	return result, rows.Err()
}

// Update cập nhật prompt
func (r *PromptRepo) Update(id int64, input UpdatePromptInput) error {
	isSlash := 0
	if input.IsSlash {
		isSlash = 1
	}
	_, err := r.db.Exec(
		`UPDATE prompts SET title=?, content=?, category=?, tags=?, is_slash=?, slash_name=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		input.Title, input.Content, input.Category, input.Tags, isSlash, input.SlashName, id,
	)
	if err != nil {
		return fmt.Errorf("update prompt: %w", err)
	}
	return nil
}

// Delete xóa prompt theo ID
func (r *PromptRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM prompts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete prompt: %w", err)
	}
	return nil
}
