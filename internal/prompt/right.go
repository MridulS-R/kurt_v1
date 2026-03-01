package prompt

import (
	"fmt"
	"strings"
	"time"

	"kurt_v1/internal/prompt/modules"
)

type RenderRightArgs struct {
	Shell      string
	CWD        string
	StatusCode int
	DurationMs int64
	NoColor    bool
	Config     ConfigView
}

type RenderRightInfo struct {
	Modules []string
}

// RenderRight renders a compact right prompt.
// v1: time + duration (if shown by threshold).
func RenderRight(a RenderRightArgs) (string, RenderRightInfo, error) {
	info := RenderRightInfo{Modules: []string{}}

	if !a.Config.RPromptEnabled {
		return "", info, nil
	}

	parts := []string{}

	// duration (reuse module logic)
	if a.Config.EnableDuration {
		ctx := modules.Context{DurationMs: a.DurationMs, DurationMinMs: a.Config.DurationMinMs}
		seg, ok := modules.DurationModule{}.Render(ctx)
		if ok && strings.TrimSpace(seg) != "" {
			parts = append(parts, seg)
			info.Modules = append(info.Modules, "duration")
		}
	}

	// time (always)
	if a.Config.RPromptShowTime {
		parts = append(parts, time.Now().Format(a.Config.RPromptTimeFormat))
		info.Modules = append(info.Modules, "time")
	}

	out := strings.Join(parts, " ")
	return fmt.Sprintf("%s", out), info, nil
}
