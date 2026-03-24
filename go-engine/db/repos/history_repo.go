package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// HistoryStatus là trạng thái của một history record
type HistoryStatus string

const (
	HistoryStatusSuccess HistoryStatus = "success"
	HistoryStatusError   HistoryStatus = "error"
)

// HistoryRepo xử lý ghi và truy vấn bảng history
type HistoryRepo struct {
	db *db.DB
}

// NewHistoryRepo tạo HistoryRepo mới
func NewHistoryRepo(database *db.DB) *HistoryRepo {
	return &HistoryRepo{db: database}
}

// InsertHistoryInput là input để ghi một bản ghi history
type InsertHistoryInput struct {
	UserID    int64
	Query     string
	Response  string
	Provider  string
	Model     string
	LatencyMs int64
	Status    HistoryStatus
}

// Insert ghi một bản ghi history mới
func (r *HistoryRepo) Insert(input InsertHistoryInput) error {
	status := input.Status
	if status == "" {
		status = HistoryStatusSuccess
	}
	// Định dạng timestamp theo chuẩn SQLite để date() và datetime() nhận diện đúng
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		`INSERT INTO history (user_id, query, response, provider, model, latency_ms, status, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Query, input.Response, input.Provider, input.Model,
		input.LatencyMs, status, now,
	)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

// DailySummary tổng hợp theo ngày, provider và model
type DailySummary struct {
	Date         string `json:"date"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Requests     int    `json:"requests"`
	Errors       int    `json:"errors"`
	AvgLatencyMs int    `json:"avg_latency_ms"`
}

// SummaryByPeriod trả về tổng hợp theo ngày trong khoảng `days` ngày gần nhất
func (r *HistoryRepo) SummaryByPeriod(userID int64, days int) ([]DailySummary, error) {
	rows, err := r.db.Query(
		`SELECT date(timestamp) as date, provider, model,
		        COUNT(*) as requests,
		        SUM(CASE WHEN status='error' THEN 1 ELSE 0 END) as errors,
		        CAST(AVG(latency_ms) AS INTEGER) as avg_latency_ms
		 FROM history
		 WHERE user_id = ? AND timestamp >= datetime('now', '-' || ? || ' days')
		 GROUP BY date(timestamp), provider, model
		 ORDER BY date DESC, requests DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("summary by period: %w", err)
	}
	defer rows.Close()

	var result []DailySummary
	for rows.Next() {
		var s DailySummary
		if err := rows.Scan(&s.Date, &s.Provider, &s.Model, &s.Requests, &s.Errors, &s.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ProviderTotals tổng hợp theo provider
type ProviderTotals struct {
	Provider    string  `json:"provider"`
	Requests    int     `json:"requests"`
	Errors      int     `json:"errors"`
	SuccessRate float64 `json:"success_rate"`
}

// TotalsByProvider trả về tổng hợp theo provider trong khoảng `days` ngày gần nhất
func (r *HistoryRepo) TotalsByProvider(userID int64, days int) ([]ProviderTotals, error) {
	rows, err := r.db.Query(
		`SELECT provider,
		        COUNT(*) as requests,
		        SUM(CASE WHEN status='error' THEN 1 ELSE 0 END) as errors
		 FROM history
		 WHERE user_id = ? AND timestamp >= datetime('now', '-' || ? || ' days')
		 GROUP BY provider
		 ORDER BY requests DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("totals by provider: %w", err)
	}
	defer rows.Close()

	var result []ProviderTotals
	for rows.Next() {
		var p ProviderTotals
		if err := rows.Scan(&p.Provider, &p.Requests, &p.Errors); err != nil {
			return nil, fmt.Errorf("scan provider totals: %w", err)
		}
		if p.Requests > 0 {
			p.SuccessRate = float64(p.Requests-p.Errors) / float64(p.Requests) * 100
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// HistoryEntry là một bản ghi history đầy đủ
type HistoryEntry struct {
	ID        int64  `json:"id"`
	Query     string `json:"query"`
	Response  string `json:"response"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	LatencyMs int64  `json:"latency_ms"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// List trả về history entries với pagination
func (r *HistoryRepo) List(userID int64, limit, offset int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}
	rows, err := r.db.Query(
		`SELECT id, query, COALESCE(response,''), COALESCE(provider,''), COALESCE(model,''),
		        COALESCE(latency_ms,0), COALESCE(status,'success'), timestamp
		 FROM history WHERE user_id = ?
		 ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var result []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.Query, &e.Response, &e.Provider, &e.Model, &e.LatencyMs, &e.Status, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// Search tìm history theo query text
func (r *HistoryRepo) Search(userID int64, search string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 1000 {
		limit = 1000
	}
	pattern := "%" + search + "%"
	rows, err := r.db.Query(
		`SELECT id, query, COALESCE(response,''), COALESCE(provider,''), COALESCE(model,''),
		        COALESCE(latency_ms,0), COALESCE(status,'success'), timestamp
		 FROM history WHERE user_id = ? AND (query LIKE ? OR response LIKE ?)
		 ORDER BY timestamp DESC LIMIT ?`,
		userID, pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search history: %w", err)
	}
	defer rows.Close()

	var result []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.Query, &e.Response, &e.Provider, &e.Model, &e.LatencyMs, &e.Status, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
