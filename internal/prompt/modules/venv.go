package modules

import (
	"os"
	"path/filepath"
)

// VenvModule shows the active Python virtual environment name.
type VenvModule struct{}

func (VenvModule) Name() string { return "venv" }

func (VenvModule) Render(ctx Context) (string, bool) {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return "", false
	}
	name := filepath.Base(venv)
	if name == ".venv" || name == "venv" {
		// Use parent dir name for generic names
		parent := filepath.Base(filepath.Dir(venv))
		if parent != "" && parent != "." {
			name = parent
		}
	}
	return "(" + name + ")", true
}
