package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_nonEmpty(t *testing.T) {
	cfg := Default()
	if cfg.Style == "" {
		t.Error("default style should not be empty")
	}
	if len(cfg.Modules.Order) == 0 {
		t.Error("default module order should not be empty")
	}
	if cfg.Perf.GitTTLms <= 0 {
		t.Error("default git TTL should be positive")
	}
}

func TestMergeDefaults_fillsNilPointers(t *testing.T) {
	cfg := MergeDefaults(Config{})
	if cfg.Style == "" {
		t.Error("style should be filled in")
	}
	if cfg.Prompt.TwoLine == nil {
		t.Error("TwoLine should not be nil after merge")
	}
	if cfg.RPrompt.Enabled == nil {
		t.Error("RPrompt.Enabled should not be nil after merge")
	}
	if cfg.RPrompt.TimeFormat == "" {
		t.Error("TimeFormat should not be empty after merge")
	}
}

func TestLoad_missingFile(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("KURT_CONFIG", filepath.Join(dir, "nonexistent.toml"))
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("missing config file should not error: %v", err)
	}
	if cfg.Style == "" {
		t.Error("should return defaults when file missing")
	}
}

func TestLoad_validTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := `style = "powerline"
[think]
provider = "anthropic"
model = "claude-sonnet-4-6"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("KURT_CONFIG", cfgPath)
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Style != "powerline" {
		t.Errorf("style: got %q, want powerline", cfg.Style)
	}
	if cfg.Think.Provider != "anthropic" {
		t.Errorf("provider: got %q", cfg.Think.Provider)
	}
	if cfg.Think.Model != "claude-sonnet-4-6" {
		t.Errorf("model: got %q", cfg.Think.Model)
	}
}

func TestLoad_invalidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(cfgPath, []byte("not valid toml ][[["), 0600)
	os.Setenv("KURT_CONFIG", cfgPath)
	defer os.Unsetenv("KURT_CONFIG")

	_, _, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestSetKey_thinkerProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := SetKey(path, "think.provider", "anthropic"); err != nil {
		t.Fatal(err)
	}

	os.Setenv("KURT_CONFIG", path)
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Think.Provider != "anthropic" {
		t.Errorf("got %q, want anthropic", cfg.Think.Provider)
	}
}

func TestSetKey_style(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = SetKey(path, "style", "powerline")

	os.Setenv("KURT_CONFIG", path)
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, _ := Load()
	if cfg.Style != "powerline" {
		t.Errorf("got %q, want powerline", cfg.Style)
	}
}

func TestSetKey_overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = SetKey(path, "think.provider", "openai")
	_ = SetKey(path, "think.provider", "groq")

	os.Setenv("KURT_CONFIG", path)
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, _ := Load()
	if cfg.Think.Provider != "groq" {
		t.Errorf("overwrite: got %q, want groq", cfg.Think.Provider)
	}
}

func TestSetKey_unknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := SetKey(path, "nonexistent.key", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestSetKey_multipleKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = SetKey(path, "think.provider", "anthropic")
	_ = SetKey(path, "think.model", "claude-sonnet-4-6")
	_ = SetKey(path, "style", "minimal")

	os.Setenv("KURT_CONFIG", path)
	defer os.Unsetenv("KURT_CONFIG")

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Think.Provider != "anthropic" {
		t.Errorf("provider: %q", cfg.Think.Provider)
	}
	if cfg.Think.Model != "claude-sonnet-4-6" {
		t.Errorf("model: %q", cfg.Think.Model)
	}
	if cfg.Style != "minimal" {
		t.Errorf("style: %q", cfg.Style)
	}
}
