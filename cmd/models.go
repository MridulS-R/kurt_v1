package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/models"
)

const reset = "\x1b[0m"
const bold = "\x1b[1m"
const dim = "\x1b[2m"
const cyan = "\x1b[36m"
const red = "\x1b[31m"
const yellow = "\x1b[33m"
const green = "\x1b[32m"
const gray = "\x1b[90m"

func modelsCmd() *cobra.Command {
	var host string

	c := &cobra.Command{
		Use:   "models",
		Short: "Manage and explore local LLM models",
		Long: `Download, remove, search, and get hardware-matched recommendations for local models.

All model management uses Ollama as the backend.
Start Ollama first: ollama serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsList(ollamaHost(host))
		},
	}
	c.PersistentFlags().StringVar(&host, "host", "", "Ollama host (default: http://127.0.0.1:11434)")

	c.AddCommand(modelsListCmd(&host))
	c.AddCommand(modelsPullCmd(&host))
	c.AddCommand(modelsRemoveCmd(&host))
	c.AddCommand(modelsSearchCmd(&host))
	c.AddCommand(modelsRecommendCmd(&host))
	return c
}

// ── list ──────────────────────────────────────────────────────────────────────

func modelsListCmd(host *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed models",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsList(ollamaHost(*host))
		},
	}
}

func runModelsList(host string) error {
	client := models.NewOllamaClient(host)
	installed, err := client.List()
	if err != nil {
		return err
	}
	sys := models.GetSysInfo()

	if len(installed) == 0 {
		fmt.Println("No models installed. Try: kurt models pull llama3.2:3b")
		return nil
	}

	// Build a set of installed tags for fit lookup
	installedSet := map[string]bool{}
	for _, m := range installed {
		installedSet[m.Name] = true
	}

	fmt.Printf("\n%s%sInstalled models%s  (%d)\n\n", bold, cyan, reset, len(installed))
	fmt.Printf("  %-36s  %-8s  %-10s  %-6s  %s\n",
		"Model", "Size", "Params", "Quant", "Fit for your system")
	fmt.Printf("  %s\n", strings.Repeat("─", 80))

	for _, m := range installed {
		sizeStr := formatSize(m.Size)
		params := m.Details.ParameterSize
		quant := m.Details.QuantizationLevel

		// Find catalog entry for fit assessment
		fit, found := fitFromCatalog(m.Name, sys)
		fitStr := ""
		if found {
			fitStr = fit.Color() + fit.Label() + " fits" + reset
			if fit == models.FitTight {
				fitStr = fit.Color() + fit.Label() + " tight" + reset
			} else if fit == models.FitTooBig {
				fitStr = fit.Color() + fit.Label() + " too large" + reset
			}
		}

		fmt.Printf("  %-36s  %-8s  %-10s  %-6s  %s\n",
			m.Name, sizeStr, params, quant, fitStr)
	}
	fmt.Printf("\n  System: %s%s%s  |  %s%.0f GB RAM%s  |  %d CPU cores\n\n",
		bold, sys.CPUBrand, reset,
		cyan, sys.TotalRAMGB, reset,
		sys.CPUCores)
	return nil
}

// ── pull ──────────────────────────────────────────────────────────────────────

func modelsPullCmd(host *string) *cobra.Command {
	return &cobra.Command{
		Use:   "pull <model>",
		Short: "Download a model from Ollama",
		Example: `  kurt models pull llama3.2:3b
  kurt models pull qwen2.5:7b-instruct
  kurt models pull deepseek-r1:7b`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			client := models.NewOllamaClient(ollamaHost(*host))

			// Check system fit before pulling
			sys := models.GetSysInfo()
			for _, e := range models.Catalog {
				if e.Tag == name || strings.HasPrefix(name, strings.Split(e.Tag, ":")[0]) {
					fit := models.ModelFit(e, sys)
					if fit == models.FitTooBig {
						fmt.Printf("%s⚠ Warning:%s %s needs ~%.0f GB RAM, you have %.0f GB total.\n",
							yellow, reset, e.Tag, e.RAMgb, sys.TotalRAMGB)
						fmt.Print("Continue anyway? [y/N] ")
						var ans string
						fmt.Scanln(&ans)
						if strings.ToLower(ans) != "y" {
							return nil
						}
					} else if fit == models.FitTight {
						fmt.Printf("%s⚡ Note:%s %s needs ~%.0f GB — tight on your %.0f GB system.\n",
							yellow, reset, e.Tag, e.RAMgb, sys.TotalRAMGB)
					}
					break
				}
			}

			fmt.Printf("\nPulling %s%s%s...\n", bold, name, reset)
			if err := client.Pull(name, os.Stdout); err != nil {
				return err
			}
			fmt.Printf("\n%s✓%s  %s downloaded successfully.\n\n", green, reset, name)
			return nil
		},
	}
}

// ── remove ────────────────────────────────────────────────────────────────────

func modelsRemoveCmd(host *string) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:     "remove <model>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove an installed model",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !force {
				fmt.Printf("Remove %s%s%s? [y/N] ", bold, name, reset)
				var ans string
				fmt.Scanln(&ans)
				if strings.ToLower(ans) != "y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}
			client := models.NewOllamaClient(ollamaHost(*host))
			if err := client.Remove(name); err != nil {
				return err
			}
			fmt.Printf("%s✓%s  Removed %s\n", green, reset, name)
			return nil
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return c
}

// ── search ────────────────────────────────────────────────────────────────────

func modelsSearchCmd(host *string) *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Browse the model catalog (filter by name or use-case)",
		Example: `  kurt models search
  kurt models search coding
  kurt models search reasoning
  kurt models search qwen`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			results := models.Search(query)
			sys := models.GetSysInfo()

			// Load installed to mark them
			client := models.NewOllamaClient(ollamaHost(*host))
			installed := map[string]bool{}
			if list, err := client.List(); err == nil {
				for _, m := range list {
					installed[m.Name] = true
				}
			}

			title := "All models"
			if query != "" {
				title = fmt.Sprintf("Results for %q", query)
			}
			fmt.Printf("\n%s%s%s  (%d)\n", bold, title, reset, len(results))
			fmt.Printf("  System: %.0f GB RAM — %s%s\n\n", sys.TotalRAMGB, sys.CPUBrand, reset)

			// Group by fit
			type row struct {
				fit models.Fit
				e   models.CatalogEntry
			}
			var rows []row
			for _, e := range results {
				rows = append(rows, row{models.ModelFit(e, sys), e})
			}
			sort.Slice(rows, func(i, j int) bool {
				if rows[i].fit != rows[j].fit {
					return rows[i].fit < rows[j].fit
				}
				return rows[i].e.RAMgb < rows[j].e.RAMgb
			})

			fmt.Printf("  %-34s  %-6s  %-5s  %-10s  %s\n",
				"Tag", "RAM", "Speed", "Use cases", "Description")
			fmt.Printf("  %s\n", strings.Repeat("─", 90))

			lastFit := models.Fit(-1)
			for _, r := range rows {
				if r.fit != lastFit {
					lastFit = r.fit
					switch r.fit {
					case models.FitGreat, models.FitGood:
						fmt.Printf("\n  %s✓ Fits on your system%s\n", green, reset)
					case models.FitTight:
						fmt.Printf("\n  %s⚡ Tight fit%s\n", yellow, reset)
					case models.FitTooBig:
						fmt.Printf("\n  %s✗ Too large%s\n", red, reset)
					}
				}
				instMark := "  "
				if installed[r.e.Tag] {
					instMark = green + "● " + reset
				}
				usecases := strings.Join(r.e.UseCases, ", ")
				fmt.Printf("  %s%-34s  %4.0fGB  %-5s  %-10s  %s%s%s\n",
					instMark,
					r.e.Tag,
					r.e.RAMgb,
					speedEmoji(r.e.Speed),
					usecases,
					dim, r.e.Description, reset)
			}
			fmt.Printf("\n  %s●%s = installed   Pull a model: kurt models pull <tag>\n\n", green, reset)
			return nil
		},
	}
}

// ── recommend ────────────────────────────────────────────────────────────────

func modelsRecommendCmd(host *string) *cobra.Command {
	return &cobra.Command{
		Use:   "recommend",
		Short: "Show models best suited to your hardware",
		RunE: func(cmd *cobra.Command, args []string) error {
			sys := models.GetSysInfo()
			client := models.NewOllamaClient(ollamaHost(*host))
			installed := map[string]bool{}
			if list, err := client.List(); err == nil {
				for _, m := range list {
					installed[m.Name] = true
				}
			}

			fmt.Printf("\n%s%sSystem Analysis%s\n\n", bold, cyan, reset)

			gpuLabel := "CPU only"
			if sys.HasGPU {
				gpuLabel = "Metal GPU (unified memory)"
			}
			fmt.Printf("  Chip      %s%s%s\n", bold, sys.CPUBrand, reset)
			fmt.Printf("  RAM       %s%.0f GB%s\n", bold, sys.TotalRAMGB, reset)
			fmt.Printf("  Free RAM  ~%.1f GB\n", sys.AvailableRAMGB)
			fmt.Printf("  CPU       %d cores\n", sys.CPUCores)
			fmt.Printf("  GPU       %s\n", gpuLabel)

			// Usable RAM for models
			usable := sys.TotalRAMGB - 2.0
			fmt.Printf("\n  Usable for models: ~%.0f GB (%.0f GB total − 2 GB OS overhead)\n", usable, sys.TotalRAMGB)

			// Recommendations
			fmt.Printf("\n%s%sRecommendations%s\n", bold, cyan, reset)

			var great, good, tight, tooBig []models.CatalogEntry
			for _, e := range models.Catalog {
				switch models.ModelFit(e, sys) {
				case models.FitGreat:
					great = append(great, e)
				case models.FitGood:
					good = append(good, e)
				case models.FitTight:
					tight = append(tight, e)
				default:
					tooBig = append(tooBig, e)
				}
			}

			printGroup := func(label, color string, entries []models.CatalogEntry) {
				if len(entries) == 0 {
					return
				}
				fmt.Printf("\n  %s%s%s\n", color, label, reset)
				for _, e := range entries {
					mark := "  "
					if installed[e.Tag] {
						mark = green + "● " + reset
					}
					fmt.Printf("  %s%-34s  %4.0fGB  %-5s  %s%s\n",
						mark, e.Tag, e.RAMgb, speedEmoji(e.Speed),
						dim+e.Description+reset, "")
				}
			}

			printGroup("✓ Great fit — runs comfortably", green, great)
			printGroup("✓ Good fit", green, good)
			printGroup("⚡ Tight — may swap, slower", yellow, tight)
			printGroup("✗ Too large for your RAM", red, tooBig[:min(len(tooBig), 4)])

			fmt.Printf("\n  %s●%s = already installed\n", green, reset)
			fmt.Printf("  Pull any model: %skurt models pull <tag>%s\n\n", bold, reset)
			return nil
		},
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ollamaHost(flag string) string {
	if flag != "" {
		return flag
	}
	if v := os.Getenv("KURT_OLLAMA_HOST"); v != "" {
		return v
	}
	return "http://127.0.0.1:11434"
}

func formatSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/(1<<20))
	default:
		return fmt.Sprintf("%d KB", b/1024)
	}
}

func speedEmoji(s string) string {
	switch s {
	case "blazing":
		return "⚡⚡"
	case "very fast":
		return "⚡"
	case "fast":
		return "▶"
	case "balanced":
		return "◼"
	default:
		return "▸"
	}
}

func fitFromCatalog(name string, sys models.SysInfo) (models.Fit, bool) {
	for _, e := range models.Catalog {
		if e.Tag == name || strings.HasPrefix(name, strings.Split(e.Tag, ":")[0]) {
			return models.ModelFit(e, sys), true
		}
	}
	return 0, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
