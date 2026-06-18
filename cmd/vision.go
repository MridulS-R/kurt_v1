package cmd

import (
	"bytes"
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
)

func visionCmd() *cobra.Command {
	var provider, model, baseURL, host string

	c := &cobra.Command{
		Use:   "vision <image> [question]",
		Short: "Ask an LLM about an image",
		Long: `Send an image to a vision-capable model and ask a question about it.

Supported providers: anthropic (claude-3+), openai (gpt-4o), ollama (llava)

Examples:
  kurt vision screenshot.png "what errors do you see?"
  kurt vision diagram.jpg "explain this architecture"
  kurt vision chart.png "summarize the trend"
  kurt vision model.png --provider anthropic "describe in detail"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imagePath := args[0]
			question := "Describe this image in detail."
			if len(args) > 1 {
				question = strings.Join(args[1:], " ")
			}

			imgData, mimeType, err := loadImage(imagePath)
			if err != nil {
				return fmt.Errorf("loading image: %w", err)
			}

			cfg, _, _ := config.Load()
			providerName := firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama")
			resolvedModel := firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model)
			resolvedBaseURL := firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL)
			resolvedHost := firstOf(host, cfg.Think.Host)

			switch providerName {
			case "anthropic":
				return visionAnthropic(imgData, mimeType, question, resolvedModel, os.Getenv("ANTHROPIC_API_KEY"))
			case "ollama":
				return visionOllama(imgData, question, resolvedModel, resolvedHost)
			default:
				return visionOpenAI(imgData, mimeType, question, resolvedModel, resolvedBaseURL, os.Getenv("OPENAI_API_KEY"), providerName)
			}
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "Vision provider (anthropic/openai/ollama)")
	c.Flags().StringVar(&model, "model", "", "Model override")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	c.Flags().StringVar(&host, "host", "", "Ollama host override")
	return c
}

// ── Anthropic vision ──────────────────────────────────────────────────────────

func visionAnthropic(imgData []byte, mimeType, question, model, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	if model == "" {
		model = "claude-opus-4-5-20251101"
	}

	req := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"stream":     true,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": mimeType,
							"data":       base64.StdEncoding.EncodeToString(imgData),
						},
					},
					{"type": "text", "text": question},
				},
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic http %d: %s", resp.StatusCode, b)
	}

	scanner := bufio.NewScanner(resp.Body)
	var lastEvent string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			lastEvent = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") || lastEvent != "content_block_delta" {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var d struct {
			Delta struct{ Type, Text string } `json:"delta"`
		}
		if json.Unmarshal([]byte(payload), &d) == nil && d.Delta.Type == "text_delta" {
			fmt.Fprint(os.Stdout, d.Delta.Text)
		}
	}
	fmt.Fprintln(os.Stdout)
	return scanner.Err()
}

// ── OpenAI vision ─────────────────────────────────────────────────────────────

func visionOpenAI(imgData []byte, mimeType, question, model, baseURL, apiKey, providerName string) error {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o"
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData))
	req := map[string]interface{}{
		"model":  model,
		"stream": true,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "image_url", "image_url": map[string]string{"url": dataURL}},
					{"type": "text", "text": question},
				},
			},
		},
	}

	body, _ := json.Marshal(req)
	base := strings.TrimRight(baseURL, "/")
	httpReq, _ := http.NewRequest("POST", base+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s http %d: %s", providerName, resp.StatusCode, b)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct{ Content string } `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(payload), &chunk) == nil {
			for _, c := range chunk.Choices {
				fmt.Fprint(os.Stdout, c.Delta.Content)
			}
		}
	}
	fmt.Fprintln(os.Stdout)
	return scanner.Err()
}

// ── Ollama vision ─────────────────────────────────────────────────────────────

func visionOllama(imgData []byte, question, model, host string) error {
	if host == "" {
		host = envOr("KURT_OLLAMA_HOST", "http://127.0.0.1:11434")
	}
	if model == "" {
		model = "llava"
	}

	req := map[string]interface{}{
		"model":  model,
		"prompt": question,
		"stream": true,
		"images": []string{base64.StdEncoding.EncodeToString(imgData)},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(strings.TrimRight(host, "/")+"/api/generate",
		"application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ollama vision: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama http %d: %s", resp.StatusCode, b)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var d struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if json.Unmarshal([]byte(line), &d) == nil {
			fmt.Fprint(os.Stdout, d.Response)
			if d.Done {
				break
			}
		}
	}
	fmt.Fprintln(os.Stdout)
	return scanner.Err()
}

// ── image loading ─────────────────────────────────────────────────────────────

func loadImage(path string) ([]byte, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, 20*1024*1024)) // 20MB cap
	if err != nil {
		return nil, "", err
	}

	mime := extToMIME(filepath.Ext(strings.ToLower(path)))
	if mime == "" {
		return nil, "", fmt.Errorf("unsupported image format (supported: jpg, png, gif, webp)")
	}
	return data, mime, nil
}

func extToMIME(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	return ""
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
