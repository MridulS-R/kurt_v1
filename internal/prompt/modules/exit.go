package modules

import "fmt"

type ExitModule struct{}

func (m ExitModule) Name() string { return "exit" }

func (m ExitModule) Render(ctx Context) (string, bool) {
	if ctx.StatusCode == 0 {
		return "", false
	}
	if ctx.ExitCompact {
		return fmt.Sprintf("✗%d", ctx.StatusCode), true
	}
	return fmt.Sprintf("exit=%d", ctx.StatusCode), true
}
