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

const (
	anthropicBaseURL = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
)

// CompletionRequest là request gửi đến Anthropic
type CompletionRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
}

// AnthropicProvider gọi Anthropic Messages API với streaming
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicProvider tạo provider mới dùng base URL mặc định
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return NewAnthropicProviderWithBaseURL(apiKey, anthropicBaseURL)
}

// NewAnthropicProviderWithBaseURL tạo provider với base URL tuỳ chỉnh (dùng cho test)
func NewAnthropicProviderWithBaseURL(apiKey, baseURL string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// StreamComplete gọi Anthropic API với streaming SSE
func (p *AnthropicProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}
	if req.System != "" {
		body["system"] = req.System
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Server-Sent Events
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB buffer cho SSE events lớn
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			onChunk(event.Delta.Text)
		}
	}
	return scanner.Err()
}

// Name trả về tên chính (khớp DB key)
func (p *AnthropicProvider) Name() string { return "anthropic" }

// DisplayName trả về tên hiển thị
func (p *AnthropicProvider) DisplayName() string { return "Anthropic (Claude)" }

// Aliases trả về tất cả alias cho @mention routing
func (p *AnthropicProvider) Aliases() []string {
	return []string{"claude", "sonnet", "opus", "haiku", "anthropic"}
}

// GetAuthType trả về loại xác thực
func (p *AnthropicProvider) GetAuthType() AuthType { return AuthAPIKey }

// Validate kiểm tra API key hợp lệ
func (p *AnthropicProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anthropic validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs (hardcode — ít thay đổi)
func (p *AnthropicProvider) Models(_ context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-5-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-opus-4-5",
	}, nil
}
