package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/think"
)

func thinkCmd() *cobra.Command {
	var provider string
	var model string
	var baseURL string
	var host string
	var includeLast bool
	var includeGit bool
	var includeFailures bool
	var includeGitLog bool
	var includeEnv bool
	var cwd string

	c := &cobra.Command{
		Use:   "think [question]",
		Short: "Ask an AI assistant with rich local context",
		Long: `Ask an AI assistant about your current shell environment.

Supported providers (set with --provider or KURT_PROVIDER):
  ollama       Local Ollama server (default)
  openai       OpenAI API          — needs OPENAI_API_KEY
  anthropic    Anthropic Claude    — needs ANTHROPIC_API_KEY
  groq         Groq (fast)         — needs GROQ_API_KEY
  together     Together AI         — needs TOGETHER_API_KEY
  openrouter   OpenRouter          — needs OPENROUTER_API_KEY
  lmstudio     LM Studio (local)
  openai-compat  Any OpenAI-compatible URL — set --base-url

Examples:
  kurt think "why is my docker compose failing?"
  KURT_PROVIDER=groq kurt think "explain git rebase"
  kurt think --provider anthropic "review my last command"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cwd == "" {
				cwd, _ = os.Getwd()
			}

			// Resolve provider config: flag > env > config file > default
			cfg, _, _ := config.Load()
			providerName := firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama")
			resolvedModel := firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model)
			resolvedBaseURL := firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL)
			resolvedHost := firstOf(host, cfg.Think.Host)

			p, err := think.New(think.ProviderConfig{
				Name:    providerName,
				Model:   resolvedModel,
				BaseURL: resolvedBaseURL,
				Host:    resolvedHost,
			})
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}

			buildCtx := func() think.Context {
				ctx := think.Context{CWD: cwd}
				if includeLast {
					ctx.Last = think.CollectLastFromEnv()
				}
				if includeGit {
					ctx.Git = think.CollectGit(cwd)
				}
				if includeFailures {
					ctx.Failures = think.CollectFailures(8)
				}
				if includeGitLog {
					ctx.GitLog = think.CollectGitLog(cwd, 10)
				}
				if includeEnv {
					ctx.Env = think.CollectEnv()
				}
				ctx.ProjectType = think.CollectProjectType(cwd)
				return ctx
			}

			q := strings.TrimSpace(strings.Join(args, " "))
			if q != "" {
				return p.ThinkStream(buildCtx(), q, os.Stdout)
			}

			// Interactive loop
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Fprintf(os.Stderr, "kurt think [%s] — Ctrl-C to exit\n", providerName)
			for {
				fmt.Fprint(os.Stderr, "kurt> ")
				if !scanner.Scan() {
					break
				}
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if line == "/exit" || line == "/quit" {
					break
				}
				if err := p.ThinkStream(buildCtx(), line, os.Stdout); err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
				}
			}
			return nil
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "LLM provider (ollama/openai/anthropic/groq/together/openrouter/lmstudio/openai-compat)")
	c.Flags().StringVar(&model, "model", "", "Model name (overrides provider default)")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL (for openai-compat providers)")
	c.Flags().StringVar(&host, "host", "", "Ollama host (ollama provider only)")
	c.Flags().BoolVar(&includeLast, "last", true, "Include last command context")
	c.Flags().BoolVar(&includeGit, "git", true, "Include git branch/dirty status")
	c.Flags().BoolVar(&includeFailures, "failures", true, "Include recent failure history")
	c.Flags().BoolVar(&includeGitLog, "git-log", true, "Include recent git commits")
	c.Flags().BoolVar(&includeEnv, "env", true, "Include relevant env vars")
	c.Flags().StringVar(&cwd, "cwd", "", "Working directory context (defaults to current)")
	return c
}

// firstOf returns the first non-empty string from the list.
func firstOf(vals ...string) string {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func envDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
