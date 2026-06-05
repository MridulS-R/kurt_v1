package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kurt version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("kurt", Version)
		},
	}
}
