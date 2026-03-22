package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

var (
	ErrUserExists       = errors.New("username already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

// Service xử lý authentication logic
type Service struct {
	users     *repos.UserRepo
	jwtSecret string
}

// NewService tạo auth service mới
func NewService(userRepo *repos.UserRepo, jwtSecret string) *Service {
	return &Service{users: userRepo, jwtSecret: jwtSecret}
}

// IsFirstRun kiểm tra xem có user nào trong DB chưa
func (s *Service) IsFirstRun() (bool, error) {
	count, err := s.users.Count()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Register tạo user mới với bcrypt password
func (s *Service) Register(username, password string) (*repos.User, error) {
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Kiểm tra username đã tồn tại chưa
	existing, err := s.users.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), config.DefaultBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	return s.users.Create(username, string(hash))
}

// Login xác thực user và trả về JWT token
func (s *Service) Login(username, password string) (string, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return "", ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidPassword
	}

	_ = s.users.UpdateLastLogin(user.ID)

	return issueToken(user.ID, user.Username, s.jwtSecret)
}

// ValidateToken kiểm tra JWT và trả về claims
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, s.jwtSecret)
}
