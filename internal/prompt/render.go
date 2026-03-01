package prompt

import (
	"fmt"
	"path/filepath"
	"strings"

	"kurt_v1/internal/prompt/modules"
)

type RenderArgs struct {
	Shell      string
	CWD        string
	StatusCode int
	DurationMs int64
	NoColor    bool
	Config     ConfigView
}

type RenderInfo struct {
	Modules []string
}

func Render(a RenderArgs) (string, RenderInfo, error) {
	// v1: module order is config-driven (with safe defaults).
	mods := modulesFromConfig(a.Config)

	ctx := modules.Context{
		Shell:         a.Shell,
		CWD:           filepath.Clean(a.CWD),
		StatusCode:    a.StatusCode,
		DurationMs:    a.DurationMs,
		DurationMinMs: a.Config.DurationMinMs,
		NoColor:       a.NoColor,
	}

	parts := make([]string, 0, len(mods))
	info := RenderInfo{Modules: []string{}}
	for _, m := range mods {
		seg, ok := m.Render(ctx)
		if ok && strings.TrimSpace(seg) != "" {
			parts = append(parts, seg)
			info.Modules = append(info.Modules, m.Name())
		}
	}

	line1 := strings.Join(parts, " ")
	// Input line: keep it simple for now
	line2 := "❯ "
	if a.StatusCode != 0 {
		line2 = "✗ "
	}

	if a.Config.TwoLine {
		out := fmt.Sprintf("%s\n%s", line1, line2)
		return out, info, nil
	}
	// One-line mode
	out := fmt.Sprintf("%s %s", line1, line2)
	return out, info, nil
}
