package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/memory"
	"kurt_v1/internal/think"
)

func logCmdCmd() *cobra.Command {
	var exitCode int
	var cwd string
	var durationMs int64
	var gitBranch string

	c := &cobra.Command{
		Use:    "log-cmd [command]",
		Short:  "Log a shell command to kurt memory (called by shell hooks)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			command := strings.TrimSpace(strings.Join(args, " "))
			if command == "" {
				return nil
			}
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			if gitBranch == "" {
				if g := think.CollectGit(cwd); g != nil {
					gitBranch = g.Branch
				}
			}
			_ = memory.Log(command, cwd, gitBranch, exitCode, durationMs)
			return nil
		},
	}
	c.Flags().IntVar(&exitCode, "exit", 0, "Exit code")
	c.Flags().StringVar(&cwd, "cwd", "", "Working directory")
	c.Flags().Int64Var(&durationMs, "duration-ms", 0, "Command duration in ms")
	c.Flags().StringVar(&gitBranch, "git-branch", "", "Git branch (auto-detected if omitted)")
	return c
}
