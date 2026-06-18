package think

import (
	"os"
	"testing"
)

func TestNew_ollama_default(t *testing.T) {
	p, err := New(ProviderConfig{Name: "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*OllamaClient); !ok {
		t.Fatalf("expected *OllamaClient, got %T", p)
	}
}

func TestNew_empty_defaults_to_ollama(t *testing.T) {
	p, err := New(ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*OllamaClient); !ok {
		t.Fatalf("expected *OllamaClient for empty config, got %T", p)
	}
}

func TestNew_anthropic_missingKey(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")
	_, err := New(ProviderConfig{Name: "anthropic"})
	if err == nil {
		t.Fatal("expected error when ANTHROPIC_API_KEY not set")
	}
}

func TestNew_anthropic_withKey(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	p, err := New(ProviderConfig{Name: "anthropic"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*AnthropicProvider); !ok {
		t.Fatalf("expected *AnthropicProvider, got %T", p)
	}
}

func TestNew_openai_missingKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	_, err := New(ProviderConfig{Name: "openai"})
	if err == nil {
		t.Fatal("expected error when OPENAI_API_KEY not set")
	}
}

func TestNew_openai_withKey(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	p, err := New(ProviderConfig{Name: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	ap, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected *OpenAIProvider, got %T", p)
	}
	if ap.ProviderName != "openai" {
		t.Errorf("ProviderName: got %q, want openai", ap.ProviderName)
	}
}

func TestNew_unknownProvider(t *testing.T) {
	_, err := New(ProviderConfig{Name: "nonexistent-provider"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNew_maxTokens_anthropic(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	p, err := New(ProviderConfig{Name: "anthropic", MaxTokens: 512})
	if err != nil {
		t.Fatal(err)
	}
	ap := p.(*AnthropicProvider)
	if ap.MaxTokens != 512 {
		t.Errorf("MaxTokens: got %d, want 512", ap.MaxTokens)
	}
}

func TestNew_maxTokens_openai(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	p, err := New(ProviderConfig{Name: "openai", MaxTokens: 256})
	if err != nil {
		t.Fatal(err)
	}
	op := p.(*OpenAIProvider)
	if op.MaxTokens != 256 {
		t.Errorf("MaxTokens: got %d, want 256", op.MaxTokens)
	}
}

func TestNew_maxTokens_ollama(t *testing.T) {
	p, err := New(ProviderConfig{Name: "ollama", MaxTokens: 1024})
	if err != nil {
		t.Fatal(err)
	}
	oc := p.(*OllamaClient)
	if oc.MaxTokens != 1024 {
		t.Errorf("MaxTokens: got %d, want 1024", oc.MaxTokens)
	}
}

func TestNew_groq(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "test-key")
	defer os.Unsetenv("GROQ_API_KEY")

	p, err := New(ProviderConfig{Name: "groq"})
	if err != nil {
		t.Fatal(err)
	}
	op, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected *OpenAIProvider for groq, got %T", p)
	}
	if op.ProviderName != "groq" {
		t.Errorf("ProviderName: got %q, want groq", op.ProviderName)
	}
}

func TestNew_lmstudio_noKeyRequired(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	_, err := New(ProviderConfig{Name: "lmstudio"})
	if err != nil {
		t.Fatalf("lmstudio should not require API key: %v", err)
	}
}

func TestNew_modelOverride(t *testing.T) {
	p, err := New(ProviderConfig{Name: "ollama", Model: "mistral:7b"})
	if err != nil {
		t.Fatal(err)
	}
	oc := p.(*OllamaClient)
	if oc.Model != "mistral:7b" {
		t.Errorf("model: got %q, want mistral:7b", oc.Model)
	}
}
