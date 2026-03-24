package api

import "encoding/json"

// Giới hạn kích thước input
const (
	MaxTitleLen    = 200
	MaxContentLen  = 50000
	MaxTagsLen     = 500
	MaxNameLen     = 32
	MaxLimit       = 1000
	MaxTemplateLen = 100000 // 100KB max template size
)

// extractToken lấy token từ request params (dùng cho rate limiting)
func extractToken(req *Request) string {
	if req == nil || req.Params == nil {
		return ""
	}
	raw, err := json.Marshal(req.Params)
	if err != nil {
		return ""
	}
	var p struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return ""
	}
	return p.Token
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
