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
			q := strings.TrimSpace(strings.Join(args, " "))
			if q == "" {
				// Read from stdin interactively
				fmt.Fprint(os.Stderr, "Question: ")
				r := bufio.NewReader(os.Stdin)
				line, _ := r.ReadString('\n')
				q = strings.TrimSpace(line)
			}
			if q == "" {
				return fmt.Errorf("empty question")
			}

			if cwd == "" {
				cwd, _ = os.Getwd()
			}

			ctx := think.Context{CWD: cwd}
			if includeLast {
				ctx.Last = think.CollectLastFromEnv()
			}
			if includeGit {
				ctx.Git = think.CollectGit(cwd)
			}

			client := think.OllamaClient{Host: host, Model: model}
			ans, err := client.Think(ctx, q)
			if err != nil {
				return err
			}
			fmt.Println(ans)
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
