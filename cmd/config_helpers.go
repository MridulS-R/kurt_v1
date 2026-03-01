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
		TwoLine: cfg.Prompt.TwoLine,
		Order:   cfg.Modules.Order,

		EnableDir:      cfg.Module.Dir.Enabled != nil && *cfg.Module.Dir.Enabled,
		EnableGit:      cfg.Module.Git.Enabled != nil && *cfg.Module.Git.Enabled,
		EnableDuration: cfg.Module.Duration.Enabled != nil && *cfg.Module.Duration.Enabled,
		EnableExit:     cfg.Module.Exit.Enabled != nil && *cfg.Module.Exit.Enabled,

		DurationMinMs: 500,
	}
	if cfg.Module.Duration.MinMs != nil {
		cv.DurationMinMs = *cfg.Module.Duration.MinMs
	}
	return cv, path, nil
}
