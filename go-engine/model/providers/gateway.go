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

// GatewayPreset là template cho các gateway phổ biến
type GatewayPreset struct {
	Name         string
	DisplayName  string
	BaseURL      string
	DefaultModel string
}

// GatewayPresets trả về danh sách preset templates
func GatewayPresets() []GatewayPreset {
	return []GatewayPreset{
		{Name: "ollama", DisplayName: "Ollama (Local)", BaseURL: "http://localhost:11434/v1", DefaultModel: "llama3.2"},
		{Name: "litellm", DisplayName: "LiteLLM", BaseURL: "http://localhost:4000/v1", DefaultModel: "gpt-4o"},
		{Name: "openrouter", DisplayName: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", DefaultModel: "openai/gpt-4o"},
		{Name: "vllm", DisplayName: "vLLM", BaseURL: "http://localhost:8000/v1", DefaultModel: ""},
	}
}

// GatewayProvider gọi bất kỳ server nào tương thích OpenAI API
type GatewayProvider struct {
	name         string
	displayName  string
	baseURL      string
	apiKey       string
	defaultModel string
	aliases      []string
	client       *http.Client
}

// NewGatewayProvider tạo gateway provider mới
func NewGatewayProvider(name, displayName, baseURL, apiKey, defaultModel string, aliases []string) *GatewayProvider {
	if aliases == nil {
		aliases = []string{name}
	}
	return &GatewayProvider{
		name:         name,
		displayName:  displayName,
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiKey:       apiKey,
		defaultModel: defaultModel,
		aliases:      aliases,
		client:       &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GatewayProvider) Name() string        { return p.name }
func (p *GatewayProvider) DisplayName() string { return p.displayName }
func (p *GatewayProvider) Aliases() []string   { return p.aliases }
func (p *GatewayProvider) GetAuthType() AuthType {
	if p.apiKey != "" {
		return AuthAPIKey
	}
	return AuthNone
}

// StreamComplete gọi /chat/completions với SSE (chuẩn OpenAI)
func (p *GatewayProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	modelName := req.Model
	if modelName == "" {
		modelName = p.defaultModel
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}
	if req.System != "" {
		messages = append([]map[string]string{{"role": "system", "content": req.System}}, messages...)
	}

	body := map[string]interface{}{
		"model":       modelName,
		"messages":    messages,
		"stream":      true,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gateway %s request: %w", p.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway %s API error %d: %s", p.name, resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}

// Validate kiểm tra gateway có đang chạy
func (p *GatewayProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway %s validate: %w", p.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway %s validate: HTTP %d", p.name, resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách models từ /models endpoint
func (p *GatewayProvider) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway %s models: %w", p.name, err)
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	names := make([]string, len(result.Data))
	for i, m := range result.Data {
		names[i] = m.ID
	}
	return names, nil
}
