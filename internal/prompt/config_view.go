package prompt

import "strings"

// ConfigView is a minimal prompt-facing view of the loaded config.
// This avoids importing the full config package into prompt modules.
// (Keeps boundaries clean.)

type ConfigView struct {
	Style   string
	TwoLine bool
	Order   []string

	// Performance knobs
	GitTTLms int64
	GpuTTLms int64

	// Minimal style colors (ANSI 256 fg)
	FgDir      int
	FgGit      int
	FgDuration int
	FgExit     int
	FgGpu      int

	// Readability
	DirMaxDepth     int
	DirTruncateMid  bool
	GitBranchMaxLen int
	GitBranchTail   bool
	ExitCompact     bool

	// Right prompt (zsh RPROMPT)
	RPromptEnabled    bool
	RPromptShowTime   bool
	RPromptTimeFormat string

	EnableDir      bool
	EnableGit      bool
	EnableDuration bool
	EnableExit     bool
	EnableGpu      bool
	EnableVenv     bool
	EnableConda    bool
	EnableNode     bool
	EnableKube     bool
	EnableBattery  bool
	EnablePython   bool
	EnableCloud    bool
	EnableTime     bool

	FgVenv    int
	FgConda   int
	FgNode    int
	FgKube    int
	FgBattery int
	FgPython  int
	FgCloud   int
	FgTime    int
	TimeFormat string

	DurationMinMs int64
	Powerline     PowerlinePalette
}

type ColorPair struct {
	Fg int
	Bg int
}

type PowerlinePalette struct {
	Dir      ColorPair
	Git      ColorPair
	Duration ColorPair
	Exit     ColorPair
}

func (p PowerlinePalette) For(name string) ColorPair {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "dir":
		return p.Dir
	case "git":
		return p.Git
	case "duration":
		return p.Duration
	case "exit":
		return p.Exit
	default:
		return ColorPair{Fg: 15, Bg: 31}
	}
}

func (c ConfigView) Enabled(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "dir":
		return c.EnableDir
	case "git":
		return c.EnableGit
	case "duration":
		return c.EnableDuration
	case "exit":
		return c.EnableExit
	case "gpu":
		return c.EnableGpu
	case "venv":
		return c.EnableVenv
	case "conda":
		return c.EnableConda
	case "node":
		return c.EnableNode
	case "kube":
		return c.EnableKube
	case "battery":
		return c.EnableBattery
	case "python":
		return c.EnablePython
	case "cloud":
		return c.EnableCloud
	case "time":
		return c.EnableTime
	default:
		return false
	}
}
