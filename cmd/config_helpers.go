package cmd

import (
	"kurt_v1/internal/config"
	"kurt_v1/internal/prompt"
)

func loadConfigView() (prompt.ConfigView, string, error) {
	cfg, path, err := config.Load()
	if err != nil {
		return prompt.ConfigView{}, path, err
	}
	cv := prompt.ConfigView{
		Style:   cfg.Style,
		TwoLine: cfg.Prompt.TwoLine == nil || *cfg.Prompt.TwoLine,
		Order:   cfg.Modules.Order,

		GitTTLms: cfg.Perf.GitTTLms,

		FgDir:           derefInt(cfg.Colors.Dir, 33),
		FgGit:           derefInt(cfg.Colors.Git, 35),
		FgDuration:      derefInt(cfg.Colors.Duration, 221),
		FgExit:          derefInt(cfg.Colors.Exit, 160),
		DirMaxDepth:     cfg.Readability.DirMaxDepth,
		DirTruncateMid:  cfg.Readability.DirTruncateMid,
		GitBranchMaxLen: cfg.Readability.GitBranchMaxLen,
		GitBranchTail:   cfg.Readability.GitBranchTail,
		ExitCompact:     cfg.Readability.ExitCompact,

		RPromptEnabled:    cfg.RPrompt.Enabled != nil && *cfg.RPrompt.Enabled,
		RPromptShowTime:   cfg.RPrompt.ShowTime != nil && *cfg.RPrompt.ShowTime,
		RPromptTimeFormat: cfg.RPrompt.TimeFormat,

		EnableDir:      cfg.Module.Dir.Enabled != nil && *cfg.Module.Dir.Enabled,
		EnableGit:      cfg.Module.Git.Enabled != nil && *cfg.Module.Git.Enabled,
		EnableDuration: cfg.Module.Duration.Enabled != nil && *cfg.Module.Duration.Enabled,
		EnableExit:     cfg.Module.Exit.Enabled != nil && *cfg.Module.Exit.Enabled,
		EnableGpu:      cfg.Module.Gpu.Enabled != nil && *cfg.Module.Gpu.Enabled,
		FgGpu:          derefInt(cfg.Module.Gpu.Color, 81),
		GpuTTLms:       derefInt64(cfg.Module.Gpu.TTLms, 2000),
		EnableVenv:    cfg.Module.Venv.Enabled != nil && *cfg.Module.Venv.Enabled,
		EnableConda:   cfg.Module.Conda.Enabled != nil && *cfg.Module.Conda.Enabled,
		EnableNode:    cfg.Module.Node.Enabled != nil && *cfg.Module.Node.Enabled,
		EnableKube:    cfg.Module.Kube.Enabled != nil && *cfg.Module.Kube.Enabled,
		EnableBattery: cfg.Module.Battery.Enabled != nil && *cfg.Module.Battery.Enabled,
		EnablePython:  cfg.Module.Python.Enabled != nil && *cfg.Module.Python.Enabled,
		EnableCloud:   cfg.Module.Cloud.Enabled != nil && *cfg.Module.Cloud.Enabled,
		EnableTime:    cfg.Module.Time.Enabled != nil && *cfg.Module.Time.Enabled,
		FgVenv:        derefInt(nil, 226),
		FgConda:       derefInt(nil, 40),
		FgNode:        derefInt(nil, 71),
		FgKube:        derefInt(nil, 33),
		FgBattery:     derefInt(nil, 220),
		FgPython:      derefInt(nil, 34),
		FgCloud:       derefInt(nil, 208),
		FgTime:        derefInt(nil, 245),
		TimeFormat:    cfg.Module.Time.Format,

		DurationMinMs: 500,
		Powerline: prompt.PowerlinePalette{
			Dir:      prompt.ColorPair{Fg: derefInt(cfg.Powerline.Dir.Fg, 15), Bg: derefInt(cfg.Powerline.Dir.Bg, 31)},
			Git:      prompt.ColorPair{Fg: derefInt(cfg.Powerline.Git.Fg, 15), Bg: derefInt(cfg.Powerline.Git.Bg, 28)},
			Duration: prompt.ColorPair{Fg: derefInt(cfg.Powerline.Duration.Fg, 16), Bg: derefInt(cfg.Powerline.Duration.Bg, 220)},
			Exit:     prompt.ColorPair{Fg: derefInt(cfg.Powerline.Exit.Fg, 15), Bg: derefInt(cfg.Powerline.Exit.Bg, 160)},
		},
	}
	if cfg.Module.Duration.MinMs != nil {
		cv.DurationMinMs = *cfg.Module.Duration.MinMs
	}
	return cv, path, nil
}
