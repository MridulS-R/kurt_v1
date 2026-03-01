package think

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
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

type Context struct {
	CWD  string       `json:"cwd"`
	Last *LastCommand `json:"last,omitempty"`
	Git  *GitInfo     `json:"git,omitempty"`
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
		s = strings.TrimPrefix(s, "-")
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
		s = strings.TrimPrefix(s, "-")
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
