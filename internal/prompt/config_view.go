package prompt

import "strings"

// ConfigView is a minimal prompt-facing view of the loaded config.
// This avoids importing the full config package into prompt modules.
// (Keeps boundaries clean.)

type ConfigView struct {
	Style   string
	TwoLine bool
	Order   []string

	EnableDir      bool
	EnableGit      bool
	EnableDuration bool
	EnableExit     bool

	DurationMinMs int64
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
	default:
		return false
	}
}
