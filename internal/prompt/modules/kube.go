package modules

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KubeModule shows the current kubectl context from ~/.kube/config.
type KubeModule struct{}

func (KubeModule) Name() string { return "kube" }

func (KubeModule) Render(ctx Context) (string, bool) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	b, err := os.ReadFile(kubeconfig)
	if err != nil {
		return "", false
	}

	var kc struct {
		CurrentContext string `yaml:"current-context"`
	}
	if err := yaml.Unmarshal(b, &kc); err != nil || kc.CurrentContext == "" {
		return "", false
	}

	return "⎈ " + kc.CurrentContext, true
}
