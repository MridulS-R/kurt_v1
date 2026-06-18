package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the CLI.
func Execute() {
	root := &cobra.Command{
		Use:   "kurt",
		Short: "kurt_v1 — a fast, modular shell prompt",
	}

	root.AddCommand(promptCmd())
	root.AddCommand(rpromptCmd())
	root.AddCommand(initCmd())
	root.AddCommand(suggestCmd())
	root.AddCommand(thinkCmd())
	root.AddCommand(explainCmd())
	root.AddCommand(versionCmd())
	root.AddCommand(logFailureCmd())
	root.AddCommand(modelsCmd())
	root.AddCommand(sessionCmd())
	root.AddCommand(agentCmd())
	root.AddCommand(mcpCmd())
	root.AddCommand(pipeCmd())
	root.AddCommand(tokensCmd())
	root.AddCommand(costCmd())
	root.AddCommand(logCmdCmd())
	root.AddCommand(recallCmd())
	root.AddCommand(benchCmd())
	root.AddCommand(promptsCmd())
	root.AddCommand(visionCmd())
	root.AddCommand(evalCmd())
	root.AddCommand(ragCmd())
	root.AddCommand(configCmd())
	root.AddCommand(cacheCmd())
	root.AddCommand(diffCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(updateCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
