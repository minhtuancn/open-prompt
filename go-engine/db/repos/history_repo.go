package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
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
	Status    string // "success" | "error"
}

// Insert ghi một bản ghi history mới
func (r *HistoryRepo) Insert(input InsertHistoryInput) error {
	status := input.Status
	if status == "" {
		status = "success"
	}
	_, err := r.db.Exec(
		`INSERT INTO history (user_id, query, response, provider, model, latency_ms, status, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Query, input.Response, input.Provider, input.Model,
		input.LatencyMs, status, time.Now().UTC(),
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
