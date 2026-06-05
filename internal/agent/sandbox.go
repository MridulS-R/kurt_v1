package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Sandbox abstracts where agent commands execute.
type Sandbox interface {
	Run(command string) (string, error)
	WriteFile(name, content string) error
	ReadFile(name string) (string, error)
	Dir() string
	Kind() string
	Cleanup() error
}

// ── tmpdir sandbox ────────────────────────────────────────────────────────────

type TmpdirSandbox struct{ dir string }

func NewTmpdirSandbox() (*TmpdirSandbox, error) {
	dir, err := os.MkdirTemp("", "kurt-agent-*")
	if err != nil {
		return nil, err
	}
	return &TmpdirSandbox{dir: dir}, nil
}

func (s *TmpdirSandbox) Kind() string { return "tmpdir" }
func (s *TmpdirSandbox) Dir() string  { return s.dir }

func (s *TmpdirSandbox) Run(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = s.dir
	cmd.Env = sandboxEnv(s.dir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s *TmpdirSandbox) WriteFile(name, content string) error {
	return sandboxWrite(s.dir, name, content)
}

func (s *TmpdirSandbox) ReadFile(name string) (string, error) {
	return sandboxRead(s.dir, name)
}

func (s *TmpdirSandbox) Cleanup() error { return os.RemoveAll(s.dir) }

// ── docker sandbox ────────────────────────────────────────────────────────────

type DockerSandbox struct {
	image       string
	hostDir     string // tmpdir mounted as /workspace in container
	containerID string
}

func NewDockerSandbox(image string) (*DockerSandbox, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH — install Docker Desktop or use --sandbox tmpdir")
	}
	if image == "" {
		image = "alpine:latest"
	}
	dir, err := os.MkdirTemp("", "kurt-agent-*")
	if err != nil {
		return nil, err
	}
	out, err := exec.Command("docker", "run", "-d", "--rm",
		"--network=none",
		"-v", dir+":/workspace",
		"-w", "/workspace",
		image,
		"sh", "-c", "while true; do sleep 30; done",
	).Output()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("docker run failed: %w — is Docker running?", err)
	}
	id := strings.TrimSpace(string(out))
	return &DockerSandbox{image: image, hostDir: dir, containerID: id}, nil
}

func (s *DockerSandbox) Kind() string { return "docker(" + s.image + ")" }
func (s *DockerSandbox) Dir() string  { return s.hostDir }

func (s *DockerSandbox) Run(command string) (string, error) {
	out, err := exec.Command("docker", "exec", s.containerID,
		"sh", "-c", command).CombinedOutput()
	return string(out), err
}

func (s *DockerSandbox) WriteFile(name, content string) error {
	return sandboxWrite(s.hostDir, name, content)
}

func (s *DockerSandbox) ReadFile(name string) (string, error) {
	return sandboxRead(s.hostDir, name)
}

func (s *DockerSandbox) Cleanup() error {
	_ = exec.Command("docker", "kill", s.containerID).Run()
	return os.RemoveAll(s.hostDir)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sandboxEnv(dir string) []string {
	// Minimal env: no AWS keys, no tokens, no cloud credentials.
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin",
		"TERM=xterm-256color",
		"TMPDIR=" + os.TempDir(),
		"PWD=" + dir,
	}
	// Carry through language runtimes that are safe.
	for _, k := range []string{"GOPATH", "GOROOT", "NVM_DIR", "PYENV_ROOT"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func sandboxWrite(base, name, content string) error {
	p, err := safePath(base, name)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0700)
	return os.WriteFile(p, []byte(content), 0600)
}

func sandboxRead(base, name string) (string, error) {
	p, err := safePath(base, name)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

func safePath(base, name string) (string, error) {
	p := filepath.Join(base, filepath.Clean("/"+name))
	if !strings.HasPrefix(p, base) {
		return "", fmt.Errorf("path %q escapes sandbox", name)
	}
	return p, nil
}
