package api

// Giới hạn kích thước input
const (
	MaxTitleLen   = 200
	MaxContentLen = 50000
	MaxTagsLen    = 500
	MaxNameLen    = 32
)

// truncateString cắt chuỗi nếu vượt quá maxLen
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
