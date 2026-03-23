package api

// handleHealthCheck trả về trạng thái sức khỏe providers
func (r *Router) handleHealthCheck(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	if r.healthChecker == nil {
		return []interface{}{}, nil
	}

	return r.healthChecker.GetAll(), nil
}
