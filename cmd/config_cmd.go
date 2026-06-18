package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
)

func configCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View and manage kurt configuration",
	}
	c.AddCommand(configViewCmd())
	c.AddCommand(configPathCmd())
	c.AddCommand(configGetCmd())
	c.AddCommand(configSetCmd())
	return c
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
}

func configViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show all active config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "# config: %s\n\n", path)

			boolStr := func(b *bool) string {
				if b == nil {
					return "unset"
				}
				if *b {
					return "true"
				}
				return "false"
			}
			intStr := func(i *int, def int) string {
				if i == nil {
					return fmt.Sprintf("%d (default)", def)
				}
				return fmt.Sprintf("%d", *i)
			}

			fmt.Printf("style             = %s\n", cfg.Style)
			fmt.Printf("prompt.two_line   = %s\n", boolStr(cfg.Prompt.TwoLine))
			fmt.Printf("\n[think]\n")
			fmt.Printf("provider          = %s\n", orDefault(cfg.Think.Provider, "ollama"))
			fmt.Printf("model             = %s\n", orDefault(cfg.Think.Model, "(provider default)"))
			fmt.Printf("base_url          = %s\n", orDefault(cfg.Think.BaseURL, "(provider default)"))
			fmt.Printf("host              = %s\n", orDefault(cfg.Think.Host, "http://127.0.0.1:11434"))
			fmt.Printf("\n[modules]\n")
			fmt.Printf("order             = [%s]\n", strings.Join(cfg.Modules.Order, ", "))
			fmt.Printf("\n[module.dir]\n")
			fmt.Printf("enabled           = %s\n", boolStr(cfg.Module.Dir.Enabled))
			fmt.Printf("\n[module.git]\n")
			fmt.Printf("enabled           = %s\n", boolStr(cfg.Module.Git.Enabled))
			fmt.Printf("\n[module.duration]\n")
			fmt.Printf("enabled           = %s\n", boolStr(cfg.Module.Duration.Enabled))
			fmt.Printf("min_ms            = %s\n", func() string {
				if cfg.Module.Duration.MinMs == nil {
					return "500 (default)"
				}
				return fmt.Sprintf("%d", *cfg.Module.Duration.MinMs)
			}())
			fmt.Printf("\n[module.gpu]\n")
			fmt.Printf("enabled           = %s\n", boolStr(cfg.Module.Gpu.Enabled))
			fmt.Printf("ttl_ms            = %s\n", func() string {
				if cfg.Module.Gpu.TTLms == nil {
					return "2000 (default)"
				}
				return fmt.Sprintf("%d", *cfg.Module.Gpu.TTLms)
			}())
			fmt.Printf("color             = %s\n", intStr(cfg.Module.Gpu.Color, 81))
			fmt.Printf("\n[module.venv]   enabled = %s\n", boolStr(cfg.Module.Venv.Enabled))
			fmt.Printf("[module.conda]  enabled = %s\n", boolStr(cfg.Module.Conda.Enabled))
			fmt.Printf("[module.node]   enabled = %s\n", boolStr(cfg.Module.Node.Enabled))
			fmt.Printf("[module.kube]   enabled = %s\n", boolStr(cfg.Module.Kube.Enabled))
			fmt.Printf("[module.battery] enabled = %s\n", boolStr(cfg.Module.Battery.Enabled))
			fmt.Printf("[module.python]  enabled = %s\n", boolStr(cfg.Module.Python.Enabled))
			fmt.Printf("[module.cloud]   enabled = %s\n", boolStr(cfg.Module.Cloud.Enabled))
			fmt.Printf("[module.time]   enabled = %s  format = %s\n",
				boolStr(cfg.Module.Time.Enabled), orDefault(cfg.Module.Time.Format, "15:04"))
			fmt.Printf("\n[perf]\n")
			fmt.Printf("git_ttl_ms        = %d\n", cfg.Perf.GitTTLms)
			fmt.Printf("\n[rprompt]\n")
			fmt.Printf("enabled           = %s\n", boolStr(cfg.RPrompt.Enabled))
			fmt.Printf("show_time         = %s\n", boolStr(cfg.RPrompt.ShowTime))
			fmt.Printf("time_format       = %s\n", cfg.RPrompt.TimeFormat)
			fmt.Printf("\n[colors]\n")
			fmt.Printf("dir=%s  git=%s  duration=%s  exit=%s\n",
				intStr(cfg.Colors.Dir, 33),
				intStr(cfg.Colors.Git, 35),
				intStr(cfg.Colors.Duration, 221),
				intStr(cfg.Colors.Exit, 160),
			)
			fmt.Printf("\n[env overrides]\n")
			for _, k := range envKeys() {
				if v := os.Getenv(k); v != "" {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}
			return nil
		},
	}
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single config value",
		Long: `Print the active value for a config key. Dot-notation supported.

Examples:
  kurt config get think.provider
  kurt config get modules.order
  kurt config get style`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			key := strings.ToLower(strings.TrimSpace(args[0]))
			val, ok := configGet(cfg, key)
			if !ok {
				return fmt.Errorf("unknown key %q — run 'kurt config view' to see all keys", key)
			}
			fmt.Println(val)
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (writes to config file)",
		Long: `Set a config key in the TOML config file. Creates the file if it doesn't exist.

Examples:
  kurt config set think.provider anthropic
  kurt config set think.model claude-sonnet-4-6
  kurt config set style powerline`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := config.Load()
			if err != nil {
				return err
			}
			key := strings.ToLower(strings.TrimSpace(args[0]))
			val := args[1]
			if err := config.SetKey(path, key, val); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Set %s = %s  (%s)\n", key, val, path)
			return nil
		},
	}
}

func configGet(cfg config.Config, key string) (string, bool) {
	switch key {
	case "style":
		return cfg.Style, true
	case "prompt.two_line":
		if cfg.Prompt.TwoLine == nil {
			return "true (default)", true
		}
		return fmt.Sprintf("%v", *cfg.Prompt.TwoLine), true
	case "think.provider":
		return orDefault(cfg.Think.Provider, "ollama"), true
	case "think.model":
		return orDefault(cfg.Think.Model, "(provider default)"), true
	case "think.base_url":
		return orDefault(cfg.Think.BaseURL, ""), true
	case "think.host":
		return orDefault(cfg.Think.Host, "http://127.0.0.1:11434"), true
	case "modules.order":
		return strings.Join(cfg.Modules.Order, ", "), true
	case "perf.git_ttl_ms":
		return fmt.Sprintf("%d", cfg.Perf.GitTTLms), true
	case "rprompt.enabled":
		if cfg.RPrompt.Enabled == nil {
			return "true (default)", true
		}
		return fmt.Sprintf("%v", *cfg.RPrompt.Enabled), true
	case "rprompt.time_format":
		return cfg.RPrompt.TimeFormat, true
	}
	return "", false
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func envKeys() []string {
	keys := []string{
		"KURT_CONFIG", "KURT_PROVIDER", "KURT_MODEL",
		"KURT_BASE_URL", "KURT_OLLAMA_HOST",
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY",
		"GROQ_API_KEY", "TOGETHER_API_KEY",
		"OPENROUTER_API_KEY",
	}
	sort.Strings(keys)
	return keys
}
