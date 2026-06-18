package prompts

import (
	"os"
	"testing"
)

func withTempDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return func() { os.Setenv("HOME", origHome) }
}

func TestAddAndGet(t *testing.T) {
	defer withTempDir(t)()

	if err := Add("greet", "Hello {{.input}}!", "greeting"); err != nil {
		t.Fatal(err)
	}
	p, err := Get("greet")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "greet" {
		t.Errorf("name: got %q, want %q", p.Name, "greet")
	}
	if p.Template != "Hello {{.input}}!" {
		t.Errorf("template: got %q", p.Template)
	}
	if p.Description != "greeting" {
		t.Errorf("description: got %q", p.Description)
	}
}

func TestGet_notFound(t *testing.T) {
	defer withTempDir(t)()
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing prompt")
	}
}

func TestList_empty(t *testing.T) {
	defer withTempDir(t)()
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestList_multiple(t *testing.T) {
	defer withTempDir(t)()
	_ = Add("a", "tmpl a", "")
	_ = Add("b", "tmpl b", "")
	_ = Add("c", "tmpl c", "")
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("want 3, got %d", len(list))
	}
}

func TestRemove(t *testing.T) {
	defer withTempDir(t)()
	_ = Add("temp", "tmpl", "")
	if err := Remove("temp"); err != nil {
		t.Fatal(err)
	}
	_, err := Get("temp")
	if err == nil {
		t.Fatal("expected error after remove")
	}
}

func TestRemove_notFound(t *testing.T) {
	defer withTempDir(t)()
	err := Remove("ghost")
	if err == nil {
		t.Fatal("expected error removing nonexistent prompt")
	}
}

func TestAdd_overwrite(t *testing.T) {
	defer withTempDir(t)()
	_ = Add("x", "v1", "")
	_ = Add("x", "v2", "updated")
	p, _ := Get("x")
	if p.Template != "v2" {
		t.Errorf("overwrite: got %q, want v2", p.Template)
	}
}

func TestRender_input(t *testing.T) {
	result, err := Render("Hello {{.input}}!", "world", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello world!" {
		t.Errorf("got %q", result)
	}
}

func TestRender_vars(t *testing.T) {
	result, err := Render("{{.name}} is {{.lang}}", "", map[string]string{
		"name": "Go",
		"lang": "fast",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Go is fast" {
		t.Errorf("got %q", result)
	}
}

func TestRender_noVars(t *testing.T) {
	result, err := Render("static text", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "static text" {
		t.Errorf("got %q", result)
	}
}

func TestRender_invalidTemplate(t *testing.T) {
	_, err := Render("{{.unclosed", "", nil)
	if err == nil {
		t.Fatal("expected parse error for invalid template")
	}
}

func TestRender_missingVar(t *testing.T) {
	// Go templates render missing vars as "<no value>" by default
	result, err := Render("{{.missing}}", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Should not error — Go templates are lenient
	_ = result
}
