package modules

type Context struct {
	Shell         string
	CWD           string
	StatusCode    int
	DurationMs    int64
	DurationMinMs int64
	GitTTLms      int64
	NoColor       bool
}

type Module interface {
	Name() string
	Render(ctx Context) (segment string, ok bool)
}
