package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// GeminiModelPool defines models in priority sequence for automatic 429 failover.
var GeminiModelPool = []string{
	"gemini-3.1-flash-lite-preview",
	"gemma-4-31b-it",
	"gemma-4-26b-it",
	"gemini-2.5-flash-lite",
}

// CloudAIEngine manages optional Gemini API calls with 429 failover and batching.
type CloudAIEngine struct {
	mu         sync.RWMutex
	apiKey     string
	httpClient *http.Client
}

// NewCloudAIEngine creates a CloudAIEngine.
func NewCloudAIEngine(apiKey string) *CloudAIEngine {
	return &CloudAIEngine{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetApiKey updates the API key at runtime.
func (c *CloudAIEngine) SetApiKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKey = strings.TrimSpace(key)
}

// HasKey checks if a valid Gemini API key is configured.
func (c *CloudAIEngine) HasKey() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.apiKey) > 10
}

// BatchImageAnalysis sends up to 8 photos in a single prompt to Gemini.
func (c *CloudAIEngine) BatchImageAnalysis(ctx context.Context, filePaths []string) (map[string][]string, error) {
	c.mu.RLock()
	key := c.apiKey
	c.mu.RUnlock()

	if key == "" {
		return nil, fmt.Errorf("no Gemini API key configured")
	}

	if len(filePaths) > 8 {
		filePaths = filePaths[:8]
	}

	// Try each model in pool with automatic 429 failover
	var lastErr error
	for _, model := range GeminiModelPool {
		res, err := c.callGeminiModel(ctx, model, key, filePaths)
		if err == nil {
			return res, nil
		}
		lastErr = err
		// If 429 rate limit or 503, loop and try next model
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "quota") {
			continue
		}
		// If other critical error, return immediately
		break
	}

	return nil, fmt.Errorf("gemini batch analysis failed: %w", lastErr)
}

func (c *CloudAIEngine) callGeminiModel(ctx context.Context, model string, apiKey string, filePaths []string) (map[string][]string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	parts := []map[string]any{
		{
			"text": "Analyze each image and output a JSON map with format: {\"index_0\": [\"tag1\", \"tag2\"], ...} describing scene, objects, lighting, and detected text.",
		},
	}

	for _, path := range filePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		mime := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(path), ".png") {
			mime = "image/png"
		}
		b64 := base64.StdEncoding.EncodeToString(data)
		parts = append(parts, map[string]any{
			"inline_data": map[string]string{
				"mime_type": mime,
				"data":      b64,
			},
		})
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": parts,
			},
		},
		"generationConfig": map[string]any{
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return nil, fmt.Errorf("rate limited (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error (%d): %s", resp.StatusCode, string(b))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	outputText := geminiResp.Candidates[0].Content.Parts[0].Text
	var parsed map[string][]string
	if err := json.Unmarshal([]byte(outputText), &parsed); err != nil {
		// Try extracting JSON block
		cleanText := strings.TrimPrefix(outputText, "```json")
		cleanText = strings.TrimPrefix(cleanText, "```")
		cleanText = strings.TrimSuffix(cleanText, "```")
		_ = json.Unmarshal([]byte(cleanText), &parsed)
	}

	return parsed, nil
}
