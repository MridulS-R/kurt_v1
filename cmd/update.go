package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	var checkOnly bool

	c := &cobra.Command{
		Use:   "update",
		Short: "Self-update the kurt binary from GitHub releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			binPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("could not resolve current binary: %w", err)
			}
			if resolved, err := filepath.EvalSymlinks(binPath); err == nil {
				binPath = resolved
			}

			rel, ok, err := fetchLatestRelease()
			if err != nil || !ok {
				fmt.Println("No releases found at github.com/strk/kurt — self-update not available.")
				return nil
			}

			if checkOnly {
				fmt.Println(rel.TagName)
				return nil
			}

			// Find matching asset
			wantOS := runtime.GOOS
			wantArch := runtime.GOARCH
			var assetURL, assetName string
			for _, a := range rel.Assets {
				lname := strings.ToLower(a.Name)
				if strings.Contains(lname, wantOS) && strings.Contains(lname, wantArch) {
					assetURL = a.BrowserDownloadURL
					assetName = a.Name
					break
				}
			}
			if assetURL == "" {
				fmt.Printf("No binary for %s/%s found in release.\n", wantOS, wantArch)
				return nil
			}

			// Download to a temp file alongside the target dir (same filesystem for atomic rename)
			destDir := filepath.Dir(binPath)
			tmp, err := os.CreateTemp(destDir, ".kurt-update-*")
			if err != nil {
				return fmt.Errorf("create temp file: %w", err)
			}
			tmpName := tmp.Name()
			cleanup := func() { os.Remove(tmpName) }

			client := &http.Client{Timeout: 5 * time.Minute}
			resp, err := client.Get(assetURL)
			if err != nil {
				tmp.Close()
				cleanup()
				return fmt.Errorf("download %s: %w", assetName, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				tmp.Close()
				cleanup()
				return fmt.Errorf("download %s: status %d", assetName, resp.StatusCode)
			}

			if _, err := io.Copy(tmp, resp.Body); err != nil {
				tmp.Close()
				cleanup()
				return fmt.Errorf("write temp: %w", err)
			}
			if err := tmp.Close(); err != nil {
				cleanup()
				return fmt.Errorf("close temp: %w", err)
			}
			if err := os.Chmod(tmpName, 0o755); err != nil {
				cleanup()
				return fmt.Errorf("chmod temp: %w", err)
			}

			if err := os.Rename(tmpName, binPath); err != nil {
				cleanup()
				return fmt.Errorf("replace binary: %w", err)
			}

			fmt.Printf("Updated to %s\n", rel.TagName)
			return nil
		},
	}

	c.Flags().BoolVar(&checkOnly, "check", false, "Just print the latest version and exit")
	return c
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease() (ghRelease, bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/strk/kurt/releases/latest", nil)
	if err != nil {
		return ghRelease{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return ghRelease{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ghRelease{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return ghRelease{}, false, fmt.Errorf("github status %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ghRelease{}, false, err
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return ghRelease{}, false, nil
	}
	return rel, true, nil
}
