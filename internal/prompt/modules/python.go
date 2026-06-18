package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonModule shows the active Python version from pyenv, pyvenv.cfg, or python3.
type PythonModule struct{}

func (PythonModule) Name() string { return "python" }

func (PythonModule) Render(ctx Context) (string, bool) {
	// 1. Check for pyenv local version file (.python-version) walking up from cwd
	if ver := pyenvVersionFromDir(ctx.CWD); ver != "" {
		return "py:" + ver, true
	}

	// 2. Check PYENV_VERSION env var
	if v := strings.TrimSpace(os.Getenv("PYENV_VERSION")); v != "" {
		return "py:" + v, true
	}

	// 3. pyvenv.cfg in cwd (created by venv)
	if ver := pyvenvCfgVersionFromDir(ctx.CWD); ver != "" {
		return "py:" + ver, true
	}

	return "", false
}

func pyenvVersion() string {
	return pyenvVersionFromDir("")
}

func pyenvVersionFromDir(startDir string) string {
	cwd := startDir
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	dir := cwd
	for {
		b, err := os.ReadFile(filepath.Join(dir, ".python-version"))
		if err == nil {
			ver := strings.TrimSpace(string(b))
			if ver != "" {
				return ver
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func pyvenvCfgVersion() string { return pyvenvCfgVersionFromDir("") }

func pyvenvCfgVersionFromDir(dir string) string {
	path := "pyvenv.cfg"
	if dir != "" {
		path = filepath.Join(dir, "pyvenv.cfg")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "version") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// pythonBinVersion runs python3 --version as a last resort (slow, use sparingly).
func pythonBinVersion() string {
	out, err := exec.Command("python3", "--version").Output()
	if err != nil {
		return ""
	}
	// "Python 3.11.4" → "3.11.4"
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "Python ")
	return s
}
