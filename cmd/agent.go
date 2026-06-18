package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/agent"
	"kurt_v1/internal/think"
)

func agentCmd() *cobra.Command {
	var (
		provider    string
		model       string
		baseURL     string
		host        string
		sandbox     string
		dockerImage string
		maxSteps    int
		autoExec    bool
	)

	c := &cobra.Command{
		Use:   "agent <task>",
		Short: "Run an AI agent in an isolated sandbox",
		Long: `Run an autonomous AI agent that executes shell commands in an isolated environment.

Sandbox modes:
  tmpdir   Isolated temp directory, restricted env (default, always available)
  docker   Docker container with network disabled — requires Docker Desktop

The agent loop:
  1. You describe a task
  2. The model plans and proposes shell commands
  3. Each command is shown to you for confirmation (skip with --yes)
  4. Output is fed back to the model
  5. Repeat until DONE or --steps limit

Examples:
  kurt agent "write a Python script that generates Fibonacci numbers"
  kurt agent --sandbox docker "set up a node project with express"
  kurt agent --yes "create a Makefile with build and test targets"`,

		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := strings.TrimSpace(strings.Join(args, " "))

			p, providerName, err := providerForSession(provider, model, baseURL, host)
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}

			// Build sandbox
			var sb agent.Sandbox
			switch strings.ToLower(sandbox) {
			case "docker":
				sb, err = agent.NewDockerSandbox(dockerImage)
				if err != nil {
					return fmt.Errorf("docker sandbox: %w", err)
				}
			default:
				sb, err = agent.NewTmpdirSandbox()
				if err != nil {
					return fmt.Errorf("tmpdir sandbox: %w", err)
				}
			}
			defer sb.Cleanup()

			printAgentBanner(task, sb, providerName, p)

			runner := &agent.Runner{
				Provider: p,
				Sandbox:  sb,
				MaxSteps: maxSteps,
				AutoExec: autoExec,
				Out:      os.Stdout,
				ErrOut:   os.Stderr,
				In:       bufio.NewReader(os.Stdin),
			}
			if err := runner.Run(task); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "\nSandbox at: %s\nkurt agent rm to delete, or inspect files there.\n", sb.Dir())
			return nil
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "LLM provider (ollama/openai/anthropic/...)")
	c.Flags().StringVar(&model, "model", "", "Model name")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL (openai-compat providers)")
	c.Flags().StringVar(&host, "host", "", "Ollama host")
	c.Flags().StringVar(&sandbox, "sandbox", "tmpdir", "Sandbox type: tmpdir or docker")
	c.Flags().StringVar(&dockerImage, "image", "alpine:latest", "Docker image (docker sandbox only)")
	c.Flags().IntVar(&maxSteps, "steps", 12, "Maximum agent steps before stopping")
	c.Flags().BoolVar(&autoExec, "yes", false, "Auto-execute all commands without confirmation")
	return c
}

func printAgentBanner(task string, sb agent.Sandbox, providerName string, p think.Provider) {
	_ = p
	c := confidentialityFor(providerName)

	isolation := "process-level (tmpdir, host kernel)"
	if sb.Kind() != "tmpdir" {
		isolation = "container-level (Docker, isolated kernel)"
	}

	fmt.Fprintf(os.Stderr, "\nkurt agent\n")
	fmt.Fprintf(os.Stderr, "  Task:        %s\n", task)
	fmt.Fprintf(os.Stderr, "  Provider:    %s\n", providerName)
	fmt.Fprintf(os.Stderr, "  Sandbox:     %s\n", sb.Kind())
	fmt.Fprintf(os.Stderr, "  Isolation:   %s\n", isolation)
	fmt.Fprintf(os.Stderr, "  Workspace:   %s\n", sb.Dir())
	fmt.Fprintf(os.Stderr, "  Confidentiality: %s %d%% — %s\n\n",
		confidentialityBar(c.Score), c.Score, c.Note)
}
