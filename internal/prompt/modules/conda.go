package modules

import "os"

// CondaModule shows the active Conda environment name.
// ShowBase: if true, also show the "base" environment (default: suppress it).
type CondaModule struct {
	ShowBase bool
}

func (CondaModule) Name() string { return "conda" }

func (m CondaModule) Render(ctx Context) (string, bool) {
	env := os.Getenv("CONDA_DEFAULT_ENV")
	if env == "" {
		return "", false
	}
	if env == "base" && !m.ShowBase {
		return "", false
	}
	return "conda:" + env, true
}
