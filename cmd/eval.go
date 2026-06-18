package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/prompts"
	"kurt_v1/internal/think"
)

func evalCmd() *cobra.Command {
	var provider, model, baseURL, host string
	var promptName string
	var outputFile string
	var concurrency int
	var maxRows int

	c := &cobra.Command{
		Use:   "eval <csv-file>",
		Short: "Batch-evaluate a prompt template against a CSV dataset",
		Long: `Run a saved prompt template against every row in a CSV file.
Column names become template variables (e.g. {{.question}}, {{.code}}).
Results are written to a new CSV with added columns:
  kurt_output, kurt_latency_ms

Examples:
  kurt eval data.csv --prompt review
  kurt eval questions.csv --prompt qa --output results.csv
  kurt eval bench.csv --prompt summarize --max-rows 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputCSV := args[0]
			if promptName == "" {
				return fmt.Errorf("--prompt is required")
			}

			p, err := prompts.Get(promptName)
			if err != nil {
				return fmt.Errorf("prompt %q: %w", promptName, err)
			}

			f, err := os.Open(inputCSV)
			if err != nil {
				return fmt.Errorf("opening %s: %w", inputCSV, err)
			}
			defer f.Close()

			r := csv.NewReader(f)
			headers, err := r.Read()
			if err != nil {
				return fmt.Errorf("reading CSV headers: %w", err)
			}

			rows, err := r.ReadAll()
			if err != nil {
				return fmt.Errorf("reading CSV rows: %w", err)
			}
			if maxRows > 0 && len(rows) > maxRows {
				rows = rows[:maxRows]
			}

			cfg, _, _ := config.Load()
			prov, err := think.New(think.ProviderConfig{
				Name:    firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama"),
				Model:   firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model),
				BaseURL: firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL),
				Host:    firstOf(host, cfg.Think.Host),
			})
			if err != nil {
				return err
			}

			// Set up output
			var outW io.Writer = os.Stdout
			if outputFile != "" {
				outF, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("creating output file: %w", err)
				}
				defer outF.Close()
				outW = outF
			}

			w := csv.NewWriter(outW)
			outHeaders := append(headers, "kurt_output", "kurt_latency_ms")
			if err := w.Write(outHeaders); err != nil {
				return err
			}

			total := len(rows)
			for i, row := range rows {
				vars := map[string]string{}
				for j, h := range headers {
					if j < len(row) {
						vars[h] = row[j]
					}
				}

				rendered, err := prompts.Render(p.Template, "", vars)
				if err != nil {
					fmt.Fprintf(os.Stderr, "row %d: render error: %v\n", i+1, err)
					outRow := append(row, "ERROR: "+err.Error(), "0")
					_ = w.Write(outRow)
					continue
				}

				msgs := []think.ChatMsg{{Role: "user", Content: rendered}}
				var buf bytes.Buffer
				start := time.Now()
				streamErr := prov.ChatStream(msgs, &buf)
				latency := time.Since(start).Milliseconds()

				var output string
				if streamErr != nil {
					output = "ERROR: " + streamErr.Error()
				} else {
					output = strings.TrimSpace(buf.String())
				}

				outRow := append(row, output, fmt.Sprintf("%d", latency))
				if err := w.Write(outRow); err != nil {
					return err
				}
				w.Flush()

				fmt.Fprintf(os.Stderr, "[%d/%d] %dms\n", i+1, total, latency)
			}

			_ = concurrency // reserved for future parallel mode
			w.Flush()
			return w.Error()
		},
	}

	c.Flags().StringVar(&promptName, "prompt", "", "Saved prompt template name (required)")
	c.Flags().StringVar(&provider, "provider", "", "LLM provider override")
	c.Flags().StringVar(&model, "model", "", "Model override")
	c.Flags().StringVar(&baseURL, "base-url", "", "API base URL override")
	c.Flags().StringVar(&host, "host", "", "Ollama host override")
	c.Flags().StringVar(&outputFile, "output", "", "Output CSV file (default: stdout)")
	c.Flags().IntVar(&concurrency, "concurrency", 1, "Parallel workers (reserved)")
	c.Flags().IntVar(&maxRows, "max-rows", 0, "Limit rows processed (0 = all)")
	return c
}
