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
	// Humanize: ms, s, m+s, h+m
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms), true
	}
	secTotal := ms / 1000
	if secTotal < 60 {
		// keep 1 decimal up to 60s
		sec := float64(ms) / 1000.0
		return fmt.Sprintf("%.1fs", sec), true
	}
	mins := secTotal / 60
	secs := secTotal % 60
	if mins < 60 {
		return fmt.Sprintf("%dm%02ds", mins, secs), true
	}
	hrs := mins / 60
	mins2 := mins % 60
	return fmt.Sprintf("%dh%02dm", hrs, mins2), true
}
