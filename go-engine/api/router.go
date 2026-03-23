package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	prompts  *repos.PromptRepo
}

func newRouter(s *Server) (*Router, error) {
	users := repos.NewUserRepo(s.db)
	settings := repos.NewSettingsRepo(s.db)
	prompts := repos.NewPromptRepo(s.db)
	// Derive JWT secret từ socket secret bằng HMAC-SHA256.
	// Mỗi session có socket secret ngẫu nhiên → JWT secret cũng unique per-session.
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte("jwt-signing-key"))
	jwtSecret := hex.EncodeToString(mac.Sum(nil))
	authSvc, err := auth.NewService(users, jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	return &Router{
		server:   s,
		auth:     authSvc,
		users:    users,
		settings: settings,
		prompts:  prompts,
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
	case "providers.list":
		return r.handleProvidersList(req)
	case "providers.detect":
		return r.handleProvidersDetect(req)
	case "providers.connect":
		return r.handleProvidersConnect(req)
	case "providers.set_priority":
		return r.handleProvidersSetPriority(req)
	case "prompts.list":
		return r.handlePromptsList(req)
	case "prompts.create":
		return r.handlePromptsCreate(req)
	case "prompts.update":
		return r.handlePromptsUpdate(req)
	case "prompts.delete":
		return r.handlePromptsDelete(req)
	case "commands.list":
		return r.handleCommandsList(req)
	case "commands.resolve":
		return r.handleCommandsResolve(req)
	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}
