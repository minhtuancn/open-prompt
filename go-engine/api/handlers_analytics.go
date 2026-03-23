package api

import (
	"strconv"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// parsePeriodDays chuyển chuỗi period sang số ngày, mặc định 7 nếu không hợp lệ
func parsePeriodDays(period string) int {
	switch period {
	case "30d":
		return 30
	case "90d":
		return 90
	default:
		return 7
	}
}

// handleAnalyticsSummary trả về tổng hợp theo ngày, provider và model
func (r *Router) handleAnalyticsSummary(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Period string `json:"period"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	days := parsePeriodDays(p.Period)
	summary, err := r.history.SummaryByPeriod(claims.UserID, days)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if summary == nil {
		summary = []repos.DailySummary{}
	}
	return map[string]interface{}{
		"summary": summary,
		"period":  strconv.Itoa(days) + "d",
	}, nil
}

// handleAnalyticsByProvider trả về tổng hợp theo provider
func (r *Router) handleAnalyticsByProvider(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}
	var p struct {
		Period string `json:"period"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	days := parsePeriodDays(p.Period)
	providers, err := r.history.TotalsByProvider(claims.UserID, days)
	if err != nil {
		return nil, copyErr(ErrInternal)
	}
	if providers == nil {
		providers = []repos.ProviderTotals{}
	}
	return map[string]interface{}{
		"providers": providers,
		"period":    strconv.Itoa(days) + "d",
	}, nil
}
