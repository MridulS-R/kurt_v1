package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/think"
)

func diffCmd() *cobra.Command {
	var provider, model, baseURL, host string
	var staged bool
	var base string
	var brief bool

	c := &cobra.Command{
		Use:   "diff",
		Short: "AI-powered git diff review",
		Long: `Run git diff and ask an LLM to review the changes.
By default reviews unstaged+staged working tree changes.

Examples:
  kurt diff                          # review working tree
  kurt diff --staged                 # review staged changes only
  kurt diff --base main              # review diff vs main branch
  kurt diff --brief                  # one-line summary per file
  kurt diff --provider anthropic`,
		RunE: func(cmd *cobra.Command, args []string) error {
			diff, err := getDiff(staged, base)
			if err != nil {
				return err
			}
			if strings.TrimSpace(diff) == "" {
				fmt.Println("No changes to review.")
				return nil
			}

			prompt := buildDiffPrompt(diff, brief)

			cfg, _, _ := config.Load()
			p, err := think.New(think.ProviderConfig{
				Name:    firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama"),
				Model:   firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model),
				BaseURL: firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL),
				Host:    firstOf(host, cfg.Think.Host),
			})
			if err != nil {
				return err
			}

			msgs := []think.ChatMsg{{Role: "user", Content: prompt}}
			return p.ChatStream(msgs, os.Stdout)
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "LLM provider override")
	c.Flags().StringVar(&model, "model", "", "Model override")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	c.Flags().StringVar(&host, "host", "", "Ollama host override")
	c.Flags().BoolVar(&staged, "staged", false, "Review staged changes only (git diff --staged)")
	c.Flags().StringVar(&base, "base", "", "Compare against this branch/commit (e.g. main)")
	c.Flags().BoolVar(&brief, "brief", false, "One-line summary per change instead of detailed review")
	return c
}

func getDiff(staged bool, base string) (string, error) {
	var gitArgs []string
	if base != "" {
		gitArgs = []string{"diff", base + "...HEAD"}
	} else if staged {
		gitArgs = []string{"diff", "--staged"}
	} else {
		gitArgs = []string{"diff", "HEAD"}
	}

	out, err := exec.Command("git", gitArgs...).Output()
	if err != nil {
		// If HEAD doesn't exist yet (empty repo), try without HEAD
		out2, err2 := exec.Command("git", "diff").Output()
		if err2 != nil {
			return "", fmt.Errorf("git diff: %w", err)
		}
		return string(out2), nil
	}
	// Also include untracked new files summary
	if !staged && base == "" {
		status, _ := exec.Command("git", "status", "--short").Output()
		if len(bytes.TrimSpace(status)) > 0 && len(bytes.TrimSpace(out)) == 0 {
			return "Status:\n" + string(status), nil
		}
	}
	return string(out), nil
}

func buildDiffPrompt(diff string, brief bool) string {
	if brief {
		return fmt.Sprintf(`Summarize the following git diff in bullet points — one line per logical change. Be concise.

%s`, diff)
	}
	return fmt.Sprintf(`Review the following git diff. For each change:
1. Describe what changed and why it matters
2. Flag any bugs, security issues, or design problems
3. Suggest improvements if any

Be direct and concise. Skip praise.

%s`, diff)
}
