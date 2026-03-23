package providers

import "context"

// AuthType phân loại cơ chế xác thực của provider
type AuthType string

const (
	AuthAPIKey   AuthType = "api_key"
	AuthOAuth    AuthType = "oauth"
	AuthCLIToken AuthType = "cli_token"
	AuthNone     AuthType = "none"
)

// Provider là interface chung cho tất cả AI providers.
// CompletionRequest đã tồn tại ở anthropic.go với đầy đủ fields
// bao gồm Temperature — giữ nguyên, không tạo lại.
type Provider interface {
	// Name trả về tên chính (khớp DB key: "anthropic", "openai", "ollama")
	Name() string
	// DisplayName trả về tên hiển thị cho UI
	DisplayName() string
	// Aliases trả về tất cả alias (bao gồm cả Name)
	Aliases() []string
	// GetAuthType trả về loại xác thực
	// Tên GetAuthType thay vì AuthType để tránh conflict với type AuthType
	GetAuthType() AuthType
	// StreamComplete gửi request và stream kết quả qua onChunk callback
	StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error
	// Validate kiểm tra kết nối và xác thực
	Validate(ctx context.Context) error
	// Models trả về danh sách model IDs available
	Models(ctx context.Context) ([]string, error)
}
