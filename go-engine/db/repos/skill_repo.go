package repos

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Skill là model cho bảng skills
type Skill struct {
	ID         int64
	UserID     int64
	Name       string
	PromptID   sql.NullInt64
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
	CreatedAt  time.Time
}

// CreateSkillInput là input để tạo skill mới
type CreateSkillInput struct {
	UserID     int64
	Name       string
	PromptID   int64
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
}

// SkillRepo xử lý CRUD cho bảng skills
type SkillRepo struct {
	db *db.DB
}

// NewSkillRepo tạo SkillRepo mới
func NewSkillRepo(database *db.DB) *SkillRepo {
	return &SkillRepo{db: database}
}

// Create tạo skill mới
func (r *SkillRepo) Create(input CreateSkillInput) (*Skill, error) {
	var promptID interface{}
	if input.PromptID > 0 {
		promptID = input.PromptID
	}
	res, err := r.db.Exec(
		`INSERT INTO skills (user_id, name, prompt_id, prompt_text, model, provider, config_json, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Name, promptID, input.PromptText, input.Model, input.Provider, input.ConfigJSON, input.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("create skill: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create skill: get last insert id: %w", err)
	}
	return r.FindByID(id)
}

// FindByID tìm skill theo ID (trả về nil nếu không tìm thấy)
func (r *SkillRepo) FindByID(id int64) (*Skill, error) {
	s := &Skill{}
	err := r.db.QueryRow(
		`SELECT id, user_id, name, prompt_id, prompt_text, model, provider, config_json, tags, created_at
		 FROM skills WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.PromptID, &s.PromptText, &s.Model, &s.Provider, &s.ConfigJSON, &s.Tags, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find skill by id: %w", err)
	}
	return s, nil
}

// List trả về danh sách skills của user
func (r *SkillRepo) List(userID int64) ([]*Skill, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, prompt_id, prompt_text, model, provider, config_json, tags, created_at
		 FROM skills WHERE user_id = ? ORDER BY name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	var result []*Skill
	for rows.Next() {
		s := &Skill{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.PromptID, &s.PromptText, &s.Model, &s.Provider, &s.ConfigJSON, &s.Tags, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// Delete xóa skill theo ID
func (r *SkillRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM skills WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

// UpdateSkillInput là input để cập nhật skill
type UpdateSkillInput struct {
	Name       string
	PromptID   int64
	PromptText string
	Model      string
	Provider   string
	ConfigJSON string
	Tags       string
}

// Update cập nhật skill theo ID
func (r *SkillRepo) Update(id int64, input UpdateSkillInput) error {
	var promptID interface{}
	if input.PromptID > 0 {
		promptID = input.PromptID
	}
	_, err := r.db.Exec(
		`UPDATE skills SET name=?, prompt_id=?, prompt_text=?, model=?, provider=?, config_json=?, tags=?
		 WHERE id=?`,
		input.Name, promptID, input.PromptText, input.Model, input.Provider, input.ConfigJSON, input.Tags, id,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	return nil
}
