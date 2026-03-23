package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOllamaBaseURL = "http://localhost:11434"

// OllamaProvider gọi Ollama local API với streaming NDJSON
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllamaProvider tạo provider mới. baseURL="" → dùng http://localhost:11434
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// StreamComplete gọi Ollama /api/chat với streaming NDJSON
func (p *OllamaProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}
	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}
	if req.System != "" {
		messages = append([]map[string]string{{"role": "system", "content": req.System}}, messages...)
	}

	body := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
		"options": map[string]interface{}{
			"num_predict": req.MaxTokens,
			"temperature": req.Temperature,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("kết nối Ollama thất bại (Ollama có đang chạy không?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse NDJSON (Newline Delimited JSON) response của Ollama
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Done {
			break
		}
		if chunk.Message.Content != "" {
			onChunk(chunk.Message.Content)
		}
	}
	return scanner.Err()
}

// Name trả về tên chính
func (p *OllamaProvider) Name() string { return "ollama" }

// DisplayName trả về tên hiển thị
func (p *OllamaProvider) DisplayName() string { return "Ollama (Local)" }

// Aliases trả về tất cả alias
func (p *OllamaProvider) Aliases() []string {
	return []string{"ollama", "local", "llama"}
}

// GetAuthType trả về loại xác thực
func (p *OllamaProvider) GetAuthType() AuthType { return AuthNone }

// Validate kiểm tra Ollama có đang chạy
func (p *OllamaProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách models từ Ollama API (dynamic)
func (p *OllamaProvider) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama models: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	names := make([]string, len(result.Models))
	for i, m := range result.Models {
		names[i] = m.Name
	}
	return names, nil
}
