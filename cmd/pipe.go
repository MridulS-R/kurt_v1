package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/think"
)

func pipeCmd() *cobra.Command {
	var provider, model, baseURL, host string
	var maxBytes int

	c := &cobra.Command{
		Use:   "pipe <instruction>",
		Short: "Pipe stdin through an LLM with an instruction",
		Long: `Read from stdin and ask the LLM to process it with your instruction.

Examples:
  git diff | kurt pipe "write a commit message"
  cat error.log | kurt pipe "diagnose root cause and suggest fixes"
  kubectl logs my-pod | kurt pipe "summarize errors"
  cat main.go | kurt pipe "review for security issues"
  curl -s api/status | kurt pipe "is anything wrong?"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instruction := strings.TrimSpace(strings.Join(args, " "))

			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return fmt.Errorf("stdin is a terminal — pipe some input: git diff | kurt pipe %q", instruction)
			}

			input, err := io.ReadAll(io.LimitReader(os.Stdin, int64(maxBytes)))
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			if strings.TrimSpace(string(input)) == "" {
				return fmt.Errorf("stdin is empty")
			}

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

			content := strings.TrimRight(string(input), "\n")
			msgs := []think.ChatMsg{
				{Role: "system", Content: "You are a practical command-line assistant. Be concise and direct. Respond only with the result — no preamble."},
				{Role: "user", Content: fmt.Sprintf("Input:\n```\n%s\n```\n\n%s", content, instruction)},
			}
			return p.ChatStream(msgs, os.Stdout)
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "LLM provider (default: from config or ollama)")
	c.Flags().StringVar(&model, "model", "", "Model override")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	c.Flags().StringVar(&host, "host", "", "Ollama host override")
	c.Flags().IntVar(&maxBytes, "max-bytes", 100*1024, "Maximum stdin bytes to read")
	return c
}
