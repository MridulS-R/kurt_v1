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

// OpenAIProvider works with any OpenAI-compatible chat completions API.
// This covers: OpenAI, Groq, Together, OpenRouter, LM Studio, and more.
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
}

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiRequest struct {
	Model    string       `json:"model"`
	Messages []oaiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

type oaiStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatStream sends a full conversation history and streams the next reply.
func (p *OpenAIProvider) ChatStream(messages []ChatMsg, w io.Writer) error {
	var msgs []oaiMessage
	for _, m := range messages {
		msgs = append(msgs, oaiMessage{Role: m.Role, Content: m.Content})
	}
	return p.doStream(msgs, w)
}

func (p *OpenAIProvider) ThinkStream(ctx Context, question string, w io.Writer) error {
	prompt := BuildPrompt(ctx, question)
	return p.doStream([]oaiMessage{{Role: "user", Content: prompt}}, w)
}

func (p *OpenAIProvider) doStream(msgs []oaiMessage, w io.Writer) error {
	reqBody, _ := json.Marshal(oaiRequest{
		Model:    p.Model,
		Messages: msgs,
		Stream:   true,
	})

	base := strings.TrimRight(p.BaseURL, "/")
	req, err := http.NewRequest("POST", base+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", p.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}

	// OpenAI SSE format: "data: {json}\n\n", terminated by "data: [DONE]\n\n"
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
		var chunk oaiStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			fmt.Fprint(w, choice.Delta.Content)
		}
	}
	fmt.Fprintln(w)
	return scanner.Err()
}
