package cost

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kurt_v1/internal/history"
)

// UsageEntry records a single LLM API call.
type UsageEntry struct {
	At           time.Time `json:"at"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
}

var mu sync.Mutex

func filePath() (string, error) {
	d, err := history.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cost.jsonl"), nil
}

// Log appends a usage entry. Silently no-ops on write errors (best-effort tracking).
func Log(provider, model string, inputTokens, outputTokens int) {
	if inputTokens == 0 && outputTokens == 0 {
		return
	}
	in, out := modelPrice(model)
	costUSD := float64(inputTokens)/1e6*in + float64(outputTokens)/1e6*out

	mu.Lock()
	defer mu.Unlock()

	path, err := filePath()
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	e := UsageEntry{
		At: time.Now(), Provider: provider, Model: model,
		InputTokens: inputTokens, OutputTokens: outputTokens, CostUSD: costUSD,
	}
	b, _ := json.Marshal(e)
	_, _ = f.Write(append(b, '\n'))
}

// Since returns all usage entries after t (zero time = all entries).
func Since(t time.Time) ([]UsageEntry, error) {
	path, err := filePath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []UsageEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e UsageEntry
		if json.Unmarshal([]byte(line), &e) == nil && (t.IsZero() || e.At.After(t)) {
			out = append(out, e)
		}
	}
	return out, scanner.Err()
}

// ClearAll removes the cost log.
func ClearAll() error {
	path, err := filePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Summary aggregates usage entries.
type Summary struct {
	Calls        int
	InputTokens  int
	OutputTokens int
	TotalCostUSD float64
}

func Summarize(entries []UsageEntry) Summary {
	var s Summary
	for _, e := range entries {
		s.Calls++
		s.InputTokens += e.InputTokens
		s.OutputTokens += e.OutputTokens
		s.TotalCostUSD += e.CostUSD
	}
	return s
}

// FormatCost returns a human-readable cost string.
func FormatCost(usd float64) string {
	if usd == 0 {
		return "$0.000"
	}
	if usd < 0.0001 {
		return fmt.Sprintf("$%.6f", usd)
	}
	if usd < 0.01 {
		return fmt.Sprintf("$%.4f", usd)
	}
	return fmt.Sprintf("$%.3f", usd)
}

// ByModel groups entries by model and returns a map.
func ByModel(entries []UsageEntry) map[string]Summary {
	m := map[string]Summary{}
	for _, e := range entries {
		key := e.Provider + "/" + e.Model
		s := m[key]
		s.Calls++
		s.InputTokens += e.InputTokens
		s.OutputTokens += e.OutputTokens
		s.TotalCostUSD += e.CostUSD
		m[key] = s
	}
	return m
}

// ByDay groups entries by date string (YYYY-MM-DD in local time).
func ByDay(entries []UsageEntry) map[string]Summary {
	m := map[string]Summary{}
	for _, e := range entries {
		key := e.At.Local().Format("2006-01-02")
		s := m[key]
		s.Calls++
		s.InputTokens += e.InputTokens
		s.OutputTokens += e.OutputTokens
		s.TotalCostUSD += e.CostUSD
		m[key] = s
	}
	return m
}

// ── pricing ───────────────────────────────────────────────────────────────────

type priceEntry struct {
	prefix  string
	in, out float64
}

// Ordered longest-prefix-first so gpt-4o-mini matches before gpt-4o.
var pricingTable = []priceEntry{
	// Anthropic
	{"claude-haiku-4-5", 0.80, 4.00},
	{"claude-3-5-haiku", 0.80, 4.00},
	{"claude-3-haiku", 0.25, 1.25},
	{"claude-sonnet-4", 3.00, 15.00},
	{"claude-3-5-sonnet", 3.00, 15.00},
	{"claude-3-sonnet", 3.00, 15.00},
	{"claude-opus-4", 15.00, 75.00},
	{"claude-3-opus", 15.00, 75.00},
	// OpenAI (longer prefixes first)
	{"gpt-4o-mini", 0.15, 0.60},
	{"gpt-4o", 2.50, 10.00},
	{"gpt-4-turbo", 10.00, 30.00},
	{"gpt-4", 30.00, 60.00},
	{"gpt-3.5-turbo", 0.50, 1.50},
	{"o1-mini", 1.10, 4.40},
	{"o1", 15.00, 60.00},
	{"o3-mini", 1.10, 4.40},
	// Groq
	{"llama-3.3-70b", 0.59, 0.79},
	{"llama-3.1-70b", 0.59, 0.79},
	{"llama-3.1-8b", 0.05, 0.08},
	{"mixtral-8x7b", 0.24, 0.24},
	// Together
	{"meta-llama/llama-3.3-70b", 0.88, 0.88},
	{"meta-llama/llama-3.1-70b", 0.88, 0.88},
	{"meta-llama/llama-3.1-8b", 0.18, 0.18},
	// OpenRouter (common)
	{"openai/gpt-4o-mini", 0.15, 0.60},
	{"openai/gpt-4o", 2.50, 10.00},
	// Ollama is local/free → 0, 0
}

func modelPrice(model string) (in, out float64) {
	m := strings.ToLower(strings.TrimSpace(model))
	for _, e := range pricingTable {
		if strings.HasPrefix(m, e.prefix) {
			return e.in, e.out
		}
	}
	return 0, 0
}
