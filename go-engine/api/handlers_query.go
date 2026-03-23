package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
	"github.com/minhtuancn/open-prompt/go-engine/model"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		Input     string            `json:"input"`
		Model     string            `json:"model"`
		System    string            `json:"system"`
		SlashName string            `json:"slash_name"`
		ExtraVars map[string]string `json:"extra_vars"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	// Nếu có slash_name, resolve template trước khi gửi lên model
	finalInput := p.Input
	if p.SlashName != "" {
		builder := engine.NewPromptBuilder()
		resolver := engine.NewCommandResolver(r.prompts, builder)
		resolved, resolveErr := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
		if resolveErr != nil {
			return nil, &RPCError{Code: -32002, Message: resolveErr.Error()}
		}
		if resolved.NeedsVars {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("slash command cần thêm biến: %v", resolved.RequiredVars)}
		}
		finalInput = resolved.RenderedPrompt
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

	// Bắt đầu tính latency và thu thập chunks
	start := time.Now()
	var chunks []string

	// Stream response qua JSON-RPC notifications
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	streamErr := modelRouter.Stream(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: finalInput,
		System: p.System,
	}, func(chunk string) {
		chunks = append(chunks, chunk)
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	latency := time.Since(start).Milliseconds()

	if streamErr != nil {
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": "",
			"done":  true,
			"error": fmt.Sprintf("%v", streamErr),
		})
		// Ghi history với trạng thái error
		_ = r.history.Insert(repos.InsertHistoryInput{
			UserID:    claims.UserID,
			Query:     finalInput,
			// TODO: lấy provider thực từ model router khi hỗ trợ multi-provider
			Provider:  "anthropic",
			Model:     modelName,
			LatencyMs: latency,
			Status:    "error",
		})
		return nil, nil // notification đã gửi
	}

	// Gửi notification done
	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	// Ghi history với trạng thái success
	_ = r.history.Insert(repos.InsertHistoryInput{
		UserID:    claims.UserID,
		Query:     finalInput,
		Response:  strings.Join(chunks, ""),
		// TODO: lấy provider thực từ model router khi hỗ trợ multi-provider
		Provider:  "anthropic",
		Model:     modelName,
		LatencyMs: latency,
		Status:    "success",
	})

	return nil, nil // response delivered via notifications
}
