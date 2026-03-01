package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/suggest"
)

func suggestCmd() *cobra.Command {
	var buffer string
	var cwd string

	c := &cobra.Command{
		Use:   "suggest",
		Short: "Suggest command completion (history + heuristics)",
		RunE: func(cmd *cobra.Command, args []string) error {
			buffer = strings.TrimSpace(buffer)
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			out, err := suggest.Suggest(suggest.Args{Buffer: buffer, CWD: cwd})
			if err != nil {
				return err
			}
			// Print only the remainder (or nothing)
			fmt.Fprint(os.Stdout, out)
			return nil
		},
	}

	c.Flags().StringVar(&buffer, "buffer", "", "Current command line buffer")
	c.Flags().StringVar(&cwd, "cwd", "", "Working directory")
	return c
}
