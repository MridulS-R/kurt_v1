package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/prompt"
)

func promptCmd() *cobra.Command {
	var shell string
	var cwd string
	var status int
	var durationMs int64

	c := &cobra.Command{
		Use:   "prompt",
		Short: "Render the prompt for the current shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			out, info, err := prompt.Render(prompt.RenderArgs{
				Shell:      shell,
				CWD:        cwd,
				StatusCode: status,
				DurationMs: durationMs,
				NoColor:    false,
			})
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, out)

			if os.Getenv("KURT_DEBUG") == "1" {
				elapsed := time.Since(start)
				fmt.Fprintf(os.Stderr, "\n[kurt] render=%s modules=%v\n", elapsed, info.Modules)
			}
			return nil
		},
	}

	c.Flags().StringVar(&shell, "shell", "zsh", "Target shell (zsh/bash)")
	c.Flags().StringVar(&cwd, "cwd", "", "Current working directory")
	c.Flags().IntVar(&status, "status", 0, "Last command exit code")
	c.Flags().Int64Var(&durationMs, "duration-ms", 0, "Last command duration in ms")

	_ = c.MarkFlagRequired("cwd")
	return c
}
