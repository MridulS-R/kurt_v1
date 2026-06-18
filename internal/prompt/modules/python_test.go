package modules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonModule_Name(t *testing.T) {
	m := PythonModule{}
	if m.Name() != "python" {
		t.Errorf("Name()=%q want %q", m.Name(), "python")
	}
}

func TestPythonModule_PythonVersionFile(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("PYENV_VERSION")
	if err := os.WriteFile(filepath.Join(dir, ".python-version"), []byte("3.11.4\n"), 0644); err != nil {
		t.Fatal(err)
	}
	seg, ok := PythonModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "py:3.11.4" {
		t.Errorf("got %q want %q", seg, "py:3.11.4")
	}
}

func TestPythonModule_EnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PYENV_VERSION", "3.10.0")
	seg, ok := PythonModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "py:3.10.0" {
		t.Errorf("got %q want %q", seg, "py:3.10.0")
	}
}

func TestPythonModule_NoMarker(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("PYENV_VERSION")
	_, ok := PythonModule{}.Render(Context{CWD: dir})
	if ok {
		t.Error("expected ok=false when no python markers present")
	}
}

func TestPythonModule_PyvenvCfg(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("PYENV_VERSION")
	cfg := "version = 3.12.0\nhome = /usr/bin\n"
	if err := os.WriteFile(filepath.Join(dir, "pyvenv.cfg"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	seg, ok := PythonModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true with pyvenv.cfg")
	}
	if seg != "py:3.12.0" {
		t.Errorf("got %q want %q", seg, "py:3.12.0")
	}
}
