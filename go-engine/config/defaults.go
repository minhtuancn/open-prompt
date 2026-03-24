package config

import "os"

const (
	// DefaultTimeout là timeout mặc định cho AI requests (ms)
	DefaultTimeout = 30000

	// DefaultBcryptCost là cost factor cho bcrypt hashing
	DefaultBcryptCost = 12

	// DefaultJWTExpiry là thời gian expire của JWT session (ngày)
	DefaultJWTExpiry = 7

	// SocketEnvKey là env variable chứa shared secret
	SocketEnvKey = "OP_SOCKET_SECRET"

	// NamedPipeName là tên Named Pipe (Windows)
	NamedPipeName = `\\.\pipe\open-prompt`

	// HistoryRetentionDays là số ngày giữ raw history
	HistoryRetentionDays = 90
)

// SocketPath trả về path cho Unix socket.
// Ưu tiên: OP_SOCKET_PATH > XDG_RUNTIME_DIR/open-prompt.sock > /tmp/open-prompt.sock
func SocketPath() string {
	if p := os.Getenv("OP_SOCKET_PATH"); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir + "/open-prompt.sock"
	}
	return "/tmp/open-prompt.sock"
}

// DBFileName trả về tên file database.
// Có thể override qua OP_DB_PATH (full path) hoặc dùng default "open-prompt.db"
func DBFileName() string {
	return "open-prompt.db"
}

// DBPath trả về full path tới database file nếu override qua OP_DB_PATH
func DBPath() string {
	return os.Getenv("OP_DB_PATH")
}
