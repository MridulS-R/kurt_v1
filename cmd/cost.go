package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/cost"
)

func costCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cost",
		Short: "Show API token usage and cost",
		Long: `Show cumulative LLM API spend tracked across all kurt commands.

Usage is recorded automatically from kurt think, kurt pipe, kurt agent, and kurt session.
Local Ollama calls are tracked (tokens only, $0 cost).`,
	}

	// kurt cost [--days N] [--breakdown]
	var days int
	var breakdown bool

	show := &cobra.Command{
		Use:   "show",
		Short: "Show cost summary (default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCostShow(days, breakdown)
		},
	}
	show.Flags().IntVar(&days, "days", 0, "Limit to last N days (0 = all time)")
	show.Flags().BoolVar(&breakdown, "breakdown", false, "Break down by model")
	c.AddCommand(show)

	// Make "kurt cost" with no subcommand also show
	c.Flags().IntVar(&days, "days", 0, "Limit to last N days (0 = all time)")
	c.Flags().BoolVar(&breakdown, "breakdown", false, "Break down by model")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return runCostShow(days, breakdown)
	}

	reset := &cobra.Command{
		Use:   "reset",
		Short: "Clear the cost log",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(os.Stderr, "Reset cost log? [y/N] ")
			var yn string
			fmt.Scanln(&yn)
			if strings.ToLower(strings.TrimSpace(yn)) != "y" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
			if err := cost.ClearAll(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Cost log cleared.")
			return nil
		},
	}
	c.AddCommand(reset)

	return c
}

func runCostShow(days int, breakdown bool) error {
	var since time.Time
	if days > 0 {
		since = time.Now().AddDate(0, 0, -days)
	}

	entries, err := cost.Since(since)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		if days > 0 {
			fmt.Printf("No API calls recorded in the last %d days.\n", days)
		} else {
			fmt.Println("No API calls recorded yet.")
			fmt.Println("Usage is tracked automatically from kurt think, pipe, agent, and session.")
		}
		return nil
	}

	total := cost.Summarize(entries)

	// Header
	period := "all time"
	if days > 0 {
		period = fmt.Sprintf("last %d days", days)
	}
	fmt.Printf("API usage (%s)\n", period)
	fmt.Println(strings.Repeat("─", 50))

	if breakdown {
		byModel := cost.ByModel(entries)
		keys := make([]string, 0, len(byModel))
		for k := range byModel {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return byModel[keys[i]].TotalCostUSD > byModel[keys[j]].TotalCostUSD
		})
		for _, k := range keys {
			s := byModel[k]
			fmt.Printf("  %-40s  %5d calls  %6s in  %6s out  %s\n",
				k, s.Calls,
				formatTokens(s.InputTokens), formatTokens(s.OutputTokens),
				cost.FormatCost(s.TotalCostUSD))
		}
		fmt.Println(strings.Repeat("─", 50))
	}

	// Show last 7 days breakdown
	if days == 0 || days > 7 {
		byDay := cost.ByDay(entries)
		dayKeys := make([]string, 0, len(byDay))
		for k := range byDay {
			dayKeys = append(dayKeys, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dayKeys)))
		limit := 7
		if len(dayKeys) < limit {
			limit = len(dayKeys)
		}
		if limit > 0 {
			fmt.Println("Recent days:")
			for _, d := range dayKeys[:limit] {
				s := byDay[d]
				fmt.Printf("  %s  %4d calls  %s\n", d, s.Calls, cost.FormatCost(s.TotalCostUSD))
			}
			fmt.Println(strings.Repeat("─", 50))
		}
	}

	fmt.Printf("Total: %d calls  %s in  %s out  %s\n",
		total.Calls,
		formatTokens(total.InputTokens), formatTokens(total.OutputTokens),
		cost.FormatCost(total.TotalCostUSD))

	return nil
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
