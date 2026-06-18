package modules

import "time"

// TimeModule shows the current time in the prompt.
type TimeModule struct{}

func (TimeModule) Name() string { return "time" }

func (TimeModule) Render(ctx Context) (string, bool) {
	format := ctx.TimeFormat
	if format == "" {
		format = "15:04"
	}
	return time.Now().Format(format), true
}
