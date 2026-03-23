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

const defaultCopilotBaseURL = "https://api.githubcopilot.com"

// CopilotProvider gọi GitHub Copilot Chat API
type CopilotProvider struct {
	token   string
	baseURL string
	client  *http.Client
}

// NewCopilotProvider tạo provider mới
func NewCopilotProvider(token string) *CopilotProvider {
	return NewCopilotProviderWithBaseURL(token, defaultCopilotBaseURL)
}

// NewCopilotProviderWithBaseURL tạo provider với custom URL (dùng cho test)
func NewCopilotProviderWithBaseURL(token, baseURL string) *CopilotProvider {
	return &CopilotProvider{
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *CopilotProvider) Name() string         { return "copilot" }
func (p *CopilotProvider) DisplayName() string  { return "GitHub Copilot" }
func (p *CopilotProvider) Aliases() []string    { return []string{"copilot", "gh", "github"} }
func (p *CopilotProvider) GetAuthType() AuthType { return AuthCLIToken }

// StreamComplete gọi Copilot /chat/completions (OpenAI-compatible)
func (p *CopilotProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	modelName := req.Model
	if modelName == "" {
		modelName = "gpt-4o"
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
	httpReq.Header.Set("Authorization", "Bearer "+p.token)
	httpReq.Header.Set("Editor-Version", "open-prompt/0.1.0")
	httpReq.Header.Set("Copilot-Integration-Id", "open-prompt")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("copilot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("copilot API error %d: %s", resp.StatusCode, string(respBody))
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

// Validate kiểm tra token hợp lệ
func (p *CopilotProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Editor-Version", "open-prompt/0.1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("copilot validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("copilot validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs
func (p *CopilotProvider) Models(_ context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4o-mini", "o1-mini"}, nil
}
