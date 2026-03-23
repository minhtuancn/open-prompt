package api

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// handleHistoryList trả về lịch sử queries với pagination
func (r *Router) handleHistoryList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token  string `json:"token"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	entries, err := r.history.List(claims.UserID, p.Limit, p.Offset)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("list history: %v", err)}
	}
	if entries == nil {
		entries = []repos.HistoryEntry{}
	}

	return entries, nil
}

// handleHistorySearch tìm kiếm history theo text
func (r *Router) handleHistorySearch(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token  string `json:"token"`
		Search string `json:"search"`
		Limit  int    `json:"limit"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.Search == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "search không được để trống"}
	}

	entries, err := r.history.Search(claims.UserID, p.Search, p.Limit)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("search history: %v", err)}
	}
	if entries == nil {
		entries = []repos.HistoryEntry{}
	}

	return entries, nil
}
