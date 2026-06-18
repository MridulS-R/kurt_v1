package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/cache"
)

func cacheCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cache",
		Short: "Manage the LLM response cache",
	}
	c.AddCommand(cacheListCmd())
	c.AddCommand(cacheClearCmd())
	return c
}

func cacheListCmd() *cobra.Command {
	var n int
	c := &cobra.Command{
		Use:   "list",
		Short: "List cached responses",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := cache.List(n)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("Cache is empty.")
				return nil
			}
			fmt.Printf("%-16s  %-12s  %-24s  %-20s  %s\n",
				"Key", "Provider", "Model", "Cached At", "Preview")
			fmt.Println(strings.Repeat("─", 90))
			for _, e := range entries {
				preview := strings.ReplaceAll(e.Input, "\n", " ")
				runes := []rune(preview)
				if len(runes) > 40 {
					preview = string(runes[:37]) + "…"
				}
				model := e.Model
				if len(model) > 23 {
					model = model[:20] + "…"
				}
				fmt.Printf("%-16s  %-12s  %-24s  %-20s  %s\n",
					e.Key[:8]+"…",
					e.Provider,
					model,
					e.At.Format("2006-01-02 15:04"),
					preview,
				)
			}
			return nil
		},
	}
	c.Flags().IntVar(&n, "n", 50, "Max entries to show")
	return c
}

func cacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Delete all cached responses",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cache.ClearAll(); err != nil {
				return err
			}
			fmt.Println("Cache cleared.")
			return nil
		},
	}
}
