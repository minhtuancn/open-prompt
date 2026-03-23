package api_test

import (
	"testing"
)

func TestAnalyticsSummaryRequiresAuth(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": "bad-token", "period": "7d",
	})
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Errorf("expected -32001, got %v", resp.Error)
	}
}

func TestAnalyticsSummaryEmpty(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser1", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": token, "period": "7d",
	})
	if resp.Error != nil {
		t.Fatalf("analytics.summary error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["summary"]; !exists {
		t.Error("phải có field 'summary'")
	}
}

func TestAnalyticsByProviderEmpty(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser2", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "analytics.by_provider", map[string]interface{}{
		"token": token, "period": "30d",
	})
	if resp.Error != nil {
		t.Fatalf("analytics.by_provider error: %v", resp.Error)
	}
	m := resultMap(t, resp)
	if _, exists := m["providers"]; !exists {
		t.Error("phải có field 'providers'")
	}
}

func TestAnalyticsPeriodDefault(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "analyticsuser3", "pass1234")

	resp := callRPC(t, addr, "test-secret-16chars", "analytics.summary", map[string]interface{}{
		"token": token, "period": "invalid",
	})
	if resp.Error != nil {
		t.Fatalf("period không hợp lệ phải dùng default, got error: %v", resp.Error)
	}
}
