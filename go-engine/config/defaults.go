package config

const (
	// DefaultTimeout là timeout mặc định cho AI requests (ms)
	DefaultTimeout = 30000

	// DefaultBcryptCost là cost factor cho bcrypt hashing
	DefaultBcryptCost = 12

	// DefaultJWTExpiry là thời gian expire của JWT session (ngày)
	DefaultJWTExpiry = 7

	// SocketEnvKey là env variable chứa shared secret
	SocketEnvKey = "OP_SOCKET_SECRET"

	// SocketPath là path của Unix socket (Linux/macOS)
	SocketPath = "/tmp/open-prompt.sock"

	// NamedPipeName là tên Named Pipe (Windows)
	NamedPipeName = `\\.\pipe\open-prompt`

	// DBFileName là tên file SQLite
	DBFileName = "open-prompt.db"

	// HistoryRetentionDays là số ngày giữ raw history
	HistoryRetentionDays = 90
)
