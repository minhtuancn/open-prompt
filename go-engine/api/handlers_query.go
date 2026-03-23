package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		Input     string            `json:"input"`
		Model     string            `json:"model"`
		System    string            `json:"system"`
		Provider  string            `json:"provider"`
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

	// Resolve slash command nếu có
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

	// Xác định provider: explicit param > @mention > default
	alias := p.Provider
	if alias == "" {
		var cleanInput string
		alias, cleanInput = ParseMention(finalInput)
		if alias != "" {
			finalInput = cleanInput
		}
	}

	// Route đến provider
	var prov providers.Provider
	if alias != "" {
		prov, err = r.providerRegistry.Route(alias)
	} else {
		prov, err = r.providerRegistry.Default()
	}

	// Fallback: nếu registry rỗng, thử lấy API key từ settings (tương thích Phase 1)
	// TODO: Phase 2A2 sẽ register providers khi khởi động → bỏ fallback này
	if err != nil {
		apiKey, _ := r.settings.Get(claims.UserID, "anthropic_api_key")
		if apiKey != "" {
			prov = providers.NewAnthropicProvider(apiKey)
		} else {
			return nil, &RPCError{Code: ErrProviderNotFound.Code, Message: err.Error()}
		}
	}

	modelName := p.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	// Stream response
	start := time.Now()
	var sb strings.Builder

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	streamErr := prov.StreamComplete(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: finalInput,
		System: p.System,
	}, func(chunk string) {
		sb.WriteString(chunk)
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	latency := time.Since(start).Milliseconds()
	providerName := prov.Name()

	if streamErr != nil {
		// Thêm fallback_providers khi lỗi
		doneParams := map[string]interface{}{
			"delta":         "",
			"done":          true,
			"error":         fmt.Sprintf("%v", streamErr),
			"error_message": fmt.Sprintf("%s: %v", providerName, streamErr),
		}
		if names := r.providerRegistry.FallbackCandidateNames(providerName); len(names) > 0 {
			doneParams["fallback_providers"] = names
		}
		_ = SendNotification(conn, "stream.chunk", doneParams)

		_ = r.history.Insert(repos.InsertHistoryInput{
			UserID:    claims.UserID,
			Query:     finalInput,
			Provider:  providerName,
			Model:     modelName,
			LatencyMs: latency,
			Status:    repos.HistoryStatusError,
		})
		return nil, nil
	}

	// Done notification
	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	_ = r.history.Insert(repos.InsertHistoryInput{
		UserID:    claims.UserID,
		Query:     finalInput,
		Response:  sb.String(),
		Provider:  providerName,
		Model:     modelName,
		LatencyMs: latency,
		Status:    repos.HistoryStatusSuccess,
	})

	return nil, nil
}
