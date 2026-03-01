package prompt

import (
	"fmt"
	"strings"
)

// Very small Powerline renderer (v1).
// Uses 256-color ANSI so it works in macOS Terminal.
//
// Note: requires a Nerd Font / Powerline glyph support for the separator.

const plSep = "" // Powerline separator (solid)

type plSeg struct {
	Text string
	Fg   int // ANSI 256-color
	Bg   int
}

func ansiFg(n int) string { return fmt.Sprintf("\x1b[38;5;%dm", n) }
func ansiBg(n int) string { return fmt.Sprintf("\x1b[48;5;%dm", n) }
func ansiReset() string   { return "\x1b[0m" }

func renderPowerline(segments []plSeg) string {
	if len(segments) == 0 {
		return ""
	}
	var b strings.Builder

	for i, s := range segments {
		// segment body
		b.WriteString(ansiBg(s.Bg))
		b.WriteString(ansiFg(s.Fg))
		b.WriteString(" ")
		b.WriteString(strings.TrimSpace(s.Text))
		b.WriteString(" ")

		// separator into next background
		if i < len(segments)-1 {
			next := segments[i+1]
			b.WriteString(ansiFg(s.Bg))
			b.WriteString(ansiBg(next.Bg))
			b.WriteString(plSep)
		}
	}
	b.WriteString(ansiReset())
	return b.String()
}
