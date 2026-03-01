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
}

type RenderInfo struct {
	Modules []string
}

func Render(a RenderArgs) (string, RenderInfo, error) {
	// v1: hardcoded order; config comes later.
	mods := []modules.Module{
		modules.DirModule{},
		modules.GitModule{},
		modules.DurationModule{},
		modules.ExitModule{},
	}

	ctx := modules.Context{
		Shell:      a.Shell,
		CWD:        filepath.Clean(a.CWD),
		StatusCode: a.StatusCode,
		DurationMs: a.DurationMs,
		NoColor:    a.NoColor,
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

	out := fmt.Sprintf("%s\n%s", line1, line2)
	return out, info, nil
}
