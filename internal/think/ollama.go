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

func (c OllamaClient) host() string {
	h := strings.TrimRight(strings.TrimSpace(c.Host), "/")
	if h == "" {
		return "http://127.0.0.1:11434"
	}
	return h
}

func (c OllamaClient) model() string {
	m := strings.TrimSpace(c.Model)
	if m == "" {
		return "qwen2.5:7b-instruct"
	}
	return m
}

// Think sends a prompt and returns the full response (non-streaming).
func (c OllamaClient) Think(ctx Context, question string) (string, error) {
	prompt := BuildPrompt(ctx, question)
	body, _ := json.Marshal(generateReq{Model: c.model(), Prompt: prompt, Stream: false})

	req, err := http.NewRequest("POST", c.host()+"/api/generate", bytes.NewReader(body))
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

// ChatStream sends a full message history and streams the assistant reply.
// Uses /api/chat which natively supports multi-turn conversations.
func (c OllamaClient) ChatStream(messages []ChatMsg, w io.Writer) error {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type req struct {
		Model    string `json:"model"`
		Messages []msg  `json:"messages"`
		Stream   bool   `json:"stream"`
	}
	var msgs []msg
	for _, m := range messages {
		msgs = append(msgs, msg{Role: m.Role, Content: m.Content})
	}
	body, _ := json.Marshal(req{Model: c.model(), Messages: msgs, Stream: true})

	httpReq, err := http.NewRequest("POST", c.host()+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama http %d: %s", resp.StatusCode, b)
	}

	type chatResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cr chatResp
		if err := json.Unmarshal([]byte(line), &cr); err != nil {
			continue
		}
		fmt.Fprint(w, cr.Message.Content)
		if cr.Done {
			break
		}
	}
	fmt.Fprintln(w)
	return scanner.Err()
}

// ThinkStream streams tokens to w (implements Provider).
func (c OllamaClient) ThinkStream(ctx Context, question string, w io.Writer) error {
	prompt := BuildPrompt(ctx, question)
	body, _ := json.Marshal(generateReq{Model: c.model(), Prompt: prompt, Stream: true})

	req, err := http.NewRequest("POST", c.host()+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama http %d: %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var gr generateResp
		if err := json.Unmarshal([]byte(line), &gr); err != nil {
			continue
		}
		fmt.Fprint(w, gr.Response)
		if gr.Done {
			break
		}
	}
	fmt.Fprintln(w)
	return scanner.Err()
}
