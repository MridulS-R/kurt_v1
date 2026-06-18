package think

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ChatMsg is a single turn in a multi-turn conversation.
type ChatMsg struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Provider is the single interface every LLM backend must implement.
type Provider interface {
	// ThinkStream is a one-shot call with rich shell context injected.
	ThinkStream(ctx Context, question string, w io.Writer) error
	// ChatStream sends a full conversation history and streams the next reply.
	// Used by sessions for persistent multi-turn memory.
	ChatStream(messages []ChatMsg, w io.Writer) error
}

// ProviderConfig holds everything needed to construct a Provider.
// API keys are read from env vars, never stored here directly.
type ProviderConfig struct {
	// Name selects the backend: ollama, openai, anthropic, groq, together,
	// openrouter, lmstudio, openai-compat.
	Name string
	// Model overrides the provider default.
	Model string
	// BaseURL overrides the provider's default endpoint (useful for openai-compat).
	BaseURL string
	// Host is the Ollama server address (ollama only).
	Host string
	// MaxTokens caps the response length (0 = provider default).
	MaxTokens int
}

// named provider defaults
type providerDefaults struct {
	baseURL    string
	apiKeyEnv  string
	defaultModel string
}

var knownProviders = map[string]providerDefaults{
	"openai": {
		baseURL:      "https://api.openai.com/v1",
		apiKeyEnv:    "OPENAI_API_KEY",
		defaultModel: "gpt-4o-mini",
	},
	"anthropic": {
		baseURL:      "https://api.anthropic.com",
		apiKeyEnv:    "ANTHROPIC_API_KEY",
		defaultModel: "claude-haiku-4-5-20251001",
	},
	"groq": {
		baseURL:      "https://api.groq.com/openai/v1",
		apiKeyEnv:    "GROQ_API_KEY",
		defaultModel: "llama-3.3-70b-versatile",
	},
	"together": {
		baseURL:      "https://api.together.xyz/v1",
		apiKeyEnv:    "TOGETHER_API_KEY",
		defaultModel: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
	},
	"openrouter": {
		baseURL:      "https://openrouter.ai/api/v1",
		apiKeyEnv:    "OPENROUTER_API_KEY",
		defaultModel: "openai/gpt-4o-mini",
	},
	"lmstudio": {
		baseURL:      "http://127.0.0.1:1234/v1",
		apiKeyEnv:    "",
		defaultModel: "local-model",
	},
	"openai-compat": {
		baseURL:      "",
		apiKeyEnv:    "OPENAI_API_KEY",
		defaultModel: "",
	},
}

// New constructs the right Provider for cfg.
func New(cfg ProviderConfig) (Provider, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Name))
	if name == "" || name == "ollama" {
		host := strings.TrimSpace(cfg.Host)
		if host == "" {
			host = envOr("KURT_OLLAMA_HOST", "http://127.0.0.1:11434")
		}
		model := strings.TrimSpace(cfg.Model)
		if model == "" {
			model = envOr("KURT_OLLAMA_MODEL", "qwen2.5:7b-instruct")
		}
		return &OllamaClient{Host: host, Model: model, MaxTokens: cfg.MaxTokens}, nil
	}

	if name == "anthropic" {
		d := knownProviders["anthropic"]
		model := fallback(cfg.Model, envOr("ANTHROPIC_MODEL", d.defaultModel))
		apiKey := envOr(d.apiKeyEnv, "")
		if apiKey == "" {
			return nil, fmt.Errorf("anthropic: set %s environment variable", d.apiKeyEnv)
		}
		return &AnthropicProvider{APIKey: apiKey, Model: model, MaxTokens: cfg.MaxTokens}, nil
	}

	// OpenAI + all OpenAI-compatible providers (groq, together, openrouter, lmstudio, openai-compat)
	d, isKnown := knownProviders[name]
	if !isKnown {
		return nil, fmt.Errorf("unknown provider %q — valid: ollama, openai, anthropic, groq, together, openrouter, lmstudio, openai-compat", name)
	}

	baseURL := fallback(cfg.BaseURL, d.baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("%s: set --base-url or KURT_BASE_URL", name)
	}
	model := fallback(cfg.Model, "")
	if model == "" && d.apiKeyEnv != "" {
		model = envOr(strings.ToUpper(strings.ReplaceAll(name, "-", "_"))+"_MODEL", d.defaultModel)
	}
	if model == "" {
		model = d.defaultModel
	}
	apiKey := ""
	if d.apiKeyEnv != "" {
		apiKey = envOr(d.apiKeyEnv, "")
		if apiKey == "" {
			// Also accept OPENAI_API_KEY as a universal fallback for compat providers
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" && name != "lmstudio" {
			return nil, fmt.Errorf("%s: set %s environment variable", name, d.apiKeyEnv)
		}
	}
	return &OpenAIProvider{BaseURL: baseURL, APIKey: apiKey, Model: model, ProviderName: name, MaxTokens: cfg.MaxTokens}, nil
}

func envOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func fallback(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}
