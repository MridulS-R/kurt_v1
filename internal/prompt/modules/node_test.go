package modules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNodeModule_Name(t *testing.T) {
	m := NodeModule{}
	if m.Name() != "node" {
		t.Errorf("Name()=%q want %q", m.Name(), "node")
	}
}

func TestNodeModule_NvmrcDetection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("18.20.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	seg, ok := NodeModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "node:v18.20.0" {
		t.Errorf("got %q want %q", seg, "node:v18.20.0")
	}
}

func TestNodeModule_NodeVersionFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".node-version"), []byte("20.11.0"), 0644); err != nil {
		t.Fatal(err)
	}
	seg, ok := NodeModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "node:v20.11.0" {
		t.Errorf("got %q want %q", seg, "node:v20.11.0")
	}
}

func TestNodeModule_NoMarker(t *testing.T) {
	dir := t.TempDir()
	_, ok := NodeModule{}.Render(Context{CWD: dir})
	if ok {
		t.Error("expected ok=false when no node markers present")
	}
}

func TestNodeModule_PrefixV(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v16.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	seg, ok := NodeModule{}.Render(Context{CWD: dir})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "node:v16.0.0" {
		t.Errorf("got %q want %q", seg, "node:v16.0.0")
	}
}
