package api

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Server là JSON-RPC server qua Unix socket / Named Pipe / TCP (test)
type Server struct {
	secret      string
	secretBytes []byte // precomputed để tránh alloc mỗi request
	db          *db.DB
	router      *Router
	listener    net.Listener
}

// NewServer tạo server mới
func NewServer(secret string, database *db.DB) (*Server, error) {
	s := &Server{
		secret:      secret,
		secretBytes: []byte(secret),
		db:          database,
	}
	router, err := newRouter(s)
	if err != nil {
		return nil, fmt.Errorf("init router: %w", err)
	}
	s.router = router
	return s, nil
}

// PrepareListener tạo listener trước khi Listen() được gọi.
// Phải gọi trước Listen() để lấy Addr() (cần thiết trên Windows để biết port).
func (s *Server) PrepareListener() error {
	ln, err := createListener()
	if err != nil {
		return fmt.Errorf("create listener: %w", err)
	}
	s.listener = ln
	return nil
}

// Addr trả về địa chỉ listener (chỉ hợp lệ sau PrepareListener hoặc TestAddr).
func (s *Server) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// TestAddr tạo TCP listener trên random port và trả về addr (chỉ dùng cho test)
func (s *Server) TestAddr() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("TestAddr: failed to create listener: %v", err))
	}
	s.listener = ln
	return ln.Addr().String()
}

// Listen bắt đầu accept loop. Phải gọi PrepareListener() trước.
func (s *Server) Listen() error {
	if s.listener == nil {
		return fmt.Errorf("listener not prepared: call PrepareListener first")
	}
	log.Printf("listening on %s", s.listener.Addr())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// net.ErrClosed được trả về khi listener bị đóng có chủ ý
			if strings.Contains(err.Error(), "use of closed") {
				return nil
			}
			log.Printf("accept error: %v", err)
			return err
		}
		go s.handleConn(conn)
	}
}

// Close đóng server
func (s *Server) Close() {
	if s.router != nil {
		if s.router.rateLimiter != nil {
			s.router.rateLimiter.Stop()
		}
		if s.router.healthChecker != nil {
			s.router.healthChecker.Stop()
		}
		if s.router.tokenExpiryWatcher != nil {
			s.router.tokenExpiryWatcher.Stop()
		}
	}
	if s.listener != nil {
		s.listener.Close()
	}
}

// handleConn xử lý một connection
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	// 1MB scanner buffer — limits max single JSON-RPC message size.
	// Increase if clients need to send larger payloads (e.g. base64 attachments).
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		resp := s.processMessage(conn, line)
		if resp != nil {
			data, err := json.Marshal(resp)
			if err != nil {
				log.Printf("marshal response: %v", err)
				continue
			}
			if _, err := conn.Write(append(data, '\n')); err != nil {
				log.Printf("write response: %v", err)
				return
			}
		}
	}
}

// processMessage decode và dispatch một message
func (s *Server) processMessage(conn net.Conn, data []byte) *Response {
	// Decode envelope với secret
	var envelope struct {
		Secret  string  `json:"secret"`
		Request Request `json:"request"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return &Response{JSONRPC: "2.0", Error: copyErr(ErrInvalidParams)}
	}

	// Validate secret — dùng constant-time compare để chống timing attack
	if subtle.ConstantTimeCompare([]byte(envelope.Secret), s.secretBytes) != 1 {
		return &Response{JSONRPC: "2.0", Error: copyErr(ErrUnauthorized), ID: envelope.Request.ID}
	}

	// Dispatch
	result, rpcErr := s.router.dispatch(conn, &envelope.Request)
	if rpcErr != nil {
		return &Response{JSONRPC: "2.0", Error: rpcErr, ID: envelope.Request.ID}
	}
	resp := NewResponse(envelope.Request.ID, result)
	return &resp
}

// createListener tạo Unix socket (Linux/macOS) hoặc TCP localhost (Windows)
func createListener() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	sockPath := config.SocketPath()
	os.Remove(sockPath) // xóa stale socket
	return net.Listen("unix", sockPath)
}

// SendNotification gửi JSON-RPC notification qua connection (dùng cho streaming)
func SendNotification(conn net.Conn, method string, params interface{}) error {
	n := Notification{JSONRPC: "2.0", Method: method, Params: params}
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}
