package api_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

func setupServer(t *testing.T) (*api.Server, string) {
	t.Helper()
	database, _ := db.OpenInMemory()
	db.Migrate(database)
	t.Cleanup(func() { database.Close() })

	secret := "test-secret-16chars"
	srv, err := api.NewServer(secret, database)
	if err != nil {
		t.Fatal(err)
	}

	// Dùng random port TCP thay vì Unix socket cho test
	addr := srv.TestAddr()
	go srv.Listen()
	time.Sleep(50 * time.Millisecond) // đợi server ready
	t.Cleanup(srv.Close)
	return srv, addr
}

func callRPC(t *testing.T, addr, secret, method string, params interface{}) api.Response {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	req := api.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	// Thêm secret vào envelope
	msg, _ := json.Marshal(map[string]interface{}{
		"secret":  secret,
		"request": req,
	})
	conn.Write(append(msg, '\n'))

	var resp api.Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestAuthRegisterAndLogin(t *testing.T) {
	_, addr := setupServer(t)

	// Register
	resp := callRPC(t, addr, "test-secret-16chars", "auth.register", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	if resp.Error != nil {
		t.Fatalf("register error: %v", resp.Error)
	}

	// Login
	resp = callRPC(t, addr, "test-secret-16chars", "auth.login", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	if resp.Error != nil {
		t.Fatalf("login error: %v", resp.Error)
	}
}

func TestUnauthorizedRequest(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "wrong-secret", "auth.is_first_run", nil)
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Errorf("expected unauthorized error, got %v", resp.Error)
	}
}

func TestSecretEmptyStringRejected(t *testing.T) {
	_, addr := setupServer(t)
	resp := callRPC(t, addr, "", "auth.is_first_run", nil)
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Errorf("empty secret phải bị từ chối, got %v", resp.Error)
	}
}

func TestRegisterPasswordTooLong(t *testing.T) {
	_, addr := setupServer(t)
	longPwd := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 73 chars
	resp := callRPC(t, addr, "test-secret-16chars", "auth.register", map[string]string{
		"username": "testuser",
		"password": longPwd,
	})
	if resp.Error == nil {
		t.Error("password > 72 bytes phải bị từ chối")
	}
}

func TestLoginErrorIsGeneric(t *testing.T) {
	_, addr := setupServer(t)
	// Đăng ký user
	callRPC(t, addr, "test-secret-16chars", "auth.register", map[string]string{
		"username": "existinguser",
		"password": "password123",
	})

	// Login với username không tồn tại
	resp1 := callRPC(t, addr, "test-secret-16chars", "auth.login", map[string]string{
		"username": "nonexistent",
		"password": "anypassword",
	})
	// Login với sai password
	resp2 := callRPC(t, addr, "test-secret-16chars", "auth.login", map[string]string{
		"username": "existinguser",
		"password": "wrongpassword",
	})

	// Cả 2 trường hợp phải trả về cùng error code và message để chống user enumeration
	if resp1.Error == nil || resp2.Error == nil {
		t.Fatal("cả 2 login fail phải có error")
	}
	if resp1.Error.Code != resp2.Error.Code {
		t.Errorf("error code khác nhau: user-not-found=%d, wrong-password=%d", resp1.Error.Code, resp2.Error.Code)
	}
	if resp1.Error.Message != resp2.Error.Message {
		t.Errorf("error message khác nhau: user-not-found=%q, wrong-password=%q", resp1.Error.Message, resp2.Error.Message)
	}
}
