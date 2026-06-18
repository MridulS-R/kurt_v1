package cache

import (
	"os"
	"testing"
)

func withTempHome(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return func() { os.Setenv("HOME", orig) }
}

func TestKey_deterministic(t *testing.T) {
	k1 := Key("anthropic", "claude", "hello")
	k2 := Key("anthropic", "claude", "hello")
	if k1 != k2 {
		t.Fatal("Key not deterministic")
	}
}

func TestKey_different(t *testing.T) {
	k1 := Key("openai", "gpt-4o", "hello")
	k2 := Key("openai", "gpt-4o", "world")
	if k1 == k2 {
		t.Fatal("different inputs produced same key")
	}
}

func TestKey_providerModel(t *testing.T) {
	k1 := Key("openai", "gpt-4", "same")
	k2 := Key("anthropic", "claude", "same")
	if k1 == k2 {
		t.Fatal("different providers produced same key")
	}
}

func TestKey_length(t *testing.T) {
	k := Key("p", "m", "i")
	if len(k) != 32 {
		t.Errorf("key length: want 32 hex chars, got %d", len(k))
	}
}

func TestPutAndGet(t *testing.T) {
	defer withTempHome(t)()

	if err := Put("p", "m", "input", "response", 24); err != nil {
		t.Fatal(err)
	}
	key := Key("p", "m", "input")
	resp, ok := Get(key, 24)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if resp != "response" {
		t.Errorf("got %q, want %q", resp, "response")
	}
}

func TestGet_miss(t *testing.T) {
	defer withTempHome(t)()
	_, ok := Get("nosuchkey", 24)
	if ok {
		t.Fatal("expected cache miss for unknown key")
	}
}

func TestGet_noFile(t *testing.T) {
	defer withTempHome(t)()
	// No Put() call — file doesn't exist
	_, ok := Get(Key("x", "y", "z"), 24)
	if ok {
		t.Fatal("expected miss when cache file absent")
	}
}

func TestClearAll(t *testing.T) {
	defer withTempHome(t)()
	_ = Put("p", "m", "i", "r", 24)
	if err := ClearAll(); err != nil {
		t.Fatal(err)
	}
	_, ok := Get(Key("p", "m", "i"), 24)
	if ok {
		t.Fatal("expected miss after clear")
	}
}

func TestClearAll_idempotent(t *testing.T) {
	defer withTempHome(t)()
	if err := ClearAll(); err != nil {
		t.Fatal("first clear:", err)
	}
	if err := ClearAll(); err != nil {
		t.Fatal("second clear:", err)
	}
}

func TestList_empty(t *testing.T) {
	defer withTempHome(t)()
	entries, err := List(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d", len(entries))
	}
}

func TestList_returnsNewestFirst(t *testing.T) {
	defer withTempHome(t)()
	_ = Put("p", "m", "first", "r1", 24)
	_ = Put("p", "m", "second", "r2", 24)
	entries, err := List(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Input != "second" {
		t.Errorf("newest first: got input %q, want %q", entries[0].Input, "second")
	}
}

func TestList_limitN(t *testing.T) {
	defer withTempHome(t)()
	for i := 0; i < 10; i++ {
		_ = Put("p", "m", string(rune('a'+i)), "r", 24)
	}
	entries, err := List(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3, got %d", len(entries))
	}
}

func TestGet_noTTL(t *testing.T) {
	defer withTempHome(t)()
	_ = Put("p", "m", "nottle", "r", 0)
	_, ok := Get(Key("p", "m", "nottle"), 0)
	if !ok {
		t.Fatal("expected hit for TTL=0 (never expire)")
	}
}

func TestPut_multipleEntries(t *testing.T) {
	defer withTempHome(t)()
	_ = Put("p", "m", "a", "ra", 24)
	_ = Put("p", "m", "b", "rb", 24)
	_ = Put("p", "m", "c", "rc", 24)

	ra, ok := Get(Key("p", "m", "a"), 24)
	if !ok || ra != "ra" {
		t.Errorf("entry a: got %q ok=%v", ra, ok)
	}
	rc, ok := Get(Key("p", "m", "c"), 24)
	if !ok || rc != "rc" {
		t.Errorf("entry c: got %q ok=%v", rc, ok)
	}
}

func TestPut_unicode(t *testing.T) {
	defer withTempHome(t)()
	_ = Put("p", "m", "你好世界", "こんにちは", 24)
	resp, ok := Get(Key("p", "m", "你好世界"), 24)
	if !ok {
		t.Fatal("expected hit for unicode input")
	}
	if resp != "こんにちは" {
		t.Errorf("got %q", resp)
	}
}
