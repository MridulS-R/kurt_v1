package think

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kurt_v1/internal/history"
)

type LastCommand struct {
	Cmd        string `json:"cmd"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

type GitInfo struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type FailureSummary struct {
	Cmd      string `json:"cmd"`
	Count    int    `json:"count"`
	ExitCode int    `json:"last_exit_code"`
	AgeHours int    `json:"age_hours"`
}

type Context struct {
	CWD         string            `json:"cwd"`
	Last        *LastCommand      `json:"last,omitempty"`
	Git         *GitInfo          `json:"git,omitempty"`
	ProjectType string            `json:"project_type,omitempty"`
	GitLog      []string          `json:"git_log,omitempty"`
	Failures    []FailureSummary  `json:"recent_failures,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

func CollectLastFromEnv() *LastCommand {
	cmd := strings.TrimSpace(os.Getenv("__KURT_LAST_CMD"))
	exitStr := strings.TrimSpace(os.Getenv("__KURT_LAST_EXIT"))
	durStr := strings.TrimSpace(os.Getenv("__KURT_LAST_DURATION_MS"))

	exit := atoiDefault(exitStr, 0)
	dur := atoi64Default(durStr, 0)

	if cmd == "" && exit == 0 && dur == 0 {
		return nil
	}
	return &LastCommand{Cmd: cmd, ExitCode: exit, DurationMs: dur}
}

func CollectGit(cwd string) *GitInfo {
	branch := gitBranch(cwd)
	if branch == "" {
		return nil
	}
	return &GitInfo{Branch: branch, Dirty: gitDirty(cwd)}
}

// CollectProjectType detects the kind of project in cwd.
func CollectProjectType(cwd string) string {
	markers := []struct {
		file string
		kind string
	}{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"Gemfile", "ruby"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"Cargo.toml", "rust"},
		{"docker-compose.yml", "docker"},
		{"docker-compose.yaml", "docker"},
		{"compose.yml", "docker"},
		{"Dockerfile", "docker"},
	}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(cwd, m.file)); err == nil {
			return m.kind
		}
	}
	return ""
}

// CollectGitLog returns the last n commit messages as one-liners.
func CollectGitLog(cwd string, n int) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "-"+itoa(n))
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// CollectFailures returns a deduplicated summary of recent failures.
func CollectFailures(n int) []FailureSummary {
	entries, err := history.Recent(500)
	if err != nil || len(entries) == 0 {
		return nil
	}
	// Aggregate by command (trim args for grouping).
	type agg struct {
		count    int
		exitCode int
		lastAt   time.Time
	}
	m := map[string]*agg{}
	order := []string{}
	for _, e := range entries {
		key := cmdKey(e.Cmd)
		if a, ok := m[key]; ok {
			a.count++
			if e.At.After(a.lastAt) {
				a.lastAt = e.At
				a.exitCode = e.ExitCode
			}
		} else {
			m[key] = &agg{count: 1, exitCode: e.ExitCode, lastAt: e.At}
			order = append(order, key)
		}
	}
	// Sort by most recent, take top n.
	// Simple: iterate order in reverse (newest appended last).
	var out []FailureSummary
	for i := len(order) - 1; i >= 0 && len(out) < n; i-- {
		k := order[i]
		a := m[k]
		out = append(out, FailureSummary{
			Cmd:      k,
			Count:    a.count,
			ExitCode: a.exitCode,
			AgeHours: int(time.Since(a.lastAt).Hours()),
		})
	}
	return out
}

// CollectEnv returns relevant env vars (non-sensitive).
func CollectEnv() map[string]string {
	keys := []string{
		"AWS_PROFILE", "AWS_DEFAULT_REGION",
		"KUBECONFIG", "KUBECTL_CONTEXT",
		"VIRTUAL_ENV", "CONDA_DEFAULT_ENV",
		"NODE_ENV", "GO_ENV", "RAILS_ENV",
		"DOCKER_HOST", "COMPOSE_PROJECT_NAME",
	}
	out := map[string]string{}
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cmdKey(cmd string) string {
	// Use first two words as the key (e.g. "docker compose" not full args).
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return cmd
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func gitBranch(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(string(out))
	if b == "" || b == "HEAD" {
		return ""
	}
	return b
}

func gitDirty(cwd string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(buf.String()) != ""
}

func atoiDefault(s string, def int) int {
	v := 0
	neg := false
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		v = v*10 + int(ch-'0')
	}
	if neg {
		v = -v
	}
	return v
}

func atoi64Default(s string, def int64) int64 {
	v := int64(0)
	neg := false
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		v = v*10 + int64(ch-'0')
	}
	if neg {
		v = -v
	}
	return v
}
