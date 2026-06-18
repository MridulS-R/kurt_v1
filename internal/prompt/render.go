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
		Shell:           a.Shell,
		CWD:             filepath.Clean(a.CWD),
		StatusCode:      a.StatusCode,
		DurationMs:      a.DurationMs,
		DurationMinMs:   a.Config.DurationMinMs,
		GitTTLms:        a.Config.GitTTLms,
		GpuTTLms:        a.Config.GpuTTLms,
		DirMaxDepth:     a.Config.DirMaxDepth,
		DirTruncateMid:  a.Config.DirTruncateMid,
		GitBranchMaxLen: a.Config.GitBranchMaxLen,
		GitBranchTail:   a.Config.GitBranchTail,
		ExitCompact:     a.Config.ExitCompact,
		NoColor:         a.NoColor,
		TimeFormat:      a.Config.TimeFormat,
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

	style := strings.ToLower(strings.TrimSpace(a.Config.Style))
	line1 := ""
	if style == "powerline" && !a.NoColor {
		// Powerline uses its own segment rendering (requires Nerd Font).
		segs := make([]plSeg, 0, len(parts))
		for i, name := range info.Modules {
			cp := a.Config.Powerline.For(name)
			segs = append(segs, plSeg{Text: parts[i], Fg: cp.Fg, Bg: cp.Bg})
		}
		line1 = renderPowerline(segs)
	} else {
		// Minimal style: foreground colors only (or no color when --no-color).
		colored := make([]string, 0, len(parts))
		for i, name := range info.Modules {
			seg := parts[i]
			if !a.NoColor {
				fg := 0
				switch name {
				case "dir":
					fg = a.Config.FgDir
				case "git":
					fg = a.Config.FgGit
				case "duration":
					fg = a.Config.FgDuration
				case "exit":
					fg = a.Config.FgExit
				case "gpu":
					fg = a.Config.FgGpu
				case "venv":
					fg = a.Config.FgVenv
				case "conda":
					fg = a.Config.FgConda
				case "node":
					fg = a.Config.FgNode
				case "kube":
					fg = a.Config.FgKube
				case "battery":
					fg = a.Config.FgBattery
				case "python":
					fg = a.Config.FgPython
				case "cloud":
					fg = a.Config.FgCloud
				case "time":
					fg = a.Config.FgTime
				}
				if fg > 0 {
					seg = fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", fg, seg)
				}
			}
			colored = append(colored, seg)
		}
		line1 = strings.Join(colored, " ")
	}
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
