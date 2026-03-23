package api

// handleTelemetryOptIn bật/tắt telemetry cho user
func (r *Router) handleTelemetryOptIn(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token   string `json:"token"`
		Enabled bool   `json:"enabled"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	// Lưu vào settings table
	val := "false"
	if p.Enabled {
		val = "true"
	}
	if err := r.settings.Set(claims.UserID, "telemetry_enabled", val); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: err.Error()}
	}

	return map[string]bool{"enabled": p.Enabled}, nil
}

// handleTelemetryStatus trả về trạng thái telemetry
func (r *Router) handleTelemetryStatus(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	val, err := r.settings.Get(claims.UserID, "telemetry_enabled")
	if err != nil {
		return map[string]bool{"enabled": false}, nil
	}

	return map[string]bool{"enabled": val == "true"}, nil
}
