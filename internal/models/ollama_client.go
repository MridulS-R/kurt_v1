package models

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
	Host string
}

func NewOllamaClient(host string) OllamaClient {
	if host == "" {
		host = "http://127.0.0.1:11434"
	}
	return OllamaClient{Host: strings.TrimRight(host, "/")}
}

// InstalledModel is a model returned by /api/tags.
type InstalledModel struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modified_at"`
	Details struct {
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
		Family            string `json:"family"`
	} `json:"details"`
}

func (c OllamaClient) List() ([]InstalledModel, error) {
	resp, err := http.Get(c.Host + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("ollama not running at %s — start with: ollama serve", c.Host)
	}
	defer resp.Body.Close()
	var r struct {
		Models []InstalledModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return r.Models, nil
}

func (c OllamaClient) Remove(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequest("DELETE", c.Host+"/api/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, b)
	}
	return nil
}

// Pull downloads a model and writes live progress to out.
func (c OllamaClient) Pull(name string, out io.Writer) error {
	body, _ := json.Marshal(map[string]any{"name": name, "stream": true})
	resp, err := http.Post(c.Host+"/api/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ollama not running at %s — start with: ollama serve", c.Host)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, b)
	}

	type pullMsg struct {
		Status    string `json:"status"`
		Digest    string `json:"digest"`
		Total     int64  `json:"total"`
		Completed int64  `json:"completed"`
		Error     string `json:"error"`
	}

	prevHadProgress := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 128*1024), 128*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg pullMsg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Error != "" {
			if prevHadProgress {
				fmt.Fprintln(out)
			}
			return fmt.Errorf("pull failed: %s", msg.Error)
		}

		if msg.Total > 0 {
			pct := float64(msg.Completed) / float64(msg.Total) * 100
			bar := renderBar(pct, 28)
			label := msg.Status
			if len(label) > 22 {
				label = label[:10] + "…" + label[len(label)-8:]
			}
			fmt.Fprintf(out, "\r  %-24s  %s  %5.1f%%  %s / %s   ",
				label, bar, pct,
				formatBytes(msg.Completed), formatBytes(msg.Total))
			prevHadProgress = true
		} else {
			if prevHadProgress {
				fmt.Fprintln(out)
				prevHadProgress = false
			}
			fmt.Fprintf(out, "  %s\n", msg.Status)
		}
	}
	if prevHadProgress {
		fmt.Fprintln(out)
	}
	return scanner.Err()
}

// Ping returns true if Ollama is reachable.
func (c OllamaClient) Ping() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(c.Host + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func renderBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "\x1b[32m" + strings.Repeat("█", filled) + "\x1b[90m" + strings.Repeat("░", empty) + "\x1b[0m"
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/(1<<20))
	default:
		return fmt.Sprintf("%d KB", b/1024)
	}
}

func shortDigest(d string) string {
	if len(d) > 19 {
		return d[7:19]
	}
	return d
}
