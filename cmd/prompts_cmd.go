package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/prompts"
	"kurt_v1/internal/think"
)

func promptsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "prompts",
		Short: "Manage reusable prompt templates",
		Long: `Store, manage, and run reusable prompt templates.

Templates support Go template syntax. The special variable {{.input}}
is populated from stdin or the --input flag.

Examples:
  kurt prompts add commit "Write a concise git commit message for:\n\n{{.input}}"
  git diff | kurt prompts run commit

  kurt prompts add review "Review this {{.lang}} code"
  cat auth.go | kurt prompts run review lang=go`,
	}

	c.AddCommand(promptsAddCmd())
	c.AddCommand(promptsListCmd())
	c.AddCommand(promptsRunCmd())
	c.AddCommand(promptsShowCmd())
	c.AddCommand(promptsRemoveCmd())
	return c
}

func promptsAddCmd() *cobra.Command {
	var description string
	var file string

	c := &cobra.Command{
		Use:   "add <name> [template]",
		Short: "Add or update a prompt template",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var tmpl string
			if file != "" {
				b, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				tmpl = string(b)
			} else if len(args) > 1 {
				tmpl = strings.Join(args[1:], " ")
				// Unescape \n sequences in shell arguments
				tmpl = strings.ReplaceAll(tmpl, `\n`, "\n")
			} else {
				// Read from stdin
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				tmpl = string(b)
			}
			if strings.TrimSpace(tmpl) == "" {
				return fmt.Errorf("template is empty")
			}
			if err := prompts.Add(name, tmpl, description); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Saved prompt %q\n", name)
			return nil
		},
	}
	c.Flags().StringVar(&description, "description", "", "Short description of the prompt")
	c.Flags().StringVar(&file, "file", "", "Read template from file")
	return c
}

func promptsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all saved prompts",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := prompts.List()
			if err != nil {
				return err
			}
			if len(list) == 0 {
				fmt.Println("No prompts saved yet.")
				fmt.Println("Add one: kurt prompts add <name> <template>")
				return nil
			}
			maxName := 6
			for _, p := range list {
				if len(p.Name) > maxName {
					maxName = len(p.Name)
				}
			}
			fmt.Printf("%-*s  Description\n", maxName, "Name")
			fmt.Println(strings.Repeat("─", maxName+30))
			for _, p := range list {
				desc := p.Description
				if desc == "" {
					// Show first line of template as description
					lines := strings.SplitN(p.Template, "\n", 2)
					desc = lines[0]
					runes := []rune(desc)
					if len(runes) > 50 {
						desc = string(runes[:47]) + "…"
					}
				}
				fmt.Printf("%-*s  %s\n", maxName, p.Name, desc)
			}
			return nil
		},
	}
}

func promptsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Print a prompt template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := prompts.Get(args[0])
			if err != nil {
				return err
			}
			if p.Description != "" {
				fmt.Fprintf(os.Stderr, "# %s\n\n", p.Description)
			}
			fmt.Println(p.Template)
			return nil
		},
	}
}

func promptsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a saved prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := prompts.Remove(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Removed prompt %q\n", args[0])
			return nil
		},
	}
}

func promptsRunCmd() *cobra.Command {
	var provider, model, baseURL, host string
	var inputArg string
	var maxBytes int
	var cacheFlag bool
	var cacheTTL int

	c := &cobra.Command{
		Use:   "run <name> [key=value ...]",
		Short: "Run a saved prompt with optional variable substitution",
		Long: `Run a saved prompt template. The {{.input}} variable is populated
from stdin (or --input flag). Additional variables can be passed as key=value args.

Examples:
  git diff | kurt prompts run commit
  cat auth.go | kurt prompts run review lang=go
  kurt prompts run greet name=World`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			vars := parseVars(args[1:])

			p, err := prompts.Get(name)
			if err != nil {
				return err
			}

			// Get input
			var input string
			if inputArg != "" {
				input = inputArg
			} else {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					b, err := io.ReadAll(io.LimitReader(os.Stdin, int64(maxBytes)))
					if err != nil {
						return err
					}
					input = strings.TrimRight(string(b), "\n")
				}
			}

			rendered, err := prompts.Render(p.Template, input, vars)
			if err != nil {
				return err
			}

			cfg, _, _ := config.Load()
			provider2, err := think.New(think.ProviderConfig{
				Name:    firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama"),
				Model:   firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model),
				BaseURL: firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL),
				Host:    firstOf(host, cfg.Think.Host),
			})
			if err != nil {
				return err
			}

			msgs := []think.ChatMsg{
				{Role: "user", Content: rendered},
			}

			// Optionally cache the response
			if cacheFlag {
				return runWithCache(provider2, msgs, rendered, cacheTTL)
			}
			return provider2.ChatStream(msgs, os.Stdout)
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "LLM provider")
	c.Flags().StringVar(&model, "model", "", "Model override")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	c.Flags().StringVar(&host, "host", "", "Ollama host override")
	c.Flags().StringVar(&inputArg, "input", "", "Input text (overrides stdin)")
	c.Flags().IntVar(&maxBytes, "max-bytes", 100*1024, "Max stdin bytes to read")
	c.Flags().BoolVar(&cacheFlag, "cache", false, "Cache responses (reuse for identical inputs)")
	c.Flags().IntVar(&cacheTTL, "cache-ttl", 24, "Cache TTL in hours (0 = never expire)")
	return c
}

func parseVars(args []string) map[string]string {
	vars := map[string]string{}
	for _, a := range args {
		if idx := strings.Index(a, "="); idx > 0 {
			vars[a[:idx]] = a[idx+1:]
		}
	}
	return vars
}
