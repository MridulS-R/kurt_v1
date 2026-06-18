package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/session"
	"kurt_v1/internal/think"
)

type confidentialityInfo struct {
	Score       int    // 0–100
	Local       bool   // data stays on machine
	Note        string // human-readable reason
}

var providerConfidentiality = map[string]confidentialityInfo{
	"ollama":       {100, true, "runs on your machine — nothing leaves"},
	"lmstudio":     {100, true, "runs on your machine — nothing leaves"},
	"anthropic":    {40, false, "sent to Anthropic servers (no API training by default)"},
	"openai":       {40, false, "sent to OpenAI servers (no API training by default)"},
	"groq":         {40, false, "sent to Groq servers (no API training by default)"},
	"together":     {35, false, "sent to Together AI servers"},
	"openrouter":   {20, false, "routed through OpenRouter to a 3rd-party model"},
	"openai-compat":{35, false, "sent to a custom API endpoint"},
}

func confidentialityFor(provider string) confidentialityInfo {
	if c, ok := providerConfidentiality[strings.ToLower(provider)]; ok {
		return c
	}
	return confidentialityInfo{35, false, "sent to an unknown API endpoint"}
}

func confidentialityBar(score int) string {
	filled := score / 10
	empty := 10 - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func printSessionBanner(meta *session.Meta, msgCount int) {
	c := confidentialityFor(meta.Provider)
	dir, _ := session.DataDir()
	sessionPath := dir + "/" + meta.ID

	fmt.Fprintf(os.Stderr, "\nSession: %s\n", meta.ID)
	if meta.Name != meta.ID {
		fmt.Fprintf(os.Stderr, "  Name:    %s\n", meta.Name)
	}
	fmt.Fprintf(os.Stderr, "  Model:   %s via %s\n", meta.Model, meta.Provider)
	fmt.Fprintf(os.Stderr, "  Memory:  %s\n", sessionPath)
	fmt.Fprintf(os.Stderr, "  History: %d exchanges\n", msgCount)
	fmt.Fprintf(os.Stderr, "  Confidentiality: %s %d%% — %s\n\n",
		confidentialityBar(c.Score), c.Score, c.Note)
}

func sessionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "session",
		Short: "Disposable AI memory containers — persistent multi-turn conversations",
	}
	c.AddCommand(sessionNewCmd())
	c.AddCommand(sessionLsCmd())
	c.AddCommand(sessionAttachCmd())
	c.AddCommand(sessionShowCmd())
	c.AddCommand(sessionRmCmd())
	return c
}

// providerForSession builds a Provider from flags/env/config, same as thinkCmd.
func providerForSession(provider, model, baseURL, host string) (think.Provider, string, error) {
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
	return p, providerName, err
}

func sessionNewCmd() *cobra.Command {
	var provider, model, baseURL, host, name string

	c := &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new session and enter the chat REPL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				name = args[0]
			}
			p, providerName, err := providerForSession(provider, model, baseURL, host)
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}
			cwd, _ := os.Getwd()
			meta, err := session.New(name, providerName, model, cwd)
			if err != nil {
				return fmt.Errorf("create session: %w", err)
			}
			printSessionBanner(meta, 0)
			return runSessionREPL(meta, p)
		},
	}
	c.Flags().StringVar(&provider, "provider", "", "LLM provider")
	c.Flags().StringVar(&model, "model", "", "Model name")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL")
	c.Flags().StringVar(&host, "host", "", "Ollama host")
	c.Flags().StringVar(&name, "name", "", "Session name (alias for positional arg)")
	return c
}

func sessionLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := session.List()
			if err != nil {
				return err
			}
			if len(all) == 0 {
				fmt.Println("no sessions")
				return nil
			}
			for _, m := range all {
				count := session.MsgCount(m.ID)
				c := confidentialityFor(m.Provider)
				fmt.Printf("%-22s  %-12s  %3d%%  %d msgs  %s\n",
					m.ID, m.Provider, c.Score, count,
					m.UpdatedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}

func sessionAttachCmd() *cobra.Command {
	var provider, model, baseURL, host string

	c := &cobra.Command{
		Use:   "attach <id>",
		Short: "Resume an existing session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := session.Load(args[0])
			if err != nil {
				return err
			}
			// Prefer the session's original provider unless overridden.
			if provider == "" {
				provider = meta.Provider
			}
			if model == "" {
				model = meta.Model
			}
			p, _, err := providerForSession(provider, model, baseURL, host)
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}
			msgs, _ := session.History(meta.ID)
			printSessionBanner(meta, len(msgs))
			return runSessionREPL(meta, p)
		},
	}
	c.Flags().StringVar(&provider, "provider", "", "Override provider")
	c.Flags().StringVar(&model, "model", "", "Override model")
	c.Flags().StringVar(&baseURL, "base-url", "", "Override API base URL")
	c.Flags().StringVar(&host, "host", "", "Override Ollama host")
	return c
}

func sessionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Print conversation history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := session.Load(args[0])
			if err != nil {
				return err
			}
			msgs, err := session.History(meta.ID)
			if err != nil {
				return err
			}
			fmt.Printf("=== %s (%s) ===\n", meta.ID, meta.Name)
			for _, m := range msgs {
				prefix := "you"
				if m.Role == "assistant" {
					prefix = "kurt"
				} else if m.Role == "system" {
					prefix = "sys "
				}
				fmt.Printf("\n[%s] %s\n%s\n", prefix, m.Timestamp.Format("15:04:05"), m.Content)
			}
			return nil
		},
	}
}

func sessionRmCmd() *cobra.Command {
	var all bool

	c := &cobra.Command{
		Use:   "rm [id]",
		Short: "Destroy a session (or all sessions with --all)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return session.DestroyAll()
			}
			if len(args) == 0 {
				return fmt.Errorf("provide a session id or --all")
			}
			meta, err := session.Load(args[0])
			if err != nil {
				return err
			}
			if err := session.Destroy(meta.ID); err != nil {
				return err
			}
			fmt.Println("removed", meta.ID)
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "Destroy all sessions")
	return c
}

// runSessionREPL runs the interactive chat loop for a session.
// Commands: /exit, /quit, /clear, /show, /destroy
func runSessionREPL(meta *session.Meta, p think.Provider) error {
	msgs, err := session.History(meta.ID)
	if err != nil {
		return err
	}

	// Build think.ChatMsg slice from stored history.
	history := make([]think.ChatMsg, 0, len(msgs))
	for _, m := range msgs {
		history = append(history, think.ChatMsg{Role: m.Role, Content: m.Content})
	}

	fmt.Fprintln(os.Stderr, "  /clear  — clear history   /show — print history   /destroy — delete session   /exit — quit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case "/exit", "/quit":
			return nil
		case "/clear":
			if err := session.ClearHistory(meta.ID); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
			}
			history = history[:0]
			fmt.Fprintln(os.Stderr, "history cleared")
			continue
		case "/show":
			for _, m := range history {
				prefix := "you"
				if m.Role == "assistant" {
					prefix = "kurt"
				}
				fmt.Printf("[%s] %s\n", prefix, m.Content)
			}
			continue
		case "/destroy":
			_ = session.Destroy(meta.ID)
			fmt.Fprintln(os.Stderr, "session destroyed")
			return nil
		}

		// Save user message.
		if err := session.Append(meta.ID, session.Message{Role: "user", Content: line}); err != nil {
			fmt.Fprintln(os.Stderr, "warn: could not save message:", err)
		}
		history = append(history, think.ChatMsg{Role: "user", Content: line})

		// Stream assistant reply, capturing it simultaneously.
		fmt.Fprint(os.Stderr, "kurt> ")
		var buf bytes.Buffer
		w := io.MultiWriter(os.Stdout, &buf)
		if err := p.ChatStream(history, w); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			// Remove the user message we just appended so it doesn't corrupt history.
			history = history[:len(history)-1]
			continue
		}

		reply := strings.TrimSpace(buf.String())
		if reply != "" {
			if err := session.Append(meta.ID, session.Message{Role: "assistant", Content: reply}); err != nil {
				fmt.Fprintln(os.Stderr, "warn: could not save reply:", err)
			}
			history = append(history, think.ChatMsg{Role: "assistant", Content: reply})
		}

		_ = session.Touch(meta.ID)
	}
	return nil
}
