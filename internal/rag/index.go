package rag

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kurt_v1/internal/embed"
	"kurt_v1/internal/history"
)

// ChunkEntry is one stored document chunk with its embedding.
type ChunkEntry struct {
	ID     string    `json:"id"`
	Source string    `json:"source"` // original file path
	Index  int       `json:"index"`  // chunk number within source
	Text   string    `json:"text"`
	Vec    []float32 `json:"vec"`
}

// Meta holds collection metadata.
type Meta struct {
	Collection  string    `json:"collection"`
	EmbedModel  string    `json:"embed_model"`
	ChunkCount  int       `json:"chunk_count"`
	IndexedAt   time.Time `json:"indexed_at"`
}

// ── storage paths ─────────────────────────────────────────────────────────────

func collectionDir(name string) (string, error) {
	d, err := history.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(d, "rag", name)
	return dir, os.MkdirAll(dir, 0700)
}

func chunksPath(collection string) (string, error) {
	d, err := collectionDir(collection)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "chunks.jsonl"), nil
}

func metaPath(collection string) (string, error) {
	d, err := collectionDir(collection)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "meta.json"), nil
}

// ── index ─────────────────────────────────────────────────────────────────────

// IndexFiles chunks and embeds all supported files at the given paths.
// onProgress is called after each file with (filename, chunkCount).
func IndexFiles(collection string, paths []string, embedder embed.Embedder, chunkSize int, onProgress func(file string, chunks int)) error {
	// Collect all file paths.
	var files []string
	for _, p := range paths {
		found, err := walkFiles(p)
		if err != nil {
			return err
		}
		files = append(files, found...)
	}

	cpath, err := chunksPath(collection)
	if err != nil {
		return err
	}
	f, err := os.Create(cpath) // overwrite existing index
	if err != nil {
		return err
	}
	defer f.Close()

	total := 0
	embedModel := ""
	for _, file := range files {
		text, err := readFileText(file)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}

		chunks := Chunk(text, chunkSize, -1)
		for i, chunk := range chunks {
			vec, err := embedder.Embed(chunk)
			if err != nil {
				return fmt.Errorf("embedding %s chunk %d: %w", file, i, err)
			}
			id := chunkID(file, i)
			entry := ChunkEntry{
				ID: id, Source: file, Index: i, Text: chunk, Vec: vec,
			}
			b, _ := json.Marshal(entry)
			_, _ = f.Write(append(b, '\n'))
			total++
		}
		if onProgress != nil {
			onProgress(file, len(chunks))
		}
		// Detect embed model from env (best-effort)
		if embedModel == "" {
			embedModel = guessEmbedModel(embedder)
		}
	}

	// Write meta.
	meta := Meta{
		Collection: collection, EmbedModel: embedModel,
		ChunkCount: total, IndexedAt: time.Now(),
	}
	mpath, _ := metaPath(collection)
	mb, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(mpath, mb, 0600)

	return nil
}

// Search returns the top-k chunks most similar to query.
func Search(collection string, queryVec []float32, topK int) ([]ChunkEntry, error) {
	cpath, err := chunksPath(collection)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(cpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("collection %q not found — run: kurt rag index", collection)
		}
		return nil, err
	}
	defer f.Close()

	type scored struct {
		entry ChunkEntry
		score float32
	}
	var best []scored

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry ChunkEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		score := embed.Cosine(queryVec, entry.Vec)
		best = append(best, scored{entry, score})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(best, func(i, j int) bool {
		return best[i].score > best[j].score
	})

	if topK > len(best) {
		topK = len(best)
	}
	out := make([]ChunkEntry, topK)
	for i := range out {
		out[i] = best[i].entry
	}
	return out, nil
}

// ListCollections returns the names of all indexed collections.
func ListCollections() ([]Meta, error) {
	d, err := history.DataDir()
	if err != nil {
		return nil, err
	}
	ragDir := filepath.Join(d, "rag")
	entries, err := os.ReadDir(ragDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mpath := filepath.Join(ragDir, e.Name(), "meta.json")
		b, err := os.ReadFile(mpath)
		if err != nil {
			continue
		}
		var m Meta
		if json.Unmarshal(b, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

// RemoveCollection deletes a collection's index.
func RemoveCollection(name string) error {
	d, err := history.DataDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(d, "rag", name))
}

// ── file helpers ──────────────────────────────────────────────────────────────

var supportedExt = map[string]bool{
	".txt": true, ".md": true, ".markdown": true,
	".go": true, ".py": true, ".js": true, ".ts": true,
	".jsx": true, ".tsx": true, ".rs": true, ".java": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".yaml": true, ".yml": true, ".toml": true, ".json": true,
	".sh": true, ".bash": true, ".zsh": true,
	".csv": true, ".html": true, ".css": true, ".sql": true,
}

func walkFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(root))
		if supportedExt[ext] || ext == "" {
			return []string{root}, nil
		}
		return nil, nil
	}
	var out []string
	_ = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		// Skip hidden files and common noise dirs.
		name := fi.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		dir := filepath.Dir(path)
		if strings.Contains(dir, string(os.PathSeparator)+".") ||
			strings.Contains(dir, string(os.PathSeparator)+"node_modules") ||
			strings.Contains(dir, string(os.PathSeparator)+"__pycache__") ||
			strings.Contains(dir, string(os.PathSeparator)+"vendor") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if supportedExt[ext] {
			out = append(out, path)
		}
		return nil
	})
	return out, nil
}

func readFileText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 512*1024)) // 512KB cap per file
	return string(b), err
}

func chunkID(source string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", source, index)))
	return fmt.Sprintf("%x", h[:8])
}

func guessEmbedModel(e embed.Embedder) string {
	switch v := e.(type) {
	case *embed.OllamaEmbedder:
		if v.Model != "" {
			return v.Model
		}
		return "nomic-embed-text"
	case *embed.OpenAIEmbedder:
		if v.Model != "" {
			return v.Model
		}
		return "text-embedding-3-small"
	}
	return "unknown"
}
