package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func explainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Explain current config/modules (placeholder in v1)",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("kurt_v1 explain: not implemented yet (planned: module timings + config dump)")
		},
	}
}
