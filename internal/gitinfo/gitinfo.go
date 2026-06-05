package gitinfo

import (
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
	cache = map[string]cacheEntry{}
)

// Get returns cached git info for cwd (if inside a repo).
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

	info := fetchAll(root)

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

// fetchAll gets branch, dirty flag, and ahead/behind in a single git call.
// Uses porcelain=v2 which was introduced in git 2.11 (2016).
func fetchAll(repoRoot string) Info {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v2", "--branch", "-uno")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return Info{RepoRoot: repoRoot}
	}
	return parseStatusV2(repoRoot, string(out))
}

func parseStatusV2(repoRoot, out string) Info {
	info := Info{RepoRoot: repoRoot}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			br := strings.TrimSpace(strings.TrimPrefix(line, "# branch.head "))
			if br != "(detached)" {
				info.Branch = br
			}
		case strings.HasPrefix(line, "# branch.ab "):
			// format: "+<ahead> -<behind>"
			parts := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(parts) == 2 {
				info.Ahead = atoi(strings.TrimPrefix(parts[0], "+"))
				info.Behind = atoi(strings.TrimPrefix(parts[1], "-"))
			}
		case line != "" && !strings.HasPrefix(line, "#"):
			info.Dirty = true
		}
	}
	return info
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func (i Info) String(branchMaxLen int, branchTail bool) string {
	if i.Branch == "" {
		return ""
	}
	br := shortenBranch(i.Branch, branchMaxLen, branchTail)
	b := " " + br
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

func shortenBranch(br string, maxLen int, tail bool) string {
	br = strings.TrimSpace(br)
	if br == "" {
		return br
	}
	if tail {
		if idx := strings.LastIndex(br, "/"); idx >= 0 && idx < len(br)-1 {
			br = br[idx+1:]
		}
	}
	if maxLen <= 0 {
		maxLen = 28
	}
	if len(br) <= maxLen {
		return br
	}
	if maxLen < 2 {
		return br[:maxLen]
	}
	return br[:maxLen-1] + "…"
}
