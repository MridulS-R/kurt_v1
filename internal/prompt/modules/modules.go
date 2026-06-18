package modules

type Context struct {
	Shell           string
	CWD             string
	StatusCode      int
	DurationMs      int64
	DurationMinMs   int64
	GitTTLms        int64
	GpuTTLms        int64
	DirMaxDepth     int
	DirTruncateMid  bool
	GitBranchMaxLen int
	GitBranchTail   bool
	ExitCompact     bool
	NoColor         bool
	TimeFormat      string
}

type Module interface {
	Name() string
	Render(ctx Context) (segment string, ok bool)
}
