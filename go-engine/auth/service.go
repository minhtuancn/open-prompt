package auth

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

const (
	minPasswordLen = 8
	// bcrypt silently truncate ở 72 bytes → enforce ở layer application
	MaxPasswordLen = 72
	maxUsernameLen = 64
)

var (
	ErrUserExists         = errors.New("username already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 72 characters")
	ErrUsernameTooLong    = errors.New("username must be at most 64 characters")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// Service xử lý authentication logic
type Service struct {
	users     *repos.UserRepo
	jwtSecret string
}

// NewService tạo auth service mới, trả về lỗi nếu jwtSecret quá ngắn
func NewService(userRepo *repos.UserRepo, jwtSecret string) (*Service, error) {
	if len(jwtSecret) < 16 {
		return nil, errors.New("jwt secret must be at least 16 bytes")
	}
	return &Service{users: userRepo, jwtSecret: jwtSecret}, nil
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
	if len(username) > maxUsernameLen {
		return nil, ErrUsernameTooLong
	}
	if len(password) < minPasswordLen {
		return nil, ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLen {
		return nil, ErrPasswordTooLong
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

// dummyHash là hash của "" dùng để cân bằng thời gian khi user không tồn tại,
// tránh user enumeration qua timing attack.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), config.DefaultBcryptCost)

// Login xác thực user và trả về JWT token
func (s *Service) Login(username, password string) (string, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		// Chạy dummy bcrypt để cân bằng thời gian — chống user enumeration
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password)) //nolint:errcheck
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	if err := s.users.UpdateLastLogin(user.ID); err != nil {
		// Không block login nếu audit trail thất bại
		fmt.Fprintf(os.Stderr, "warn: update last login for user %d: %v\n", user.ID, err)
	}

	return issueToken(user.ID, user.Username, s.jwtSecret)
}

// ValidateToken kiểm tra JWT và trả về claims
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, s.jwtSecret)
}
