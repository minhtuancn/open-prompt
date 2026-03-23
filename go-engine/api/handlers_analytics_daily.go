package api

import "fmt"

// handleAnalyticsAggregate chạy aggregation vào usage_daily table
func (r *Router) handleAnalyticsAggregate(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	// Aggregate từ history vào usage_daily — INSERT OR REPLACE cho today
	_, err := r.server.db.Exec(`
		INSERT OR REPLACE INTO usage_daily (date, user_id, provider, model, requests, input_tokens, output_tokens, errors, fallbacks, avg_latency_ms)
		SELECT
			date(timestamp) as date,
			user_id,
			COALESCE(provider, 'unknown') as provider,
			COALESCE(model, 'unknown') as model,
			COUNT(*) as requests,
			SUM(COALESCE(input_tokens, 0)) as input_tokens,
			SUM(COALESCE(output_tokens, 0)) as output_tokens,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors,
			SUM(CASE WHEN status = 'fallback' THEN 1 ELSE 0 END) as fallbacks,
			CAST(AVG(COALESCE(latency_ms, 0)) AS INTEGER) as avg_latency_ms
		FROM history
		WHERE user_id = ? AND date(timestamp) >= date('now', '-30 days')
		GROUP BY date(timestamp), user_id, provider, model
	`, claims.UserID)

	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("aggregate: %v", err)}
	}

	return map[string]bool{"ok": true}, nil
}

// handleAnalyticsDaily trả về daily usage stats
func (r *Router) handleAnalyticsDaily(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Days  int    `json:"days"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.Days <= 0 {
		p.Days = 30
	}

	rows, err := r.server.db.Query(`
		SELECT date, provider, model, requests, input_tokens, output_tokens, errors, fallbacks, avg_latency_ms
		FROM usage_daily
		WHERE user_id = ? AND date >= date('now', '-' || ? || ' days')
		ORDER BY date DESC, requests DESC
	`, claims.UserID, p.Days)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("query: %v", err)}
	}
	defer rows.Close()

	type DailyRow struct {
		Date         string `json:"date"`
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		Requests     int    `json:"requests"`
		InputTokens  int    `json:"input_tokens"`
		OutputTokens int    `json:"output_tokens"`
		Errors       int    `json:"errors"`
		Fallbacks    int    `json:"fallbacks"`
		AvgLatencyMs int    `json:"avg_latency_ms"`
	}

	var result []DailyRow
	for rows.Next() {
		var d DailyRow
		if err := rows.Scan(&d.Date, &d.Provider, &d.Model, &d.Requests, &d.InputTokens, &d.OutputTokens, &d.Errors, &d.Fallbacks, &d.AvgLatencyMs); err != nil {
			continue
		}
		result = append(result, d)
	}
	if result == nil {
		result = []DailyRow{}
	}
	return result, nil
}
