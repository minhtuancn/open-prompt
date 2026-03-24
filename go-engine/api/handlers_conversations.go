package api

import (
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// handleConversationsList trả về danh sách conversations
func (r *Router) handleConversationsList(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Limit int    `json:"limit"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	convs, err := r.conversations.List(claims.UserID, p.Limit)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("list: %v", err)}
	}
	if convs == nil {
		convs = []repos.Conversation{}
	}
	return convs, nil
}

// handleConversationsCreate tạo conversation mới
func (r *Router) handleConversationsCreate(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Title string `json:"title"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}
	title := p.Title
	if title == "" {
		title = "Chat mới"
	}

	id, err := r.conversations.Create(claims.UserID, title)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("create: %v", err)}
	}
	return map[string]interface{}{"id": id, "title": title}, nil
}

// handleConversationsMessages trả về messages của conversation
func (r *Router) handleConversationsMessages(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token          string `json:"token"`
		ConversationID int64  `json:"conversation_id"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ConversationID == 0 {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "conversation_id bắt buộc"}
	}

	msgs, err := r.conversations.GetMessages(p.ConversationID, claims.UserID)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("messages: %v", err)}
	}
	if msgs == nil {
		msgs = []repos.Message{}
	}
	return msgs, nil
}

// handleConversationsDelete xóa conversation
func (r *Router) handleConversationsDelete(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token          string `json:"token"`
		ConversationID int64  `json:"conversation_id"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.ConversationID == 0 {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "conversation_id bắt buộc"}
	}

	if err := r.conversations.Delete(p.ConversationID, claims.UserID); err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: err.Error()}
	}
	return map[string]bool{"ok": true}, nil
}
