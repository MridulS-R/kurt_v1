package modules

import (
	"os"
	"strings"
)

type DirModule struct{}

func (m DirModule) Name() string { return "dir" }

func (m DirModule) Render(ctx Context) (string, bool) {
	cwd := ctx.CWD
	if cwd == "" {
		return "", false
	}
	// ~ expansion
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if cwd == home {
			cwd = "~"
		} else if strings.HasPrefix(cwd, home+string(os.PathSeparator)) {
			cwd = "~" + strings.TrimPrefix(cwd, home)
		}
	}
	return cwd, true
}
