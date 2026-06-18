package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version holds the build version, injected at release time via:
//
//	go build -ldflags "-X main.Version=v1.2.3"
var version = "dev"

// SetVersion is called from main to inject the build-time version string.
func SetVersion(v string) {
	if v != "" {
		version = v
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kurt version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("kurt", version)
		},
	}
}
