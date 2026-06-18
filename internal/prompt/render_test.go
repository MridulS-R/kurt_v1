package prompt

import (
	"strings"
	"testing"
)

func defaultCfg() ConfigView {
	return ConfigView{
		Style:           "minimal",
		TwoLine:         true,
		Order:           []string{"dir", "git", "duration", "exit"},
		EnableDir:       true,
		EnableGit:       true,
		EnableDuration:  true,
		EnableExit:      true,
		DirMaxDepth:     3,
		GitBranchMaxLen: 28,
		GitTTLms:        1000,
		FgDir:           33,
		FgGit:           35,
		FgDuration:      221,
		FgExit:          160,
		DurationMinMs:   500,
		ExitCompact:     true,
	}
}

func TestRender_twoLine(t *testing.T) {
	cfg := defaultCfg()
	cfg.TwoLine = true
	out, _, err := Render(RenderArgs{
		Shell:      "zsh",
		CWD:        "/tmp",
		StatusCode: 0,
		DurationMs: 0,
		Config:     cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n") {
		t.Error("two-line mode should contain newline")
	}
	if !strings.Contains(out, "❯") {
		t.Error("prompt should contain ❯")
	}
}

func TestRender_oneLine(t *testing.T) {
	cfg := defaultCfg()
	cfg.TwoLine = false
	out, _, err := Render(RenderArgs{
		Shell:  "bash",
		CWD:    "/home/user",
		Config: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 1 {
		t.Errorf("one-line mode: got %d lines", len(lines))
	}
}

func TestRender_nonZeroExit(t *testing.T) {
	cfg := defaultCfg()
	out, _, err := Render(RenderArgs{
		CWD:        "/tmp",
		StatusCode: 1,
		Config:     cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "✗") {
		t.Error("non-zero exit should show ✗")
	}
}

func TestRender_noColor(t *testing.T) {
	cfg := defaultCfg()
	out, _, err := Render(RenderArgs{
		CWD:     "/tmp",
		NoColor: true,
		Config:  cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("no-color mode should not contain ANSI escape codes")
	}
}

func TestRender_disabledModules(t *testing.T) {
	cfg := defaultCfg()
	cfg.EnableDir = false
	cfg.EnableGit = false
	cfg.EnableDuration = false
	cfg.EnableExit = false
	out, info, err := Render(RenderArgs{CWD: "/tmp", Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Modules) != 0 {
		t.Errorf("all disabled: expected 0 modules, got %v", info.Modules)
	}
	_ = out
}

func TestRender_moduleInfo(t *testing.T) {
	cfg := defaultCfg()
	cfg.EnableGit = false
	cfg.EnableDuration = false
	_, info, err := Render(RenderArgs{CWD: "/tmp", Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range info.Modules {
		if name == "git" {
			t.Error("disabled git should not appear in module info")
		}
	}
}

func TestRender_powerlineStyle(t *testing.T) {
	cfg := defaultCfg()
	cfg.Style = "powerline"
	cfg.Powerline = PowerlinePalette{
		Dir:      ColorPair{Fg: 15, Bg: 31},
		Git:      ColorPair{Fg: 15, Bg: 28},
		Duration: ColorPair{Fg: 16, Bg: 220},
		Exit:     ColorPair{Fg: 15, Bg: 160},
	}
	out, _, err := Render(RenderArgs{CWD: "/tmp", Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	// Powerline mode should have ANSI codes
	if !strings.Contains(out, "\x1b[") {
		t.Error("powerline mode should contain ANSI codes")
	}
}

func TestRender_additionalModules(t *testing.T) {
	cfg := defaultCfg()
	cfg.Order = []string{"dir", "exit"}
	cfg.EnableDir = true
	cfg.EnableExit = true
	cfg.EnableGit = false
	cfg.EnableDuration = false

	_, info, err := Render(RenderArgs{CWD: "/tmp", Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range info.Modules {
		if name != "dir" && name != "exit" {
			t.Errorf("unexpected module %q", name)
		}
	}
}

func TestConfigView_Enabled(t *testing.T) {
	cfg := ConfigView{
		EnableDir: true, EnableGit: false,
		EnableVenv: true, EnableKube: false,
	}
	if !cfg.Enabled("dir") {
		t.Error("dir should be enabled")
	}
	if cfg.Enabled("git") {
		t.Error("git should be disabled")
	}
	if !cfg.Enabled("venv") {
		t.Error("venv should be enabled")
	}
	if cfg.Enabled("kube") {
		t.Error("kube should be disabled")
	}
	if cfg.Enabled("nonexistent") {
		t.Error("nonexistent module should be disabled")
	}
}

func TestPowerlinePalette_For(t *testing.T) {
	p := PowerlinePalette{
		Dir: ColorPair{Fg: 1, Bg: 2},
	}
	cp := p.For("dir")
	if cp.Fg != 1 || cp.Bg != 2 {
		t.Errorf("dir: got %+v", cp)
	}
	// unknown returns a default
	cp2 := p.For("unknown")
	if cp2.Fg == 0 {
		t.Error("unknown module should return non-zero defaults")
	}
}
