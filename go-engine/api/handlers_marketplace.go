package api

import (
	"fmt"
	"regexp"
)

// htmlTagRegex strip HTML tags từ input
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// sanitizeMarketplaceText loại bỏ HTML tags và giới hạn độ dài
func sanitizeMarketplaceText(s string, maxLen int) string {
	s = htmlTagRegex.ReplaceAllString(s, "")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

// handleMarketplaceList trả về danh sách public prompts
func (r *Router) handleMarketplaceList(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
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

	items, err := r.marketplace.List(p.Limit, p.Offset)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("list: %v", err)}
	}
	return items, nil
}

// handleMarketplaceSearch tìm kiếm prompts trong marketplace
func (r *Router) handleMarketplaceSearch(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.Query == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "query is required"}
	}

	items, err := r.marketplace.Search(p.Query, p.Limit)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("search: %v", err)}
	}
	return items, nil
}

// handleMarketplacePublish — user publish prompt ra marketplace
func (r *Router) handleMarketplacePublish(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token       string `json:"token"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Tags        string `json:"tags"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.Title == "" || p.Content == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "title and content are required"}
	}

	// Sanitize input — strip HTML, giới hạn size (chống XSS + DB bloat)
	p.Title = sanitizeMarketplaceText(p.Title, 200)
	p.Content = sanitizeMarketplaceText(p.Content, 50000)
	p.Description = sanitizeMarketplaceText(p.Description, 1000)
	p.Category = sanitizeMarketplaceText(p.Category, 50)
	p.Tags = sanitizeMarketplaceText(p.Tags, 200)

	id, err := r.marketplace.Publish(claims.UserID, p.Title, p.Content, p.Description, p.Category, p.Tags)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("publish: %v", err)}
	}
	return map[string]int64{"id": id}, nil
}

// handleMarketplaceInstall — copy shared prompt vào user's prompts
func (r *Router) handleMarketplaceInstall(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		ID    int64  `json:"id"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	if p.ID <= 0 {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "id is required"}
	}

	if err := r.marketplace.Install(p.ID, claims.UserID); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("install: %v", err)}
	}
	return map[string]bool{"ok": true}, nil
}
