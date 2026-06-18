package memory

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

func TestLog_basic(t *testing.T) {
	defer withTempHome(t)()
	if err := Log("ls -la", "/tmp", "main", 0, 42); err != nil {
		t.Fatal(err)
	}
}

func TestLog_empty_cmd_noop(t *testing.T) {
	defer withTempHome(t)()
	if err := Log("", "/tmp", "", 0, 0); err != nil {
		t.Fatal(err)
	}
	cmds, _ := readAll()
	if len(cmds) != 0 {
		t.Fatalf("expected no entries for empty cmd, got %d", len(cmds))
	}
}

func TestSearch_basic(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("git status", "/repo", "main", 0, 100)
	_ = Log("docker ps", "/repo", "main", 0, 50)
	_ = Log("kubectl get pods", "/repo", "main", 0, 200)

	results, err := Search("git", SearchOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'git' query")
	}
	found := false
	for _, r := range results {
		if r.Cmd == "git status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'git status' not found in results")
	}
}

func TestSearch_failed_filter(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("good cmd", "/tmp", "", 0, 10)
	_ = Log("bad cmd", "/tmp", "", 1, 10)
	_ = Log("another bad", "/tmp", "", 127, 10)

	results, err := Search("", SearchOpts{Limit: 50, FailedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.ExitCode == 0 {
			t.Errorf("failed-only filter returned success: %q", r.Cmd)
		}
	}
}

func TestSearch_cwd_filter(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("cmd in alpha", "/alpha", "", 0, 10)
	_ = Log("cmd in beta", "/beta", "", 0, 10)

	results, err := Search("", SearchOpts{Limit: 50, CWDFilter: "/alpha"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.CWD != "/alpha" {
			t.Errorf("cwd filter: got %q, want /alpha", r.CWD)
		}
	}
}

func TestSearch_since_filter(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("recent cmd", "/tmp", "", 0, 10)

	cutoff := time.Now().Add(-1 * time.Hour)
	results, err := Search("", SearchOpts{Limit: 50, Since: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Cmd == "recent cmd" {
			found = true
		}
		if r.At.Before(cutoff) {
			t.Errorf("result before cutoff: %v", r.At)
		}
	}
	if !found {
		t.Error("recent cmd not found")
	}
}

func TestSearch_limit(t *testing.T) {
	defer withTempHome(t)()
	for i := 0; i < 20; i++ {
		_ = Log("repeated cmd", "/tmp", "", 0, int64(i))
	}
	results, err := Search("", SearchOpts{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 5 {
		t.Errorf("limit: got %d, want ≤5", len(results))
	}
}

func TestSearch_noMatch(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("docker ps", "/tmp", "", 0, 10)

	results, err := Search("kubernetes_xyz_nomatch", SearchOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for unmatchable query, got %d", len(results))
	}
}

func TestSearch_empty_query_returns_all(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("cmd one", "/tmp", "", 0, 10)
	_ = Log("cmd two", "/tmp", "", 0, 10)
	_ = Log("cmd three", "/tmp", "", 0, 10)

	results, err := Search("", SearchOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("empty query: want 3, got %d", len(results))
	}
}

func TestSearch_newest_first(t *testing.T) {
	defer withTempHome(t)()
	_ = Log("first", "/tmp", "", 0, 10)
	_ = Log("second", "/tmp", "", 0, 10)
	_ = Log("third", "/tmp", "", 0, 10)

	results, err := Search("", SearchOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 3 {
		t.Fatal("expected 3 results")
	}
	if results[0].Cmd != "third" {
		t.Errorf("newest first: got %q, want third", results[0].Cmd)
	}
}
