package modules

import "time"

// TimeModule shows the current time in the prompt.
// Format: Go time format string; falls back to ctx.TimeFormat then "15:04".
type TimeModule struct {
	Format string
}

func (TimeModule) Name() string { return "time" }

func (m TimeModule) Render(ctx Context) (string, bool) {
	format := m.Format
	if format == "" {
		format = ctx.TimeFormat
	}
	if format == "" {
		format = "15:04"
	}
	return time.Now().Format(format), true
}
