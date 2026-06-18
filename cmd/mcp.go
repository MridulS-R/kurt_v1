package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	mcpserver "kurt_v1/internal/mcp"
	"kurt_v1/internal/think"
)

func mcpCmd() *cobra.Command {
	var workdir string
	var provider string
	var model string
	var baseURL string
	var host string
	var shellTimeout int

	c := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
	}

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the kurt MCP server over stdio",
		Long: `Start kurt as an MCP (Model Context Protocol) server over stdio.

Connect from any MCP client by pointing its stdio transport at:
  kurt mcp serve

Claude Desktop example (~/.claude_desktop_config.json):
  {
    "mcpServers": {
      "kurt": {
        "command": "kurt",
        "args": ["mcp", "serve"]
      }
    }
  }

Available tools:
  shell_exec      Run a shell command in the working directory
  read_file       Read a file's contents
  write_file      Write content to a file
  list_directory  List files in a directory
  git_context     Get branch, status, and recent commits
  think           Ask kurt's LLM with full shell context

Available resources:
  kurt://context/shell   Working directory, project type, env vars
  kurt://context/git     Git branch, status, recent commits`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if workdir == "" {
				workdir, _ = os.Getwd()
			}
			cfg, _, _ := config.Load()
			providerName := firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama")
			resolvedModel := firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model)
			resolvedBaseURL := firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL)
			resolvedHost := firstOf(host, cfg.Think.Host)

			return mcpserver.Serve(mcpserver.Cfg{
				Workdir:         workdir,
				ShellTimeoutSec: shellTimeout,
				ProviderCfg: think.ProviderConfig{
					Name:    providerName,
					Model:   resolvedModel,
					BaseURL: resolvedBaseURL,
					Host:    resolvedHost,
				},
			})
		},
	}

	serve.Flags().StringVar(&workdir, "workdir", "", "Working directory for tool calls (default: $PWD)")
	serve.Flags().StringVar(&provider, "provider", "", "LLM provider for the think tool")
	serve.Flags().StringVar(&model, "model", "", "Model override for the think tool")
	serve.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	serve.Flags().StringVar(&host, "host", "", "Ollama host override")
	serve.Flags().IntVar(&shellTimeout, "shell-timeout", 30, "Default timeout for shell_exec in seconds")

	c.AddCommand(serve)
	return c
}
