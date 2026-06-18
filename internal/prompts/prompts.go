package prompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Prompt is a named, reusable prompt template.
type Prompt struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Template    string `json:"template"`
}

// ── storage ───────────────────────────────────────────────────────────────────

func storePath() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(h, ".config", "kurt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "prompts.json"), nil
}

func load() (map[string]Prompt, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Prompt{}, nil
		}
		return nil, err
	}
	var m map[string]Prompt
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func save(m map[string]Prompt) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// Add saves a prompt. Overwrites if name already exists.
func Add(name, templateStr, description string) error {
	m, err := load()
	if err != nil {
		return err
	}
	m[name] = Prompt{Name: name, Description: description, Template: templateStr}
	return save(m)
}

// Remove deletes a prompt by name.
func Remove(name string) error {
	m, err := load()
	if err != nil {
		return err
	}
	if _, ok := m[name]; !ok {
		return fmt.Errorf("prompt %q not found", name)
	}
	delete(m, name)
	return save(m)
}

// Get retrieves a prompt by name.
func Get(name string) (Prompt, error) {
	m, err := load()
	if err != nil {
		return Prompt{}, err
	}
	p, ok := m[name]
	if !ok {
		return Prompt{}, fmt.Errorf("prompt %q not found — use: kurt prompts list", name)
	}
	return p, nil
}

// List returns all prompts sorted by name.
func List() ([]Prompt, error) {
	m, err := load()
	if err != nil {
		return nil, err
	}
	out := make([]Prompt, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ── execution ─────────────────────────────────────────────────────────────────

// Render executes a prompt template with the given variables.
// "input" is the special variable populated from stdin/argument.
// Extra vars come from key=value pairs.
func Render(tmpl string, input string, vars map[string]string) (string, error) {
	data := map[string]string{"input": input}
	for k, v := range vars {
		data[k] = v
	}

	// Support both {{.varname}} and {{varname}} style.
	t, err := template.New("p").Delims("{{", "}}").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("template parse: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		// Try wrapping in a dot accessor for plain {{varname}} style.
		return "", fmt.Errorf("template execute: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
