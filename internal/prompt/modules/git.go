package modules

import (
	"bytes"
	"os/exec"
	"strings"
)

type GitModule struct{}

func (m GitModule) Name() string { return "git" }

func (m GitModule) Render(ctx Context) (string, bool) {
	// Quick check: are we inside a git repo?
	branch := gitBranch(ctx.CWD)
	if branch == "" {
		return "", false
	}
	dirty := gitDirty(ctx.CWD)
	if dirty {
		return " " + branch + "*", true
	}
	return " " + branch, true
}

func gitBranch(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(string(out))
	if b == "HEAD" || b == "" {
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
