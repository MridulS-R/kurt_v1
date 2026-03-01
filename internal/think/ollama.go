package think

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	Host  string
	Model string
}

type generateReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResp struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c OllamaClient) Think(ctx Context, question string) (string, error) {
	host := strings.TrimRight(strings.TrimSpace(c.Host), "/")
	if host == "" {
		host = "http://127.0.0.1:11434"
	}
	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = "qwen2.5:7b-instruct"
	}

	prompt := BuildPrompt(ctx, question)
	body, _ := json.Marshal(generateReq{Model: model, Prompt: prompt, Stream: false})

	req, err := http.NewRequest("POST", host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	hc := &http.Client{Timeout: 60 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama http %d: %s", resp.StatusCode, string(b))
	}

	var gr generateResp
	if err := json.Unmarshal(b, &gr); err != nil {
		return "", fmt.Errorf("ollama bad json: %w (%s)", err, string(b))
	}
	return strings.TrimSpace(gr.Response), nil
}
