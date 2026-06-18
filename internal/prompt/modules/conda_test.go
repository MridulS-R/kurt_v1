package modules

import (
	"testing"
)

func TestCondaModule_Name(t *testing.T) {
	m := CondaModule{}
	if m.Name() != "conda" {
		t.Errorf("Name()=%q want %q", m.Name(), "conda")
	}
}

func TestCondaModule_Active(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "myenv")
	m := CondaModule{}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if seg != "conda:myenv" {
		t.Errorf("got %q want %q", seg, "conda:myenv")
	}
}

func TestCondaModule_BaseHidden(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "base")
	m := CondaModule{ShowBase: false}
	_, ok := m.Render(Context{})
	if ok {
		t.Error("expected base env to be suppressed when ShowBase=false")
	}
}

func TestCondaModule_BaseShown(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "base")
	m := CondaModule{ShowBase: true}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true when ShowBase=true")
	}
	if seg != "conda:base" {
		t.Errorf("got %q want %q", seg, "conda:base")
	}
}

func TestCondaModule_Inactive(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "")
	m := CondaModule{}
	_, ok := m.Render(Context{})
	if ok {
		t.Error("expected ok=false when no conda env active")
	}
}
