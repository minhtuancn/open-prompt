package api

import (
	"fmt"
	"net"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// Router map method → handler
type Router struct {
	server   *Server
	auth     *auth.Service
	users    *repos.UserRepo
	settings *repos.SettingsRepo
}

func newRouter(s *Server) (*Router, error) {
	users := repos.NewUserRepo(s.db)
	settings := repos.NewSettingsRepo(s.db)
	// JWT secret — hardcoded for Phase 1, will use keychain in Phase 2
	// Must be at least 16 bytes per NewService validation
	jwtSecret := "open-prompt-jwt-secret-v1-phase1"
	authSvc, err := auth.NewService(users, jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	return &Router{
		server:   s,
		auth:     authSvc,
		users:    users,
		settings: settings,
	}, nil
}

// dispatch gọi handler tương ứng với method
func (r *Router) dispatch(conn net.Conn, req *Request) (interface{}, *RPCError) {
	switch req.Method {
	case "auth.register":
		return r.handleRegister(req)
	case "auth.login":
		return r.handleLogin(req)
	case "auth.me":
		return r.handleMe(req)
	case "auth.is_first_run":
		return r.handleIsFirstRun(req)
	case "settings.get":
		return r.handleSettingsGet(req)
	case "settings.set":
		return r.handleSettingsSet(req)
	case "query.stream":
		return r.handleQueryStream(conn, req)
	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}
