package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func explainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Explain current config/modules (placeholder in v1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfigView()
			if err != nil {
				return err
			}
			fmt.Println("config:", path)
			fmt.Println("style:", cfg.Style)
			fmt.Println("two_line:", cfg.TwoLine)
			fmt.Println("order:", cfg.Order)
			fmt.Println("enabled:")
			fmt.Println("  dir:", cfg.EnableDir)
			fmt.Println("  git:", cfg.EnableGit)
			fmt.Println("  duration:", cfg.EnableDuration, "min_ms=", cfg.DurationMinMs)
			fmt.Println("  exit:", cfg.EnableExit)
			fmt.Println("env override:")
			fmt.Println("  KURT_CONFIG:", os.Getenv("KURT_CONFIG"))
			return nil
		},
	}
}
