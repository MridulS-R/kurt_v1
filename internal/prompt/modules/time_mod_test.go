package modules

import (
	"strings"
	"testing"
	"time"
)

func TestTimeModule_Name(t *testing.T) {
	m := TimeModule{}
	if m.Name() != "time" {
		t.Errorf("Name()=%q want %q", m.Name(), "time")
	}
}

func TestTimeModule_DefaultFormat(t *testing.T) {
	m := TimeModule{}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Default format is HH:MM — must contain a colon and be 5 chars
	if len(seg) != 5 || !strings.Contains(seg, ":") {
		t.Errorf("unexpected time segment %q (want HH:MM format)", seg)
	}
}

func TestTimeModule_CustomFormat(t *testing.T) {
	m := TimeModule{Format: "2006-01-02"}
	seg, ok := m.Render(Context{})
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Should parse as a valid date in YYYY-MM-DD
	if _, err := time.Parse("2006-01-02", seg); err != nil {
		t.Errorf("segment %q does not match format 2006-01-02: %v", seg, err)
	}
}
