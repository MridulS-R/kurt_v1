package modules

import (
	"time"

	"kurt_v1/internal/gpu"
)

type GpuModule struct{}

func (m GpuModule) Name() string { return "gpu" }

func (m GpuModule) Render(ctx Context) (string, bool) {
	ttl := time.Duration(ctx.GpuTTLms) * time.Millisecond
	if ttl <= 0 {
		ttl = 2 * time.Second
	}
	s := gpu.Get(ttl)
	if s == nil {
		return "", false
	}
	label := s.Label()
	if label == "" {
		return "", false
	}
	return label, true
}
