package api

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/model"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token  string `json:"token"`
		Input  string `json:"input"`
		Model  string `json:"model"`
		System string `json:"system"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	// Lấy API key từ settings
	apiKey, _ := r.settings.Get(claims.UserID, "anthropic_api_key")
	if apiKey == "" {
		return nil, copyErr(ErrProviderNotFound)
	}

	// Build model router
	modelRouter := model.NewRouter()
	modelRouter.RegisterAnthropic(apiKey)

	modelName := p.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	// Stream response qua JSON-RPC notifications
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	streamErr := modelRouter.Stream(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: p.Input,
		System: p.System,
	}, func(chunk string) {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	if streamErr != nil {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": "",
			"done":  true,
			"error": fmt.Sprintf("%v", streamErr),
		})
		return nil, nil // notification đã gửi
	}

	// Gửi notification done
	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	return nil, nil // response delivered via notifications
}
