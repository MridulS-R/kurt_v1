package modules

import (
	"os"
	"strings"
)

type DirModule struct{}

func (m DirModule) Name() string { return "dir" }

func (m DirModule) Render(ctx Context) (string, bool) {
	cwd := ctx.CWD
	if cwd == "" {
		return "", false
	}
	// ~ expansion
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if cwd == home {
			cwd = "~"
		} else if strings.HasPrefix(cwd, home+string(os.PathSeparator)) {
			cwd = "~" + strings.TrimPrefix(cwd, home)
		}
	}

	maxDepth := ctx.DirMaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}
	cwd = shortenPath(cwd, maxDepth, ctx.DirTruncateMid)

	return cwd, true
}

func shortenPath(p string, maxDepth int, truncateMid bool) string {
	// Keep ~ prefix intact
	prefix := ""
	rest := p
	if strings.HasPrefix(p, "~") {
		prefix = "~"
		rest = strings.TrimPrefix(p, "~")
	}
	rest = strings.TrimPrefix(rest, string(os.PathSeparator))
	parts := []string{}
	if rest != "" {
		parts = strings.Split(rest, string(os.PathSeparator))
	}
	if len(parts) <= maxDepth {
		if prefix != "" {
			if len(parts) == 0 {
				return "~"
			}
			return prefix + string(os.PathSeparator) + strings.Join(parts, string(os.PathSeparator))
		}
		return p
	}

	// show last maxDepth parts
	tail := parts[len(parts)-maxDepth:]
	out := strings.Join(tail, string(os.PathSeparator))
	if truncateMid {
		out = "…" + string(os.PathSeparator) + out
	}
	if prefix != "" {
		return prefix + string(os.PathSeparator) + out
	}
	return out
}
