package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/think"
)

type benchResult struct {
	Provider string
	Model    string
	Response string
	Latency  time.Duration
	Chars    int
	Err      error
}

func benchCmd() *cobra.Command {
	var providers string
	var timeout int
	var showFull bool

	c := &cobra.Command{
		Use:   "bench <prompt>",
		Short: "Compare providers side-by-side on a prompt",
		Long: `Run the same prompt against multiple providers simultaneously
and display a latency + response comparison table.

Requires API keys to be set for the providers you specify.

Examples:
  kurt bench "explain gradient descent in 2 sentences"
  kurt bench "what is 42 * 73?" --providers openai,anthropic,groq
  kurt bench "hello" --providers ollama,lmstudio --show-full`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			cfg, _, _ := config.Load()

			providerList := parseProviders(providers, cfg)
			if len(providerList) == 0 {
				return fmt.Errorf("no providers specified — use --providers openai,anthropic,groq or configure [think] provider in config")
			}

			fmt.Fprintf(os.Stderr, "Benchmarking %d provider(s) in parallel…\n\n", len(providerList))

			results := runBench(prompt, providerList, time.Duration(timeout)*time.Second, cfg)
			printBenchTable(results, showFull)
			return nil
		},
	}

	c.Flags().StringVar(&providers, "providers", "", "Comma-separated providers (e.g. openai,anthropic,groq,ollama)")
	c.Flags().IntVar(&timeout, "timeout", 60, "Per-provider timeout in seconds")
	c.Flags().BoolVar(&showFull, "show-full", false, "Show full responses (default: first 200 chars)")
	return c
}

func parseProviders(flag string, cfg config.Config) []string {
	if flag != "" {
		var out []string
		for _, p := range strings.Split(flag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// Default: use configured provider
	p := firstOf(os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama")
	return []string{p}
}

func runBench(prompt string, providers []string, timeout time.Duration, cfg config.Config) []benchResult {
	results := make([]benchResult, len(providers))
	var wg sync.WaitGroup

	for i, name := range providers {
		wg.Add(1)
		go func(idx int, providerName string) {
			defer wg.Done()
			results[idx] = runOne(prompt, providerName, timeout, cfg)
		}(i, name)
	}
	wg.Wait()
	return results
}

func runOne(prompt, providerName string, timeout time.Duration, cfg config.Config) benchResult {
	r := benchResult{Provider: providerName}

	p, err := think.New(think.ProviderConfig{
		Name:    providerName,
		Model:   firstOf(os.Getenv("KURT_MODEL"), cfg.Think.Model),
		BaseURL: firstOf(os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL),
		Host:    cfg.Think.Host,
	})
	if err != nil {
		r.Err = err
		return r
	}

	// Reflect actual model used (best-effort)
	switch v := p.(type) {
	case *think.OpenAIProvider:
		r.Model = v.Model
	case *think.AnthropicProvider:
		r.Model = v.Model
	case *think.OllamaClient:
		r.Model = v.Model
	}

	msgs := []think.ChatMsg{
		{Role: "system", Content: "Be concise and direct."},
		{Role: "user", Content: prompt},
	}

	var buf bytes.Buffer
	start := time.Now()
	err = p.ChatStream(msgs, &buf)
	r.Latency = time.Since(start)
	r.Response = strings.TrimSpace(buf.String())
	r.Chars = utf8.RuneCountInString(r.Response)
	r.Err = err
	return r
}

func printBenchTable(results []benchResult, showFull bool) {
	maxProvider := 10
	for _, r := range results {
		if len(r.Provider) > maxProvider {
			maxProvider = len(r.Provider)
		}
	}

	header := fmt.Sprintf("%-*s  %-25s  %8s  %8s",
		maxProvider, "Provider", "Model", "Latency", "Chars")
	fmt.Println(header)
	fmt.Println(strings.Repeat("─", len(header)+2))

	for _, r := range results {
		model := r.Model
		if len(model) > 24 {
			model = model[:21] + "…"
		}
		if r.Err != nil {
			fmt.Printf("%-*s  %-25s  %8s  %8s  ERROR: %v\n",
				maxProvider, r.Provider, model, "—", "—", r.Err)
			continue
		}
		fmt.Printf("%-*s  %-25s  %8s  %8d\n",
			maxProvider, r.Provider, model,
			fmtLatency(r.Latency), r.Chars)
	}

	if showFull {
		fmt.Println()
		for _, r := range results {
			if r.Err != nil {
				continue
			}
			fmt.Printf("── %s (%s) ──\n%s\n\n", r.Provider, r.Model, r.Response)
		}
	} else {
		fmt.Println()
		for _, r := range results {
			if r.Err != nil {
				continue
			}
			preview := r.Response
			runes := []rune(preview)
			if len(runes) > 200 {
				preview = string(runes[:197]) + "…"
			}
			fmt.Printf("── %s ──\n%s\n\n", r.Provider, preview)
		}
	}
}

func fmtLatency(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
