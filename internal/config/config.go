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
	Style   string     `toml:"style"`
	Prompt  Prompt     `toml:"prompt"`
	Modules Modules    `toml:"modules"`
	Module  ModuleOpts `toml:"module"`
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
	return Config{
		Style:   "minimal",
		Prompt:  Prompt{TwoLine: true},
		Modules: Modules{Order: []string{"dir", "git", "duration", "exit"}},
		Module: ModuleOpts{
			Dir:      BasicModule{Enabled: &bTrue},
			Git:      BasicModule{Enabled: &bTrue},
			Exit:     BasicModule{Enabled: &bTrue},
			Duration: DurationMod{Enabled: &bTrue, MinMs: &min},
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
