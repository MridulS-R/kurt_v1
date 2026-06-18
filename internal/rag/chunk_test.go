package rag

import (
	"strings"
	"testing"
)

func TestChunk_empty(t *testing.T) {
	chunks := Chunk("", 100, -1)
	if len(chunks) != 0 {
		t.Fatalf("empty input: want 0 chunks, got %d", len(chunks))
	}
}

func TestChunk_shortText(t *testing.T) {
	text := "Hello world"
	chunks := Chunk(text, 100, -1)
	if len(chunks) != 1 {
		t.Fatalf("short text: want 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("got %q, want %q", chunks[0], text)
	}
}

func TestChunk_paragraphSplit(t *testing.T) {
	text := "Paragraph one.\n\nParagraph two.\n\nParagraph three."
	chunks := Chunk(text, 20, -1)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks from paragraphs, got %d", len(chunks))
	}
}

func TestChunk_noEmptyChunks(t *testing.T) {
	text := strings.Repeat("word ", 200)
	chunks := Chunk(text, 50, -1)
	for i, c := range chunks {
		if strings.TrimSpace(c) == "" {
			t.Fatalf("chunk %d is empty", i)
		}
	}
}

func TestChunk_respects_size(t *testing.T) {
	// Each "word" is 5 chars; 200 words = 1000 chars. Chunk size 50 ≈ 10 words.
	text := strings.Repeat("hello ", 200)
	chunks := Chunk(text, 10, -1)
	if len(chunks) < 5 {
		t.Fatalf("expected many chunks for size=10, got %d", len(chunks))
	}
}

func TestChunk_whitespaceOnly(t *testing.T) {
	chunks := Chunk("   \n\n   \n   ", 100, -1)
	if len(chunks) != 0 {
		t.Fatalf("whitespace-only: want 0 chunks, got %d", len(chunks))
	}
}

func TestChunk_singleLongParagraph(t *testing.T) {
	// No paragraph breaks — should still chunk
	text := strings.Repeat("a", 5000)
	chunks := Chunk(text, 100, -1)
	if len(chunks) < 2 {
		t.Fatalf("expected hard-split chunks for long paragraph, got %d", len(chunks))
	}
}

func TestChunk_allChunksNonEmpty(t *testing.T) {
	text := "Line one.\nLine two.\nLine three.\n\nNew paragraph here.\n\nAnother one."
	chunks := Chunk(text, 5, -1)
	for i, c := range chunks {
		if strings.TrimSpace(c) == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}
}
