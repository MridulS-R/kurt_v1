package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func explainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Show active config, modules, and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfigView()
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "# kurt explain\n\n")
			fmt.Printf("Config file : %s\n", path)
			fmt.Printf("Style       : %s\n", cfg.Style)
			fmt.Printf("Two-line    : %v\n", cfg.TwoLine)
			fmt.Printf("\n")

			// Provider
			provName := firstOf(os.Getenv("KURT_PROVIDER"), "")
			if provName == "" {
				provName = "(from config)"
			} else {
				provName += " (KURT_PROVIDER)"
			}
			fmt.Printf("Provider    : %s\n", provName)
			fmt.Printf("Model env   : %s\n", firstOf(os.Getenv("KURT_MODEL"), "(none)"))

			// Modules
			fmt.Printf("\n%-14s  %-8s  %s\n", "Module", "Enabled", "Notes")
			fmt.Println(strings.Repeat("─", 50))

			type modRow struct {
				name    string
				enabled bool
				notes   string
			}
			rows := []modRow{
				{"dir", cfg.EnableDir, fmt.Sprintf("max_depth=%d fg=%d", cfg.DirMaxDepth, cfg.FgDir)},
				{"git", cfg.EnableGit, fmt.Sprintf("ttl=%dms fg=%d", cfg.GitTTLms, cfg.FgGit)},
				{"duration", cfg.EnableDuration, fmt.Sprintf("min=%dms fg=%d", cfg.DurationMinMs, cfg.FgDuration)},
				{"exit", cfg.EnableExit, fmt.Sprintf("compact=%v fg=%d", cfg.ExitCompact, cfg.FgExit)},
				{"gpu", cfg.EnableGpu, fmt.Sprintf("ttl=%dms fg=%d", cfg.GpuTTLms, cfg.FgGpu)},
				{"venv", cfg.EnableVenv, fmt.Sprintf("fg=%d", cfg.FgVenv)},
				{"conda", cfg.EnableConda, fmt.Sprintf("fg=%d", cfg.FgConda)},
				{"node", cfg.EnableNode, fmt.Sprintf("fg=%d", cfg.FgNode)},
				{"python", cfg.EnablePython, fmt.Sprintf("fg=%d", cfg.FgPython)},
				{"kube", cfg.EnableKube, fmt.Sprintf("fg=%d", cfg.FgKube)},
				{"cloud", cfg.EnableCloud, fmt.Sprintf("fg=%d", cfg.FgCloud)},
				{"battery", cfg.EnableBattery, fmt.Sprintf("fg=%d", cfg.FgBattery)},
				{"time", cfg.EnableTime, fmt.Sprintf("format=%q fg=%d", cfg.TimeFormat, cfg.FgTime)},
			}
			for _, r := range rows {
				status := "false"
				if r.enabled {
					status = "true "
				}
				fmt.Printf("%-14s  %-8s  %s\n", r.name, status, r.notes)
			}

			fmt.Printf("\nModule order: [%s]\n", strings.Join(cfg.Order, ", "))

			// Active env variables
			fmt.Printf("\nEnvironment:\n")
			envVars := []string{
				"KURT_CONFIG", "KURT_PROVIDER", "KURT_MODEL",
				"KURT_BASE_URL", "KURT_OLLAMA_HOST",
				"ANTHROPIC_API_KEY", "OPENAI_API_KEY",
				"VIRTUAL_ENV", "CONDA_DEFAULT_ENV",
				"AWS_PROFILE", "KUBECONFIG",
			}
			for _, k := range envVars {
				v := os.Getenv(k)
				if v == "" {
					continue
				}
				// Mask API keys
				if strings.Contains(k, "KEY") && len(v) > 8 {
					v = v[:4] + strings.Repeat("*", len(v)-8) + v[len(v)-4:]
				}
				fmt.Printf("  %s=%s\n", k, v)
			}

			return nil
		},
	}
}
