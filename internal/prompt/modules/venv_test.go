package modules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVenvModule_noEnv(t *testing.T) {
	os.Unsetenv("VIRTUAL_ENV")
	_, ok := VenvModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment without VIRTUAL_ENV")
	}
}

func TestVenvModule_withEnv(t *testing.T) {
	os.Setenv("VIRTUAL_ENV", "/home/user/myproject/.venv")
	defer os.Unsetenv("VIRTUAL_ENV")

	seg, ok := VenvModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with VIRTUAL_ENV set")
	}
	if seg == "" {
		t.Error("segment should not be empty")
	}
}

func TestVenvModule_genericName(t *testing.T) {
	dir := t.TempDir()
	venvPath := filepath.Join(dir, "myproject", ".venv")
	os.MkdirAll(venvPath, 0755)
	os.Setenv("VIRTUAL_ENV", venvPath)
	defer os.Unsetenv("VIRTUAL_ENV")

	seg, ok := VenvModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment")
	}
	// Should use parent dir name "myproject" instead of ".venv"
	if seg == "(.venv)" {
		t.Errorf("should use parent name, got %q", seg)
	}
}

func TestCondaModule_noEnv(t *testing.T) {
	os.Unsetenv("CONDA_DEFAULT_ENV")
	_, ok := CondaModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment without CONDA_DEFAULT_ENV")
	}
}

func TestCondaModule_base(t *testing.T) {
	os.Setenv("CONDA_DEFAULT_ENV", "base")
	defer os.Unsetenv("CONDA_DEFAULT_ENV")

	_, ok := CondaModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment for 'base' conda env")
	}
}

func TestCondaModule_named(t *testing.T) {
	os.Setenv("CONDA_DEFAULT_ENV", "myenv")
	defer os.Unsetenv("CONDA_DEFAULT_ENV")

	seg, ok := CondaModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment for named conda env")
	}
	if seg == "" {
		t.Error("segment should not be empty")
	}
}

func TestNodeModule_noFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	_, ok := NodeModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment without .nvmrc")
	}
}

func TestNodeModule_nvmrc(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("18.0.0\n"), 0644)
	seg, ok := NodeModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with .nvmrc")
	}
	if seg != "node:v18.0.0" {
		t.Errorf("got %q, want node:v18.0.0", seg)
	}
}

func TestNodeModule_nvmrcWithV(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v20.1.0"), 0644)
	seg, ok := NodeModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment")
	}
	if seg != "node:v20.1.0" {
		t.Errorf("got %q", seg)
	}
}

func TestTimeModule_name(t *testing.T) {
	m := TimeModule{}
	if m.Name() != "time" {
		t.Error("wrong name")
	}
}

func TestTimeModule_renders(t *testing.T) {
	seg, ok := TimeModule{}.Render(Context{TimeFormat: "15:04"})
	if !ok {
		t.Fatal("expected time segment")
	}
	if len(seg) != 5 {
		t.Errorf("expected HH:MM format (5 chars), got %q", seg)
	}
}

func TestTimeModule_defaultFormat(t *testing.T) {
	seg, ok := TimeModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected time segment with default format")
	}
	if len(seg) != 5 {
		t.Errorf("expected HH:MM, got %q", seg)
	}
}
