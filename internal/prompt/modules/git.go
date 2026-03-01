package modules

import (
	"time"

	"kurt_v1/internal/gitinfo"
)

type GitModule struct{}

func (m GitModule) Name() string { return "git" }

func (m GitModule) Render(ctx Context) (string, bool) {
	// Prompt-friendly cache TTL
	ttl := time.Second
	if ctx.GitTTLms > 0 {
		ttl = time.Duration(ctx.GitTTLms) * time.Millisecond
	}
	info, ok := gitinfo.Get(ctx.CWD, ttl)
	if !ok {
		return "", false
	}
	return info.String(), true
}
