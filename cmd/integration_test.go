package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Package-level binary path, built once by TestMain.
// ---------------------------------------------------------------------------

var kurtBin string

func TestMain(m *testing.M) {
	// Build the binary into a temp dir shared across all integration tests.
	dir, err := os.MkdirTemp("", "kurt-integration-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)

	bin := filepath.Join(dir, "kurt")
	root := findModuleRoot()
	if root == "" {
		panic("could not find module root (go.mod not found)")
	}

	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + err.Error() + "\n" + string(out))
	}

	kurtBin = bin
	os.Exit(m.Run())
}

// findModuleRoot walks up from the directory containing this test file
// to locate the nearest go.mod file and returns its directory.
func findModuleRoot() string {
	// Start from the cmd package directory.
	dir, err := filepath.Abs(filepath.Dir("."))
	if err != nil {
		return ""
	}
	// Walk upward until we find go.mod or hit the filesystem root.
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// buildKurt compiles the binary once per individual test run into a temp dir.
// Prefer the package-level kurtBin set by TestMain; fall back to building
// per-test when called outside of TestMain (e.g. in sub-benchmarks).
func buildKurt(t *testing.T) string {
	t.Helper()
	if kurtBin != "" {
		return kurtBin
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "kurt")
	root := findModuleRoot()
	if root == "" {
		t.Fatal("could not find module root (go.mod not found)")
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// runBinary executes the kurt binary with the provided args and returns
// (combined stdout+stderr output, exit error). A nil error means exit 0.
func runBinary(t *testing.T, args ...string) (string, error) {
	t.Helper()
	bin := buildKurt(t)
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestBinary_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "--help")
	if err != nil {
		t.Fatalf("kurt --help failed: %v\noutput: %s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected non-empty help output")
	}
}

func TestBinary_Prompt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "prompt", "--cwd", "/tmp", "--status", "0")
	if err != nil {
		t.Fatalf("kurt prompt failed: %v\noutput: %s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected non-empty prompt output")
	}
}

func TestBinary_Explain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "explain")
	if err != nil {
		t.Fatalf("kurt explain failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Style") {
		t.Errorf("expected 'Style' in explain output, got: %s", out)
	}
}

func TestBinary_InitZsh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "init", "zsh")
	if err != nil {
		t.Fatalf("kurt init zsh failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "kurt prompt") {
		t.Errorf("expected 'kurt prompt' in zsh snippet, got: %s", out)
	}
	if strings.Contains(out, "%!") {
		t.Errorf("zsh snippet contains Go format error marker '%%!': %s", out)
	}
}

func TestBinary_InitBash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "init", "bash")
	if err != nil {
		t.Fatalf("kurt init bash failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "kurt prompt") {
		t.Errorf("expected 'kurt prompt' in bash snippet, got: %s", out)
	}
	if !strings.Contains(out, "PROMPT_COMMAND") {
		t.Errorf("expected 'PROMPT_COMMAND' in bash snippet, got: %s", out)
	}
	if strings.Contains(out, "%!") {
		t.Errorf("bash snippet contains Go format error marker '%%!': %s", out)
	}
}

func TestBinary_InitFish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "init", "fish")
	if err != nil {
		t.Fatalf("kurt init fish failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "kurt prompt") {
		t.Errorf("expected 'kurt prompt' in fish snippet, got: %s", out)
	}
	if !strings.Contains(out, "fish_prompt") {
		t.Errorf("expected 'fish_prompt' function in fish snippet, got: %s", out)
	}
	if strings.Contains(out, "%!") {
		t.Errorf("fish snippet contains Go format error marker '%%!': %s", out)
	}
}

func TestBinary_InitHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "init", "--help")
	if err != nil {
		t.Fatalf("kurt init --help failed: %v\noutput: %s", err, out)
	}
	// Should list available shells.
	if !strings.Contains(out, "zsh") {
		t.Errorf("expected 'zsh' in init help output, got: %s", out)
	}
	if !strings.Contains(out, "bash") {
		t.Errorf("expected 'bash' in init help output, got: %s", out)
	}
	if !strings.Contains(out, "fish") {
		t.Errorf("expected 'fish' in init help output, got: %s", out)
	}
}

func TestBinary_Suggest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// kurt suggest with a known prefix that has a heuristic expansion.
	out, err := runBinary(t, "suggest", "--buffer", "git ", "--cwd", "/tmp")
	if err != nil {
		t.Fatalf("kurt suggest failed: %v\noutput: %s", err, out)
	}
	// Output may be empty (no suggestion) or contain a suggestion — either is valid.
	_ = out
}

func TestBinary_PromptNonZeroStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "prompt", "--cwd", "/tmp", "--status", "1")
	if err != nil {
		t.Fatalf("kurt prompt --status 1 failed: %v\noutput: %s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected non-empty prompt output for non-zero status")
	}
}

func TestBinary_PromptMissingCwdFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	_, err := runBinary(t, "prompt", "--status", "0")
	if err == nil {
		t.Error("expected non-zero exit when --cwd is missing")
	}
}

func TestBinary_SuggestHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	out, err := runBinary(t, "suggest", "--help")
	if err != nil {
		t.Fatalf("kurt suggest --help failed: %v\noutput: %s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected non-empty suggest help output")
	}
}
