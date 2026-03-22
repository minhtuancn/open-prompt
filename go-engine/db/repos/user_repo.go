package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// User là model cho bảng users
type User struct {
	ID           int64
	Username     string
	DisplayName  sql.NullString
	PasswordHash string
	AvatarColor  string
	CreatedAt    time.Time
	LastLogin    sql.NullTime
}

// UserRepo xử lý CRUD cho bảng users
type UserRepo struct {
	db *db.DB
}

// NewUserRepo tạo UserRepo mới
func NewUserRepo(database *db.DB) *UserRepo {
	return &UserRepo{db: database}
}

// Create tạo user mới
func (r *UserRepo) Create(username, passwordHash string) (*User, error) {
	res, err := r.db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

// FindByUsername tìm user theo username (trả về nil nếu không tìm thấy)
func (r *UserRepo) FindByUsername(username string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, display_name, password_hash, avatar_color, created_at, last_login
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarColor, &u.CreatedAt, &u.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return u, nil
}

// FindByID tìm user theo ID (trả về nil nếu không tìm thấy)
func (r *UserRepo) FindByID(id int64) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, display_name, password_hash, avatar_color, created_at, last_login
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarColor, &u.CreatedAt, &u.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

// Count trả về tổng số users (dùng để detect first-run)
func (r *UserRepo) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// UpdateLastLogin cập nhật thời gian login cuối
func (r *UserRepo) UpdateLastLogin(id int64) error {
	_, err := r.db.Exec(`UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}
