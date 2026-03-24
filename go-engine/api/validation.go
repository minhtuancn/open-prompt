package api

// Giới hạn kích thước input
const (
	MaxTitleLen   = 200
	MaxContentLen = 50000
	MaxTagsLen    = 500
	MaxNameLen    = 32
	MaxLimit      = 1000
)

// extractToken lấy token từ request params (dùng cho rate limiting).
// Params đã được decode thành map[string]interface{} bởi json.Unmarshal nên
// dùng type assertion thay vì marshal/unmarshal lại.
func extractToken(req *Request) string {
	if req == nil || req.Params == nil {
		return ""
	}
	m, ok := req.Params.(map[string]interface{})
	if !ok {
		return ""
	}
	tok, _ := m["token"].(string)
	return tok
}

// clampLimit giới hạn limit trong khoảng hợp lệ
func clampLimit(limit, defaultVal int) int {
	if limit <= 0 {
		return defaultVal
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// truncateString cắt chuỗi nếu vượt quá maxLen
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
