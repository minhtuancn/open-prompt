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

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GeminiProvider gọi Google Gemini API với streaming SSE
type GeminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGeminiProvider tạo provider mới
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return NewGeminiProviderWithBaseURL(apiKey, defaultGeminiBaseURL)
}

// NewGeminiProviderWithBaseURL tạo provider với custom base URL (dùng cho test)
func NewGeminiProviderWithBaseURL(apiKey, baseURL string) *GeminiProvider {
	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *GeminiProvider) Name() string         { return "gemini" }
func (p *GeminiProvider) DisplayName() string  { return "Google Gemini" }
func (p *GeminiProvider) Aliases() []string    { return []string{"gemini", "google", "bard"} }
func (p *GeminiProvider) GetAuthType() AuthType { return AuthAPIKey }

// StreamComplete gọi Gemini streamGenerateContent API
func (p *GeminiProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}
	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	contents := []map[string]interface{}{
		{"role": "user", "parts": []map[string]string{{"text": req.Prompt}}},
	}
	body := map[string]interface{}{
		"contents":         contents,
		"generationConfig": map[string]interface{}{"maxOutputTokens": req.MaxTokens},
	}
	if req.System != "" {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{{"text": req.System}},
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.baseURL, modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			text := event.Candidates[0].Content.Parts[0].Text
			if text != "" {
				onChunk(text)
			}
		}
	}
	return scanner.Err()
}

// Validate kiểm tra API key
func (p *GeminiProvider) Validate(ctx context.Context) error {
	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gemini validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs
func (p *GeminiProvider) Models(_ context.Context) ([]string, error) {
	return []string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash"}, nil
}
