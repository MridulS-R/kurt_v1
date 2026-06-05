package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/prompt"
)

func rpromptCmd() *cobra.Command {
	var shell string
	var cwd string
	var status int
	var durationMs int64
	var noColor bool

	c := &cobra.Command{
		Use:   "rprompt",
		Short: "Render the right-side prompt (RPROMPT) for zsh",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			cfg, cfgPath, err := loadConfigView()
			if err != nil {
				return fmt.Errorf("config load failed: %w", err)
			}

			out, info, err := prompt.RenderRight(prompt.RenderRightArgs{
				Shell:      shell,
				CWD:        cwd,
				StatusCode: status,
				DurationMs: durationMs,
				NoColor:    noColor,
				Config:     cfg,
			})
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, out)

			if os.Getenv("KURT_DEBUG") == "1" {
				elapsed := time.Since(start)
				fmt.Fprintf(os.Stderr, "\n[kurt] rprompt=%s modules=%v config=%s\n", elapsed, info.Modules, cfgPath)
			}
			return nil
		},
	}

	c.Flags().StringVar(&shell, "shell", "zsh", "Target shell (zsh/bash)")
	c.Flags().StringVar(&cwd, "cwd", "", "Current working directory")
	c.Flags().IntVar(&status, "status", 0, "Last command exit code")
	c.Flags().Int64Var(&durationMs, "duration-ms", 0, "Last command duration in ms")
	c.Flags().BoolVar(&noColor, "no-color", false, "Disable ANSI colors")

	_ = c.MarkFlagRequired("cwd")
	return c
}
