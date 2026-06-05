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
)

// AnthropicProvider connects to the Anthropic Messages API.
type AnthropicProvider struct {
	APIKey string
	Model  string
}

const anthropicBaseURL = "https://api.anthropic.com"
const anthropicVersion = "2023-06-01"

type anthRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []oaiMessage `json:"messages"` // same role/content shape
	Stream    bool         `json:"stream"`
}

// Anthropic SSE events we care about:
// event: content_block_delta
// data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"..."}}
// event: message_stop

type anthStreamData struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
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
	reqBody, _ := json.Marshal(anthRequest{
		Model:     p.Model,
		MaxTokens: 2048,
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
		if lastEvent != "content_block_delta" {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var d anthStreamData
		if err := json.Unmarshal([]byte(payload), &d); err != nil {
			continue
		}
		if d.Delta.Type == "text_delta" {
			fmt.Fprint(w, d.Delta.Text)
		}
	}
	fmt.Fprintln(w)
	return scanner.Err()
}
