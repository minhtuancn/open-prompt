package api

import (
	"encoding/json"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// PromptExport là format export/import cho prompts
type PromptExport struct {
	Version string                   `json:"version"`
	Prompts []map[string]interface{} `json:"prompts"`
}

// handlePromptsExport export tất cả prompts thành JSON
func (r *Router) handlePromptsExport(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	prompts, err := r.prompts.List(claims.UserID, "")
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("list prompts: %v", err)}
	}

	var exported []map[string]interface{}
	for _, p := range prompts {
		exported = append(exported, map[string]interface{}{
			"title":      p.Title,
			"content":    p.Content,
			"category":   p.Category,
			"tags":       p.Tags,
			"is_slash":   p.IsSlash,
			"slash_name": p.SlashName,
		})
	}

	return PromptExport{
		Version: "1.0",
		Prompts: exported,
	}, nil
}

// handlePromptsImport import prompts từ JSON
func (r *Router) handlePromptsImport(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string          `json:"token"`
		Data  json.RawMessage `json:"data"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	var export PromptExport
	if err := json.Unmarshal(p.Data, &export); err != nil {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: fmt.Sprintf("invalid JSON: %v", err)}
	}

	imported := 0
	for _, prompt := range export.Prompts {
		title, _ := prompt["title"].(string)
		content, _ := prompt["content"].(string)
		if title == "" || content == "" {
			continue
		}
		category, _ := prompt["category"].(string)
		tags, _ := prompt["tags"].(string)
		slashName, _ := prompt["slash_name"].(string)
		isSlash := slashName != ""

		_, err := r.prompts.Create(repos.CreatePromptInput{
			UserID:    claims.UserID,
			Title:     title,
			Content:   content,
			Category:  category,
			Tags:      tags,
			IsSlash:   isSlash,
			SlashName: slashName,
		})
		if err != nil {
			continue
		}
		imported++
	}

	return map[string]interface{}{"imported": imported}, nil
}
