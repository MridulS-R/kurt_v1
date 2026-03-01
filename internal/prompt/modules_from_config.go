package prompt

import (
	"strings"

	"kurt_v1/internal/prompt/modules"
)

func modulesFromConfig(cfg ConfigView) []modules.Module {
	// Map names to module implementations.
	m := map[string]modules.Module{
		"dir":      modules.DirModule{},
		"git":      modules.GitModule{},
		"duration": modules.DurationModule{},
		"exit":     modules.ExitModule{},
	}

	order := cfg.Order
	if len(order) == 0 {
		order = []string{"dir", "git", "duration", "exit"}
	}

	out := make([]modules.Module, 0, len(order))
	seen := map[string]bool{}
	for _, raw := range order {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if !cfg.Enabled(name) {
			continue
		}
		if mod, ok := m[name]; ok {
			out = append(out, mod)
		}
	}
	return out
}
