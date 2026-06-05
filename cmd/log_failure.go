package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/history"
)

func logFailureCmd() *cobra.Command {
	var exitCode int
	var cwd string

	c := &cobra.Command{
		Use:    "log-failure [command]",
		Short:  "Log a failed command (called from shell hooks)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			command := strings.TrimSpace(strings.Join(args, " "))
			if command == "" || exitCode == 0 {
				return nil
			}
			return history.LogFailure(command, cwd, exitCode)
		},
	}

	c.Flags().IntVar(&exitCode, "exit", 0, "Exit code of the failed command")
	c.Flags().StringVar(&cwd, "cwd", "", "Working directory")
	return c
}
