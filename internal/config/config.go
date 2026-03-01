package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config is the user-facing configuration loaded from TOML.
// Keep it minimal in v1; we can expand safely over time.
//
// Default path: ~/.config/kurt/config.toml
// Override: KURT_CONFIG
//
// Example:
//
//  style = "minimal" # (reserved)
//
//  [prompt]
//  two_line = true
//
//  [modules]
//  order = ["dir","git","duration","exit"]
//
//  [module.duration]
//  min_ms = 500
//
//  [module.git]
//  enabled = true
//
//  [module.exit]
//  enabled = true
//
//  [module.dir]
//  enabled = true
//
//  [module.duration]
//  enabled = true
//

type Config struct {
	Style     string       `toml:"style"`
	Prompt    Prompt       `toml:"prompt"`
	RPrompt   RPromptCfg   `toml:"rprompt"`
	Perf      PerfCfg      `toml:"perf"`
	Modules   Modules      `toml:"modules"`
	Module    ModuleOpts   `toml:"module"`
	Powerline PowerlineCfg `toml:"powerline"`
}

type PerfCfg struct {
	GitTTLms int64 `toml:"git_ttl_ms"`
}

type RPromptCfg struct {
	Enabled    *bool  `toml:"enabled"`
	ShowTime   *bool  `toml:"show_time"`
	TimeFormat string `toml:"time_format"`
}

type PowerlineCfg struct {
	Dir      ColorPair `toml:"dir"`
	Git      ColorPair `toml:"git"`
	Duration ColorPair `toml:"duration"`
	Exit     ColorPair `toml:"exit"`
}

type ColorPair struct {
	Fg *int `toml:"fg"`
	Bg *int `toml:"bg"`
}

type Prompt struct {
	TwoLine bool `toml:"two_line"`
}

type Modules struct {
	Order []string `toml:"order"`
}

type ModuleOpts struct {
	Dir      BasicModule `toml:"dir"`
	Git      BasicModule `toml:"git"`
	Exit     BasicModule `toml:"exit"`
	Duration DurationMod `toml:"duration"`
}

type BasicModule struct {
	Enabled *bool `toml:"enabled"`
}

type DurationMod struct {
	Enabled *bool  `toml:"enabled"`
	MinMs   *int64 `toml:"min_ms"`
}

func Default() Config {
	// v1 defaults match current hardcoded behavior.
	bTrue := true
	min := int64(500)

	// Default Powerline palette (ANSI 256 colors)
	fg15 := 15
	fg16 := 16
	bg31 := 31
	bg28 := 28
	bg220 := 220
	bg160 := 160

	return Config{
		Style:  "minimal",
		Prompt: Prompt{TwoLine: true},
		RPrompt: RPromptCfg{
			Enabled:    &bTrue,
			ShowTime:   &bTrue,
			TimeFormat: "15:04",
		},
		Perf:    PerfCfg{GitTTLms: 1000},
		Modules: Modules{Order: []string{"dir", "git", "duration", "exit"}},
		Module: ModuleOpts{
			Dir:      BasicModule{Enabled: &bTrue},
			Git:      BasicModule{Enabled: &bTrue},
			Exit:     BasicModule{Enabled: &bTrue},
			Duration: DurationMod{Enabled: &bTrue, MinMs: &min},
		},
		Powerline: PowerlineCfg{
			Dir:      ColorPair{Fg: &fg15, Bg: &bg31},
			Git:      ColorPair{Fg: &fg15, Bg: &bg28},
			Duration: ColorPair{Fg: &fg16, Bg: &bg220},
			Exit:     ColorPair{Fg: &fg15, Bg: &bg160},
		},
	}
}

func DefaultPath() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".config", "kurt", "config.toml"), nil
}

func Load() (Config, string, error) {
	cfg := Default()
	path := strings.TrimSpace(os.Getenv("KURT_CONFIG"))
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return cfg, "", err
		}
		path = p
	}

	b, err := os.ReadFile(path)
	if err != nil {
		// Missing config is not an error.
		if errors.Is(err, os.ErrNotExist) {
			return cfg, path, nil
		}
		return cfg, path, err
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return Default(), path, err
	}

	// Fill any missing defaults in nested structs.
	merged := MergeDefaults(cfg)
	return merged, path, nil
}

func MergeDefaults(user Config) Config {
	def := Default()
	out := user
	if strings.TrimSpace(out.Style) == "" {
		out.Style = def.Style
	}
	// Prompt defaults
	if !out.Prompt.TwoLine {
		// If user explicitly set false, keep it.
		// If it is zero value due to missing field, we can't distinguish.
		// We'll treat missing as default=true by checking if TOML had prompt at all later.
		// For v1 simplicity: if empty config file, default is true anyway.
	}
	if out.Modules.Order == nil || len(out.Modules.Order) == 0 {
		out.Modules.Order = def.Modules.Order
	}

	// RPrompt defaults
	if out.RPrompt.Enabled == nil {
		out.RPrompt.Enabled = def.RPrompt.Enabled
	}
	if out.RPrompt.ShowTime == nil {
		out.RPrompt.ShowTime = def.RPrompt.ShowTime
	}
	if strings.TrimSpace(out.RPrompt.TimeFormat) == "" {
		out.RPrompt.TimeFormat = def.RPrompt.TimeFormat
	}

	// Module enabled flags default to true
	applyBasic := func(b BasicModule, defB BasicModule) BasicModule {
		if b.Enabled == nil {
			b.Enabled = defB.Enabled
		}
		return b
	}
	out.Module.Dir = applyBasic(out.Module.Dir, def.Module.Dir)
	out.Module.Git = applyBasic(out.Module.Git, def.Module.Git)
	out.Module.Exit = applyBasic(out.Module.Exit, def.Module.Exit)

	if out.Module.Duration.Enabled == nil {
		out.Module.Duration.Enabled = def.Module.Duration.Enabled
	}
	if out.Module.Duration.MinMs == nil {
		out.Module.Duration.MinMs = def.Module.Duration.MinMs
	}

	// Perf defaults
	if out.Perf.GitTTLms <= 0 {
		out.Perf.GitTTLms = def.Perf.GitTTLms
	}

	// Powerline palette defaults
	applyPair := func(u ColorPair, d ColorPair) ColorPair {
		if u.Fg == nil {
			u.Fg = d.Fg
		}
		if u.Bg == nil {
			u.Bg = d.Bg
		}
		return u
	}
	out.Powerline.Dir = applyPair(out.Powerline.Dir, def.Powerline.Dir)
	out.Powerline.Git = applyPair(out.Powerline.Git, def.Powerline.Git)
	out.Powerline.Duration = applyPair(out.Powerline.Duration, def.Powerline.Duration)
	out.Powerline.Exit = applyPair(out.Powerline.Exit, def.Powerline.Exit)

	if out.Prompt.TwoLine == false {
		// keep false if user set it
	} else {
		// if unset, default already true
		// (can't detect unset cleanly without pointers; acceptable for v1)
		if user.Prompt.TwoLine == false {
			// ambiguous, do nothing
		}
	}

	return out
}
