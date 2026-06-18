package modules

import "os"

// CondaModule shows the active Conda environment name.
type CondaModule struct{}

func (CondaModule) Name() string { return "conda" }

func (CondaModule) Render(ctx Context) (string, bool) {
	env := os.Getenv("CONDA_DEFAULT_ENV")
	if env == "" || env == "base" {
		return "", false
	}
	return "[" + env + "]", true
}
