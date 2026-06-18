package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check kurt environment and report what's working",
		RunE: func(cmd *cobra.Command, args []string) error {
			passed := 0
			total := 0

			check := func(ok bool, label, detail string) {
				total++
				mark := "x"
				if ok {
					mark = "v"
					passed++
				}
				if detail != "" {
					fmt.Printf("%s %s: %s\n", mark, label, detail)
				} else {
					fmt.Printf("%s %s\n", mark, label)
				}
			}

			// 1. Config file
			cfgPath := strings.TrimSpace(os.Getenv("KURT_CONFIG"))
			if cfgPath == "" {
				if p, err := config.DefaultPath(); err == nil {
					cfgPath = p
				}
			}
			cfgExists := false
			if cfgPath != "" {
				if _, err := os.Stat(cfgPath); err == nil {
					cfgExists = true
				}
			}
			if cfgExists {
				check(true, "config", cfgPath)
			} else {
				check(false, "config", fmt.Sprintf("not found at %s", cfgPath))
			}

			// 2. Ollama
			ollamaOK, ollamaDetail := checkOllama()
			check(ollamaOK, "ollama", ollamaDetail)

			// 3-5. API keys
			checkEnvKey := func(name string) {
				if v := strings.TrimSpace(os.Getenv(name)); v != "" {
					check(true, name, "set")
				} else {
					check(false, name, "unset")
				}
			}
			checkEnvKey("ANTHROPIC_API_KEY")
			checkEnvKey("OPENAI_API_KEY")
			checkEnvKey("GROQ_API_KEY")

			// 6. Shell integration
			if _, ok := os.LookupEnv("KURT_LAST_EXIT"); ok {
				check(true, "shell", "init hooked")
			} else {
				check(false, "shell", "not hooked (run: eval $(kurt init zsh))")
			}

			// 7. Data directory
			dataDir, dataDetail, dataOK := checkDataDir()
			if dataOK {
				check(true, "data dir", dataDir)
			} else {
				check(false, "data dir", dataDetail)
			}

			// 8. Git
			if out, err := exec.Command("git", "--version").Output(); err == nil {
				check(true, "git", strings.TrimSpace(string(out)))
			} else {
				check(false, "git", "not found")
			}

			fmt.Printf("\n%d/%d checks passed\n", passed, total)
			return nil
		},
	}
}

func checkOllama() (bool, string) {
	client := &http.Client{Timeout: 2 * time.Second}
	host := strings.TrimSpace(os.Getenv("KURT_OLLAMA_HOST"))
	if host == "" {
		host = "http://127.0.0.1:11434"
	}
	resp, err := client.Get(host + "/api/tags")
	if err != nil {
		return false, "not running"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("not running (status %d)", resp.StatusCode)
	}
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return true, "running"
	}
	names := []string{}
	for i, m := range body.Models {
		if i >= 3 {
			break
		}
		names = append(names, m.Name)
	}
	if len(names) == 0 {
		return true, "running (no models)"
	}
	return true, fmt.Sprintf("running (%s)", strings.Join(names, ", "))
}

func checkDataDir() (string, string, bool) {
	// Default: $XDG_DATA_HOME/kurt or ~/.local/share/kurt
	dir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Sprintf("could not resolve home: %v", err), false
		}
		dir = filepath.Join(h, ".local", "share")
	}
	dir = filepath.Join(dir, "kurt")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, fmt.Sprintf("not writable: %v", err), false
	}
	// Verify writable by creating and removing a temp file
	tmp, err := os.CreateTemp(dir, ".kurt-doctor-*")
	if err != nil {
		return dir, fmt.Sprintf("not writable: %v", err), false
	}
	name := tmp.Name()
	tmp.Close()
	os.Remove(name)
	return dir, "", true
}
