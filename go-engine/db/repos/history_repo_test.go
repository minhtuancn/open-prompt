package repos_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func TestHistoryRepo(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewHistoryRepo(database)

	// Test Insert — bản ghi thành công
	err := repo.Insert(repos.InsertHistoryInput{
		UserID:    1,
		Query:     "test query",
		Response:  "test response",
		Provider:  "anthropic",
		Model:     "claude-3-5-sonnet",
		LatencyMs: 500,
		Status:    "success",
	})
	if err != nil {
		t.Fatalf("Insert bản ghi success thất bại: %v", err)
	}

	// Test Insert — bản ghi lỗi
	err = repo.Insert(repos.InsertHistoryInput{
		UserID:    1,
		Query:     "test query 2",
		Provider:  "anthropic",
		Model:     "claude-3-5-sonnet",
		LatencyMs: 100,
		Status:    "error",
	})
	if err != nil {
		t.Fatalf("Insert bản ghi error thất bại: %v", err)
	}

	// Test SummaryByPeriod — phải trả về 1 nhóm ngày với 2 requests
	summary, err := repo.SummaryByPeriod(1, 7)
	if err != nil {
		t.Fatalf("SummaryByPeriod thất bại: %v", err)
	}
	if len(summary) == 0 {
		t.Fatal("SummaryByPeriod phải trả về ít nhất 1 row")
	}
	if summary[0].Requests != 2 {
		t.Errorf("Requests = %d, want 2", summary[0].Requests)
	}
	if summary[0].Errors != 1 {
		t.Errorf("Errors = %d, want 1", summary[0].Errors)
	}

	// Test TotalsByProvider — 2 requests, 1 error = 50% success rate
	totals, err := repo.TotalsByProvider(1, 7)
	if err != nil {
		t.Fatalf("TotalsByProvider thất bại: %v", err)
	}
	if len(totals) == 0 {
		t.Fatal("TotalsByProvider phải trả về ít nhất 1 row")
	}
	if totals[0].Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", totals[0].Provider)
	}
	if totals[0].Requests != 2 {
		t.Errorf("Requests = %d, want 2", totals[0].Requests)
	}
	if totals[0].SuccessRate != 50.0 {
		t.Errorf("SuccessRate = %f, want 50.0", totals[0].SuccessRate)
	}

	// Test isolation — user khác không được thấy dữ liệu
	empty, err := repo.SummaryByPeriod(99, 7)
	if err != nil {
		t.Fatalf("SummaryByPeriod cho user 99 thất bại: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("user 99 phải thấy 0 records, got %d", len(empty))
	}
}
