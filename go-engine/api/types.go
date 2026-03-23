package api

// Request là cấu trúc JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

// Response là cấu trúc JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Notification là JSON-RPC 2.0 notification (không có id, dùng cho streaming)
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// RPCError là error object trong JSON-RPC 2.0
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Predefined error codes
var (
	ErrUnauthorized       = &RPCError{Code: -32001, Message: "unauthorized"}
	ErrProviderNotFound   = &RPCError{Code: -32002, Message: "provider_not_found"}
	ErrAllProvidersFailed = &RPCError{Code: -32003, Message: "all_providers_failed"}
	ErrForbidden          = &RPCError{Code: -32004, Message: "forbidden"}
	ErrMethodNotFound     = &RPCError{Code: -32601, Message: "method_not_found"}
	ErrInvalidParams      = &RPCError{Code: -32602, Message: "invalid_params"}
	ErrInternal           = &RPCError{Code: -32603, Message: "internal_error"}
)

// copyErr trả về bản sao của RPCError để tránh mutation của shared pointer
func copyErr(e *RPCError) *RPCError {
	return &RPCError{Code: e.Code, Message: e.Message}
}

// NewResponse tạo success response
func NewResponse(id interface{}, result interface{}) Response {
	return Response{JSONRPC: "2.0", Result: result, ID: id}
}

// NewErrorResponse tạo error response
func NewErrorResponse(id interface{}, err *RPCError) Response {
	return Response{JSONRPC: "2.0", Error: err, ID: id}
}
