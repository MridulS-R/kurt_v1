package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertTOML_topLevel(t *testing.T) {
	lines := upsertTOML(nil, "", "style", `"powerline"`)
	if len(lines) != 1 || lines[0] != `style = "powerline"` {
		t.Errorf("got %v", lines)
	}
}

func TestUpsertTOML_topLevel_overwrite(t *testing.T) {
	lines := []string{`style = "minimal"`}
	lines = upsertTOML(lines, "", "style", `"powerline"`)
	if len(lines) != 1 || lines[0] != `style = "powerline"` {
		t.Errorf("got %v", lines)
	}
}

func TestUpsertTOML_newSection(t *testing.T) {
	lines := upsertTOML(nil, "think", "provider", `"anthropic"`)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "[think]") {
		t.Errorf("missing [think] section: %q", joined)
	}
	if !strings.Contains(joined, `provider = "anthropic"`) {
		t.Errorf("missing provider: %q", joined)
	}
}

func TestUpsertTOML_existingSection(t *testing.T) {
	lines := []string{"[think]", `provider = "openai"`}
	lines = upsertTOML(lines, "think", "provider", `"groq"`)
	joined := strings.Join(lines, "\n")
	if strings.Count(joined, "[think]") != 1 {
		t.Errorf("should not duplicate section header: %q", joined)
	}
	if !strings.Contains(joined, `provider = "groq"`) {
		t.Errorf("should have updated value: %q", joined)
	}
}

func TestUpsertTOML_addKeyToExistingSection(t *testing.T) {
	lines := []string{"[think]", `provider = "openai"`, "", "[perf]", "git_ttl_ms = 1000"}
	lines = upsertTOML(lines, "think", "model", `"gpt-4o"`)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, `model = "gpt-4o"`) {
		t.Errorf("should have added model: %q", joined)
	}
}

func TestParseKeyValue_supported(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		section string
		field   string
	}{
		{"style", "powerline", "", "style"},
		{"think.provider", "anthropic", "think", "provider"},
		{"think.model", "claude-sonnet-4-6", "think", "model"},
		{"perf.git_ttl_ms", "2000", "perf", "git_ttl_ms"},
		{"module.gpu.enabled", "true", "module.gpu", "enabled"},
	}
	for _, tt := range tests {
		section, field, _, err := parseKeyValue(tt.key, tt.value)
		if err != nil {
			t.Errorf("key %q: unexpected error: %v", tt.key, err)
			continue
		}
		if section != tt.section {
			t.Errorf("key %q: section got %q, want %q", tt.key, section, tt.section)
		}
		if field != tt.field {
			t.Errorf("key %q: field got %q, want %q", tt.key, field, tt.field)
		}
	}
}

func TestParseKeyValue_unsupported(t *testing.T) {
	_, _, _, err := parseKeyValue("totally.fake.key", "val")
	if err == nil {
		t.Error("expected error for unsupported key")
	}
}

func TestSetKey_createsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.toml")
	if err := SetKey(path, "style", "minimal"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("file should have been created")
	}
}
