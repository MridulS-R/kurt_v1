package modules

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CloudModule shows the active AWS profile or GCP project.
type CloudModule struct{}

func (CloudModule) Name() string { return "cloud" }

func (CloudModule) Render(ctx Context) (string, bool) {
	// AWS: check AWS_PROFILE or AWS_DEFAULT_PROFILE env
	if profile := strings.TrimSpace(os.Getenv("AWS_PROFILE")); profile != "" && profile != "default" {
		return "aws:" + profile, true
	}
	if profile := strings.TrimSpace(os.Getenv("AWS_DEFAULT_PROFILE")); profile != "" && profile != "default" {
		return "aws:" + profile, true
	}

	// GCP: check GOOGLE_CLOUD_PROJECT, GCLOUD_PROJECT, or gcloud active config
	for _, k := range []string{"GOOGLE_CLOUD_PROJECT", "GCLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT"} {
		if proj := strings.TrimSpace(os.Getenv(k)); proj != "" {
			return "gcp:" + proj, true
		}
	}
	if proj := gcloudActiveProject(); proj != "" {
		return "gcp:" + proj, true
	}

	return "", false
}

func gcloudActiveProject() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Read active config name
	activeFile := filepath.Join(home, ".config", "gcloud", "active_config")
	nameB, err := os.ReadFile(activeFile)
	if err != nil {
		return ""
	}
	configName := strings.TrimSpace(string(nameB))
	if configName == "" {
		configName = "default"
	}

	// Read the named config properties file
	propFile := filepath.Join(home, ".config", "gcloud", "configurations", "config_"+configName)
	f, err := os.Open(propFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "project") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
