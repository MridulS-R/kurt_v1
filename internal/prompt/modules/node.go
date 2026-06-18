package modules

import (
	"os"
	"path/filepath"
	"strings"
)

// NodeModule shows the Node.js version from .nvmrc or .node-version in the cwd.
type NodeModule struct{}

func (NodeModule) Name() string { return "node" }

func (NodeModule) Render(ctx Context) (string, bool) {
	cwd := ctx.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", false
		}
	}

	// Walk up looking for .nvmrc or .node-version
	dir := cwd
	for {
		for _, name := range []string{".nvmrc", ".node-version"} {
			path := filepath.Join(dir, name)
			b, err := os.ReadFile(path)
			if err == nil {
				ver := strings.TrimSpace(string(b))
				if ver != "" {
					if !strings.HasPrefix(ver, "v") {
						ver = "v" + ver
					}
					return "node:" + ver, true
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}
