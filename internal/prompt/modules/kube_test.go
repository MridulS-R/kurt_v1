package modules

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleKubeConfig = `apiVersion: v1
kind: Config
current-context: staging-cluster
contexts:
- name: staging-cluster
  context:
    cluster: staging
    user: admin
    namespace: staging-ns
- name: prod-cluster
  context:
    cluster: prod
    user: prod-admin
    namespace: production
clusters:
- name: staging
  cluster:
    server: https://staging.example.com
users:
- name: admin
  user:
    token: fake-token
`

const emptyContextKubeConfig = `apiVersion: v1
kind: Config
current-context: ""
contexts: []
`

func writeTempKubeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return p
}

func TestKubeModule_BasicContext(t *testing.T) {
	p := writeTempKubeConfig(t, sampleKubeConfig)
	t.Setenv("KUBECONFIG", p)

	m := KubeModule{}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "⎈ staging-cluster" {
		t.Errorf("got %q want %q", seg, "⎈ staging-cluster")
	}
}

func TestKubeModule_ShowNamespace(t *testing.T) {
	p := writeTempKubeConfig(t, sampleKubeConfig)
	t.Setenv("KUBECONFIG", p)

	m := KubeModule{ShowNamespace: true}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "⎈ staging-cluster/staging-ns" {
		t.Errorf("got %q want %q", seg, "⎈ staging-cluster/staging-ns")
	}
}

func TestKubeModule_EmptyContext(t *testing.T) {
	p := writeTempKubeConfig(t, emptyContextKubeConfig)
	t.Setenv("KUBECONFIG", p)

	m := KubeModule{}
	_, ok := m.Render(Context{})
	if ok {
		t.Error("expected ok=false when current-context is empty")
	}
}

func TestKubeModule_MissingFile(t *testing.T) {
	t.Setenv("KUBECONFIG", "/nonexistent/path/config")

	m := KubeModule{}
	_, ok := m.Render(Context{})
	if ok {
		t.Error("expected ok=false when kubeconfig is missing")
	}
}

func TestKubeModule_Name(t *testing.T) {
	m := KubeModule{}
	if m.Name() != "kube" {
		t.Errorf("Name()=%q want %q", m.Name(), "kube")
	}
}
