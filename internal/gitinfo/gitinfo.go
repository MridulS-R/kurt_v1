package gitinfo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Info struct {
	RepoRoot string
	Branch   string
	Dirty    bool
	Ahead    int
	Behind   int
}

type cacheEntry struct {
	info      Info
	fetchedAt time.Time
}

var (
	mu    sync.Mutex
	cache = map[string]cacheEntry{} // key: repoRoot
)

// Get returns cached git info for cwd (if inside a repo). It uses a short TTL.
// This is designed for prompts: fast and safe.
func Get(cwd string, ttl time.Duration) (Info, bool) {
	root := repoRoot(cwd)
	if root == "" {
		return Info{}, false
	}

	mu.Lock()
	ce, ok := cache[root]
	if ok && time.Since(ce.fetchedAt) <= ttl {
		mu.Unlock()
		return ce.info, true
	}
	mu.Unlock()

	info := Info{RepoRoot: root}
	// Use small time budgets per command.
	info.Branch = branch(root)
	info.Dirty = dirty(root)
	info.Ahead, info.Behind = aheadBehind(root)

	mu.Lock()
	cache[root] = cacheEntry{info: info, fetchedAt: time.Now()}
	mu.Unlock()

	if info.Branch == "" {
		return Info{}, false
	}
	return info, true
}

func repoRoot(cwd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

func branch(repoRoot string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoRoot
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

func dirty(repoRoot string) bool {
	// Fast-ish porcelain; avoid untracked for speed (-uno).
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "-uno")
	cmd.Dir = repoRoot
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(buf.String()) != ""
}

func aheadBehind(repoRoot string) (int, int) {
	// Best-effort. If no upstream or slow, return 0,0.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0
	}
	// Output is: behind ahead (left=upstream, right=HEAD)
	behind := atoi(parts[0])
	ahead := atoi(parts[1])
	return ahead, behind
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func (i Info) String() string {
	if i.Branch == "" {
		return ""
	}
	b := " " + i.Branch
	if i.Dirty {
		b += "*"
	}
	if i.Ahead > 0 {
		b += fmt.Sprintf(" ↑%d", i.Ahead)
	}
	if i.Behind > 0 {
		b += fmt.Sprintf(" ↓%d", i.Behind)
	}
	return b
}
