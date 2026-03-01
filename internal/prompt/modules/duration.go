package modules

import "fmt"

type DurationModule struct{}

func (m DurationModule) Name() string { return "duration" }

func (m DurationModule) Render(ctx Context) (string, bool) {
	ms := ctx.DurationMs
	if ms <= 0 {
		return "", false
	}
	min := ctx.DurationMinMs
	if min <= 0 {
		min = 500
	}
	if ms < min {
		return "", false
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms), true
	}
	sec := float64(ms) / 1000.0
	return fmt.Sprintf("%.1fs", sec), true
}
