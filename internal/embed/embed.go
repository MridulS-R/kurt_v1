package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// Embedder converts text to a dense float vector.
type Embedder interface {
	Embed(text string) ([]float32, error)
}

// ── Ollama ────────────────────────────────────────────────────────────────────

type OllamaEmbedder struct {
	Host  string
	Model string
}

func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	host := strings.TrimRight(e.Host, "/")
	if host == "" {
		host = envOr("KURT_OLLAMA_HOST", "http://127.0.0.1:11434")
	}
	model := e.Model
	if model == "" {
		model = "nomic-embed-text"
	}

	body, _ := json.Marshal(map[string]string{"model": model, "prompt": text})
	resp, err := http.Post(host+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama embed http %d", resp.StatusCode)
	}

	var out struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embedding, nil
}

// ── OpenAI-compatible ─────────────────────────────────────────────────────────

type OpenAIEmbedder struct {
	BaseURL string
	APIKey  string
	Model   string
}

func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	base := strings.TrimRight(e.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := e.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	apiKey := e.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	body, _ := json.Marshal(map[string]interface{}{"model": model, "input": text})
	req, _ := http.NewRequest("POST", base+"/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	hc := &http.Client{Timeout: 30 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai embed http %d", resp.StatusCode)
	}

	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}
	return out.Data[0].Embedding, nil
}

// ── similarity ────────────────────────────────────────────────────────────────

// Cosine returns cosine similarity in [-1, 1].
func Cosine(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
