package modules

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KubeModule shows the current kubectl context from ~/.kube/config.
// ShowNamespace: if true, appends "/namespace" to the context name.
type KubeModule struct {
	ShowNamespace bool
}

func (KubeModule) Name() string { return "kube" }

func (m KubeModule) Render(ctx Context) (string, bool) {
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
		Contexts       []struct {
			Name    string `yaml:"name"`
			Context struct {
				Namespace string `yaml:"namespace"`
			} `yaml:"context"`
		} `yaml:"contexts"`
	}
	if err := yaml.Unmarshal(b, &kc); err != nil || kc.CurrentContext == "" {
		return "", false
	}

	if m.ShowNamespace {
		for _, c := range kc.Contexts {
			if c.Name == kc.CurrentContext && c.Context.Namespace != "" {
				return fmt.Sprintf("⎈ %s/%s", kc.CurrentContext, c.Context.Namespace), true
			}
		}
	}

	return "⎈ " + kc.CurrentContext, true
}
