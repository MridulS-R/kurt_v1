package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/memory"
)

func recallCmd() *cobra.Command {
	var failedOnly bool
	var cwdFilter bool
	var days int
	var limit int

	c := &cobra.Command{
		Use:   "recall [query]",
		Short: "Search your shell command history",
		Long: `Search commands you've run, stored by the shell hook.

Commands are saved automatically once you add the kurt log-cmd hook
(already included in: kurt init zsh).

Examples:
  kurt recall "cuda"                 search commands containing "cuda"
  kurt recall "python train"         multi-word search
  kurt recall --failed               show only failed commands
  kurt recall --cwd "git rebase"     only in current directory
  kurt recall --days 7 "docker"      last 7 days only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(strings.Join(args, " "))

			opts := memory.SearchOpts{
				FailedOnly: failedOnly,
				Limit:      limit,
			}
			if cwdFilter {
				opts.CWDFilter, _ = os.Getwd()
			}
			if days > 0 {
				opts.Since = time.Now().AddDate(0, 0, -days)
			}

			results, err := memory.Search(query, opts)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				if query != "" {
					fmt.Fprintf(os.Stderr, "No commands found matching %q\n", query)
				} else {
					fmt.Fprintln(os.Stderr, "No commands in memory yet.")
					fmt.Fprintln(os.Stderr, "Commands are logged automatically via: kurt init zsh")
				}
				return nil
			}

			for _, r := range results {
				printCommand(r)
			}
			return nil
		},
	}

	c.Flags().BoolVar(&failedOnly, "failed", false, "Only show commands that failed (exit != 0)")
	c.Flags().BoolVar(&cwdFilter, "cwd", false, "Only show commands run in the current directory")
	c.Flags().IntVar(&days, "days", 0, "Only show commands from the last N days")
	c.Flags().IntVar(&limit, "n", 20, "Maximum results to show")
	return c
}

func printCommand(c memory.Command) {
	// Age line
	age := humanAge(time.Since(c.At))
	dir := shortenPath(c.CWD)
	branch := ""
	if c.GitBranch != "" {
		branch = " (" + c.GitBranch + ")"
	}
	exitStr := ""
	if c.ExitCode != 0 {
		exitStr = fmt.Sprintf("  exit:%d", c.ExitCode)
	}
	dur := ""
	if c.DurationMs > 0 {
		dur = "  " + humanDurMs(c.DurationMs)
	}
	fmt.Printf("\x1b[2m%s  %s%s%s%s\x1b[0m\n  %s\n\n",
		age, dir, branch, exitStr, dur, c.Cmd)
}

func humanAge(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return "just now"
	}
	m := s / 60
	if m < 60 {
		return fmt.Sprintf("%dm ago", m)
	}
	h := m / 60
	if h < 24 {
		return fmt.Sprintf("%dh ago", h)
	}
	days := h / 24
	if days == 1 {
		return "yesterday"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	return d.Round(24 * time.Hour * 7).String()
}

func humanDurMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := float64(ms) / 1000
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	return fmt.Sprintf("%.0fm%.0fs", s/60, float64(int(s)%60))
}

func shortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		p = "~" + p[len(home):]
	}
	return p
}
