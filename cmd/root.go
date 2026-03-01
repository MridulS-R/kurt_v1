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
	root.AddCommand(initCmd())
	root.AddCommand(explainCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
