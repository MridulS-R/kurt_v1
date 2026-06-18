package cost

import (
	"os"
	"testing"
	"time"
)

func withTempHome(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return func() { os.Setenv("HOME", orig) }
}

func TestLog_and_Since(t *testing.T) {
	defer withTempHome(t)()

	Log("anthropic", "claude-haiku-4-5-20251001", 100, 50)
	Log("openai", "gpt-4o-mini", 200, 80)

	entries, err := Since(time.Now().Add(-1 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
}

func TestSince_empty(t *testing.T) {
	defer withTempHome(t)()
	entries, err := Since(time.Now().Add(-1 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("want 0, got %d", len(entries))
	}
}

func TestSince_filters_old(t *testing.T) {
	defer withTempHome(t)()
	Log("anthropic", "claude-haiku-4-5-20251001", 100, 50)
	// Ask for entries from the future (should return nothing)
	entries, err := Since(time.Now().Add(1 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("future cutoff: want 0, got %d", len(entries))
	}
}

func TestSummarize_tokens(t *testing.T) {
	entries := []UsageEntry{
		{InputTokens: 100, OutputTokens: 50},
		{InputTokens: 200, OutputTokens: 100},
	}
	s := Summarize(entries)
	if s.InputTokens != 300 {
		t.Errorf("input tokens: got %d, want 300", s.InputTokens)
	}
	if s.OutputTokens != 150 {
		t.Errorf("output tokens: got %d, want 150", s.OutputTokens)
	}
}

func TestSummarize_cost_nonzero(t *testing.T) {
	// Summarize aggregates pre-computed CostUSD from entries; populate it directly
	entries := []UsageEntry{
		{Provider: "anthropic", Model: "claude-haiku-4-5-20251001", InputTokens: 1000, OutputTokens: 500, CostUSD: 0.001},
	}
	s := Summarize(entries)
	if s.TotalCostUSD <= 0 {
		t.Error("expected positive TotalCostUSD when entry has CostUSD")
	}
}

func TestSummarize_empty(t *testing.T) {
	s := Summarize(nil)
	if s.InputTokens != 0 || s.OutputTokens != 0 || s.TotalCostUSD != 0 {
		t.Error("empty summarize should be zero")
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		usd  float64
		want string
	}{
		{0.0, "$0.000"},
		{0.001234, "$0.0012"},
		{1.5, "$1.500"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.usd)
		if got != tt.want {
			t.Errorf("FormatCost(%f): got %q, want %q", tt.usd, got, tt.want)
		}
	}
}

func TestByModel(t *testing.T) {
	entries := []UsageEntry{
		{Provider: "openai", Model: "gpt-4o", InputTokens: 100, OutputTokens: 50},
		{Provider: "openai", Model: "gpt-4o", InputTokens: 100, OutputTokens: 50},
		{Provider: "anthropic", Model: "claude", InputTokens: 200, OutputTokens: 100},
	}
	byModel := ByModel(entries)
	if len(byModel) != 2 {
		t.Fatalf("want 2 models, got %d", len(byModel))
	}
	// ByModel keys as "provider/model"
	gpt := byModel["openai/gpt-4o"]
	if gpt.InputTokens != 200 {
		t.Errorf("gpt-4o input: got %d, want 200", gpt.InputTokens)
	}
}

func TestByDay(t *testing.T) {
	defer withTempHome(t)()
	now := time.Now()
	entries := []UsageEntry{
		{At: now, Model: "gpt-4o", InputTokens: 100},
		{At: now, Model: "gpt-4o", InputTokens: 200},
	}
	byDay := ByDay(entries)
	if len(byDay) != 1 {
		t.Fatalf("want 1 day, got %d", len(byDay))
	}
	day := now.Format("2006-01-02")
	if s, ok := byDay[day]; !ok {
		t.Errorf("day %q not found", day)
	} else if s.InputTokens != 300 {
		t.Errorf("day total: got %d, want 300", s.InputTokens)
	}
}

func TestClearAll(t *testing.T) {
	defer withTempHome(t)()
	Log("anthropic", "claude-haiku-4-5-20251001", 100, 50)
	if err := ClearAll(); err != nil {
		t.Fatal(err)
	}
	entries, _ := Since(time.Now().Add(-1 * time.Hour))
	if len(entries) != 0 {
		t.Fatalf("expected empty after clear, got %d", len(entries))
	}
}
