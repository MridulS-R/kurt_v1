package think

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kurt_v1/internal/cost"
)

// AnthropicProvider connects to the Anthropic Messages API.
type AnthropicProvider struct {
	APIKey    string
	Model     string
	MaxTokens int // 0 = use default (2048)
}

const anthropicBaseURL = "https://api.anthropic.com"
const anthropicVersion = "2023-06-01"

type anthRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []oaiMessage `json:"messages"` // same role/content shape
	Stream    bool         `json:"stream"`
}

type anthStreamData struct {
	Type    string `json:"type"`
	Delta   struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Message struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ChatStream sends a full conversation history and streams the next reply.
func (p *AnthropicProvider) ChatStream(messages []ChatMsg, w io.Writer) error {
	var msgs []oaiMessage
	for _, m := range messages {
		// Anthropic doesn't accept "system" in messages array — skip, handled separately.
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, oaiMessage{Role: m.Role, Content: m.Content})
	}
	return p.doStream(msgs, w)
}

func (p *AnthropicProvider) ThinkStream(ctx Context, question string, w io.Writer) error {
	prompt := BuildPrompt(ctx, question)
	return p.doStream([]oaiMessage{{Role: "user", Content: prompt}}, w)
}

func (p *AnthropicProvider) doStream(msgs []oaiMessage, w io.Writer) error {
	maxTok := p.MaxTokens
	if maxTok <= 0 {
		maxTok = 2048
	}
	reqBody, _ := json.Marshal(anthRequest{
		Model:     p.Model,
		MaxTokens: maxTok,
		Messages:  msgs,
		Stream:    true,
	})

	req, err := http.NewRequest("POST", anthropicBaseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}

	// Anthropic SSE: interleaved "event:" and "data:" lines.
	scanner := bufio.NewScanner(resp.Body)
	var lastEvent string
	var inputTokens, outputTokens int
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			lastEvent = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		if lastEvent == "message_stop" {
			break
		}
		payload := strings.TrimPrefix(line, "data: ")
		var d anthStreamData
		if json.Unmarshal([]byte(payload), &d) != nil {
			continue
		}
		switch lastEvent {
		case "message_start":
			inputTokens = d.Message.Usage.InputTokens
		case "message_delta":
			outputTokens = d.Usage.OutputTokens
		case "content_block_delta":
			if d.Delta.Type == "text_delta" {
				fmt.Fprint(w, d.Delta.Text)
			}
		}
	}
	fmt.Fprintln(w)
	cost.Log("anthropic", p.Model, inputTokens, outputTokens)
	return scanner.Err()
}
