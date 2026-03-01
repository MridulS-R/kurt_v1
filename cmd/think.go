package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/think"
)

func thinkCmd() *cobra.Command {
	var host string
	var model string
	var includeLast bool
	var includeGit bool
	var cwd string

	c := &cobra.Command{
		Use:   "think [question]",
		Short: "Ask an AI helper (Ollama-first) with optional local context",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cwd == "" {
				cwd, _ = os.Getwd()
			}

			mkCtx := func() think.Context {
				ctx := think.Context{CWD: cwd}
				if includeLast {
					ctx.Last = think.CollectLastFromEnv()
				}
				if includeGit {
					ctx.Git = think.CollectGit(cwd)
				}
				return ctx
			}

			client := think.OllamaClient{Host: host, Model: model}

			// If a question is provided as args, do one-shot.
			q := strings.TrimSpace(strings.Join(args, " "))
			if q != "" {
				ans, err := client.Think(mkCtx(), q)
				if err != nil {
					return err
				}
				fmt.Println(ans)
				return nil
			}

			// Otherwise, run an interactive loop until Ctrl-C.
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Fprintln(os.Stderr, "kurt think (Ctrl-C to exit)")
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
				ans, err := client.Think(mkCtx(), line)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					continue
				}
				fmt.Println(ans)
			}
			return nil
		},
	}

	c.Flags().StringVar(&host, "host", getenvDefault("KURT_OLLAMA_HOST", "http://127.0.0.1:11434"), "Ollama host")
	c.Flags().StringVar(&model, "model", getenvDefault("KURT_OLLAMA_MODEL", "qwen2.5:7b-instruct"), "Ollama model")
	c.Flags().BoolVar(&includeLast, "last", true, "Include last command info (from zsh hook env vars)")
	c.Flags().BoolVar(&includeGit, "git", true, "Include git status/branch if in a repo")
	c.Flags().StringVar(&cwd, "cwd", "", "Working directory context (defaults to current)")
	return c
}

func getenvDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
