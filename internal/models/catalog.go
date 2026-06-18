package models

import "strings"

// CatalogEntry describes a model available on Ollama.
type CatalogEntry struct {
	Name        string
	Tag         string   // ollama pull tag
	ParamB      float64  // billions of parameters
	RAMgb       float64  // typical RAM usage (Q4_K_M)
	Speed       string   // "blazing" | "very fast" | "fast" | "balanced" | "slow"
	UseCases    []string // "chat" | "coding" | "reasoning" | "math" | "multilingual"
	Description string
}

// Catalog is the built-in curated list of popular Ollama-compatible models.
var Catalog = []CatalogEntry{
	// ── Tiny / fast ──────────────────────────────────────────────────────────
	{"Llama 3.2", "llama3.2:1b", 1, 0.8, "blazing", ss("chat"), "Meta's 1B model — very fast, minimal RAM"},
	{"Phi-3.5 Mini", "phi3.5:3.8b", 3.8, 2.2, "very fast", ss("chat", "reasoning"), "Microsoft's compact model, punches above its weight"},
	{"Llama 3.2", "llama3.2:3b", 3, 2.0, "very fast", ss("chat", "quick tasks"), "Meta's 3B — great balance of speed and quality"},
	{"Gemma 2", "gemma2:2b", 2, 1.6, "very fast", ss("chat"), "Google's smallest Gemma, fast and capable"},
	// ── 7–9B sweet spot ──────────────────────────────────────────────────────
	{"Qwen 2.5", "qwen2.5:7b-instruct", 7, 4.7, "fast", ss("chat", "coding", "multilingual"), "Excellent for coding and multilingual tasks — runs on 8GB"},
	{"Llama 3.1", "llama3.1:8b", 8, 4.9, "fast", ss("chat", "coding"), "Meta's solid 8B — good all-rounder"},
	{"Mistral", "mistral:7b", 7, 4.1, "fast", ss("chat", "reasoning"), "Efficient European model, great instruction following"},
	{"DeepSeek R1", "deepseek-r1:7b", 7, 4.7, "fast", ss("reasoning", "math"), "Strong reasoning chain-of-thought model"},
	{"Gemma 2", "gemma2:9b", 9, 5.5, "fast", ss("chat", "coding"), "Google's 9B — very capable for its size"},
	{"Qwen 2.5 Coder", "qwen2.5-coder:7b", 7, 4.7, "fast", ss("coding"), "Specialized code generation, top of its class"},
	{"CodeLlama", "codellama:7b", 7, 3.8, "fast", ss("coding"), "Meta's dedicated code model"},
	// ── 12–16B balanced ──────────────────────────────────────────────────────
	{"Mistral Nemo", "mistral-nemo:12b", 12, 7.1, "balanced", ss("chat", "multilingual"), "128k context, strong multilingual support"},
	{"DeepSeek R1", "deepseek-r1:14b", 14, 9.0, "balanced", ss("reasoning", "math", "coding"), "Better reasoning quality than the 7B"},
	{"Qwen 2.5 Coder", "qwen2.5-coder:14b", 14, 9.0, "balanced", ss("coding"), "Top-tier code model for 16GB systems"},
	{"Phi-4", "phi4:14b", 14, 9.0, "balanced", ss("reasoning", "math", "coding"), "Microsoft's latest — surprisingly strong"},
	// ── 27–32B high quality ───────────────────────────────────────────────────
	{"Gemma 2", "gemma2:27b", 27, 16.0, "slow", ss("chat", "coding", "reasoning"), "Google's best open model, needs 24GB+"},
	{"DeepSeek Coder V2", "deepseek-coder-v2:16b", 16, 9.0, "balanced", ss("coding"), "Excellent for complex coding tasks"},
	{"Qwen 2.5", "qwen2.5:32b", 32, 20.0, "slow", ss("chat", "coding", "reasoning", "multilingual"), "Qwen's largest practical model"},
	// ── 70B top quality ───────────────────────────────────────────────────────
	{"Llama 3.1", "llama3.1:70b", 70, 40.0, "slow", ss("chat", "reasoning", "coding"), "Meta's best open model — needs 48GB+"},
	{"DeepSeek R1", "deepseek-r1:70b", 70, 40.0, "slow", ss("reasoning", "math"), "Top reasoning quality — needs 48GB+"},
	{"Qwen 2.5", "qwen2.5:72b", 72, 43.0, "slow", ss("chat", "coding", "reasoning"), "Qwen's flagship — near frontier quality"},
	// ── MoE ───────────────────────────────────────────────────────────────────
	{"Mixtral", "mixtral:8x7b", 47, 26.0, "balanced", ss("reasoning", "multilingual"), "MoE — high quality, needs 32GB+"},
}

// Fit classifies how well a model fits in available RAM.
type Fit int

const (
	FitGreat  Fit = iota // ≤ 60% of total RAM
	FitGood              // ≤ 80%
	FitTight             // ≤ 95%
	FitTooBig            // > 95%
)

func (f Fit) Label() string {
	switch f {
	case FitGreat:
		return "✓"
	case FitGood:
		return "✓"
	case FitTight:
		return "⚡"
	default:
		return "✗"
	}
}

func (f Fit) Color() string {
	switch f {
	case FitGreat, FitGood:
		return "\x1b[32m" // green
	case FitTight:
		return "\x1b[33m" // yellow
	default:
		return "\x1b[31m" // red
	}
}

// ModelFit returns how well the model fits on the given system.
func ModelFit(e CatalogEntry, sys SysInfo) Fit {
	ram := sys.TotalRAMGB
	// Apple Silicon uses unified memory — Ollama can access all of it.
	// Leave ~2GB headroom for OS.
	usable := ram - 2.0
	if usable < 0 {
		usable = ram * 0.7
	}
	ratio := e.RAMgb / usable
	switch {
	case ratio <= 0.60:
		return FitGreat
	case ratio <= 0.80:
		return FitGood
	case ratio <= 0.95:
		return FitTight
	default:
		return FitTooBig
	}
}

// Search filters the catalog by name, tag, or use case.
func Search(query string) []CatalogEntry {
	if query == "" {
		return Catalog
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []CatalogEntry
	for _, e := range Catalog {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Tag), q) ||
			strings.Contains(strings.ToLower(e.Description), q) ||
			containsUseCase(e.UseCases, q) {
			out = append(out, e)
		}
	}
	return out
}

func containsUseCase(usecases []string, q string) bool {
	for _, u := range usecases {
		if strings.Contains(strings.ToLower(u), q) {
			return true
		}
	}
	return false
}

func ss(s ...string) []string { return s }
