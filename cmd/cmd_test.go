package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runCmd builds a minimal root cobra command, attaches the named sub-commands,
// and executes it with the given args.  It captures os.Stdout by temporarily
// replacing the file descriptor, since the commands write directly to os.Stdout
// (not the cobra writer).
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Redirect os.Stdout to a pipe so we capture direct fmt.Println output.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	root := &cobra.Command{Use: "kurt", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(promptCmd())
	root.AddCommand(initCmd())
	root.AddCommand(explainCmd())

	// cobra errors go to stderr — silence them so test output is clean.
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)

	execErr := root.Execute()

	// Restore stdout and read captured output.
	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()

	return buf.String(), execErr
}

// ---------------------------------------------------------------------------
// getenvDefault
// ---------------------------------------------------------------------------

func TestGetenvDefault_ReturnsDefault(t *testing.T) {
	key := "KURT_TEST_VAR_XYZ_NONEXISTENT"
	os.Unsetenv(key)
	got := getenvDefault(key, "mydefault")
	if got != "mydefault" {
		t.Errorf("expected %q, got %q", "mydefault", got)
	}
}

func TestGetenvDefault_ReturnsEnv(t *testing.T) {
	key := "KURT_TEST_VAR_XYZ_SET"
	t.Setenv(key, "fromenv")
	got := getenvDefault(key, "mydefault")
	if got != "fromenv" {
		t.Errorf("expected %q, got %q", "fromenv", got)
	}
}

func TestGetenvDefault_WhitespaceOnlyIsDefault(t *testing.T) {
	key := "KURT_TEST_VAR_XYZ_WS"
	t.Setenv(key, "   ")
	got := getenvDefault(key, "fallback")
	if got != "fallback" {
		t.Errorf("whitespace-only env should return default, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// derefInt
// ---------------------------------------------------------------------------

func TestDerefInt_NilPointerReturnsDefault(t *testing.T) {
	got := derefInt(nil, 42)
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestDerefInt_NonNilPointerReturnsValue(t *testing.T) {
	v := 7
	got := derefInt(&v, 42)
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestDerefInt_ZeroPointerReturnsZero(t *testing.T) {
	v := 0
	got := derefInt(&v, 42)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// loadConfigView – uses KURT_CONFIG pointing at a temp TOML file
// ---------------------------------------------------------------------------

func TestLoadConfigView_MissingFileReturnsDefaults(t *testing.T) {
	// Point KURT_CONFIG at a path that doesn't exist — should return defaults.
	t.Setenv("KURT_CONFIG", filepath.Join(t.TempDir(), "nonexistent.toml"))

	cv, path, err := loadConfigView()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty config path")
	}
	// Defaults: style = "minimal"
	if cv.Style != "minimal" {
		t.Errorf("expected style=%q, got %q", "minimal", cv.Style)
	}
}

func TestLoadConfigView_MinimalTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `style = "powerline"

[prompt]
two_line = false
`
	if err := os.WriteFile(cfgPath, []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KURT_CONFIG", cfgPath)

	cv, _, err := loadConfigView()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cv.Style != "powerline" {
		t.Errorf("expected style=%q, got %q", "powerline", cv.Style)
	}
	if cv.TwoLine != false {
		t.Errorf("expected TwoLine=false")
	}
}

func TestLoadConfigView_ModuleEnabledFlags(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
[module.dir]
enabled = true

[module.git]
enabled = true

[module.exit]
enabled = true

[module.duration]
enabled = true
min_ms = 500
`
	if err := os.WriteFile(cfgPath, []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KURT_CONFIG", cfgPath)

	cv, _, err := loadConfigView()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cv.EnableDir {
		t.Error("expected EnableDir=true")
	}
	if !cv.EnableGit {
		t.Error("expected EnableGit=true")
	}
	if !cv.EnableExit {
		t.Error("expected EnableExit=true")
	}
	if !cv.EnableDuration {
		t.Error("expected EnableDuration=true")
	}
}

// ---------------------------------------------------------------------------
// promptCmd – cobra command execution (no network)
// ---------------------------------------------------------------------------

func TestPromptCmd_RendersWithoutError(t *testing.T) {
	t.Setenv("KURT_CONFIG", filepath.Join(t.TempDir(), "nonexistent.toml"))

	out, err := runCmd(t, "prompt", "--cwd", "/tmp", "--status", "0")
	if err != nil {
		t.Fatalf("prompt cmd failed: %v", err)
	}
	// The prompt must contain the input indicator.
	if !strings.Contains(out, "❯") && !strings.Contains(out, "✗") {
		t.Errorf("expected prompt indicator in output, got: %q", out)
	}
}

func TestPromptCmd_NonZeroStatusShowsError(t *testing.T) {
	t.Setenv("KURT_CONFIG", filepath.Join(t.TempDir(), "nonexistent.toml"))

	out, err := runCmd(t, "prompt", "--cwd", "/tmp", "--status", "1")
	if err != nil {
		t.Fatalf("prompt cmd failed: %v", err)
	}
	if !strings.Contains(out, "✗") {
		t.Errorf("expected error indicator in output for non-zero status, got: %q", out)
	}
}

func TestPromptCmd_MissingCwdFails(t *testing.T) {
	_, err := runCmd(t, "prompt", "--status", "0")
	if err == nil {
		t.Error("expected error when --cwd is missing")
	}
}

// ---------------------------------------------------------------------------
// initCmd (zsh snippet output)
// ---------------------------------------------------------------------------

func TestInitZshCmd_ContainsKurtPrompt(t *testing.T) {
	out, err := runCmd(t, "init", "zsh")
	if err != nil {
		t.Fatalf("init zsh failed: %v", err)
	}
	if !strings.Contains(out, "kurt prompt") {
		t.Errorf("expected 'kurt prompt' in zsh snippet, got: %q", out)
	}
}

func TestInitZshCmd_ContainsPrecmd(t *testing.T) {
	out, err := runCmd(t, "init", "zsh")
	if err != nil {
		t.Fatalf("init zsh failed: %v", err)
	}
	if !strings.Contains(out, "__kurt_precmd") {
		t.Errorf("expected '__kurt_precmd' hook in zsh snippet, got: %q", out)
	}
}

// ---------------------------------------------------------------------------
// explainCmd – should not panic or error with a valid (default) config
// ---------------------------------------------------------------------------

func TestExplainCmd_NoError(t *testing.T) {
	t.Setenv("KURT_CONFIG", filepath.Join(t.TempDir(), "nonexistent.toml"))

	out, err := runCmd(t, "explain")
	if err != nil {
		t.Fatalf("explain cmd failed: %v", err)
	}
	if !strings.Contains(out, "style:") {
		t.Errorf("expected 'style:' in explain output, got: %q", out)
	}
}
