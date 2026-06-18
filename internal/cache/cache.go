package cache

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kurt_v1/internal/history"
)

// Entry is a cached LLM response.
type Entry struct {
	Key      string    `json:"key"`
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Input    string    `json:"input"`    // full prompt text
	Response string    `json:"response"`
	At       time.Time `json:"at"`
	TTLHours int       `json:"ttl_hours"` // 0 = never expire
}

func filePath() (string, error) {
	d, err := history.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "response_cache.jsonl"), nil
}

// Key computes the cache key for a provider+model+input combination.
func Key(provider, model, input string) string {
	h := sha256.Sum256([]byte(provider + "|" + model + "|" + input))
	return fmt.Sprintf("%x", h[:16])
}

// Get looks up a cached response. Returns "", false if not found or expired.
func Get(key string, ttlHours int) (string, bool) {
	path, err := filePath()
	if err != nil {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if json.Unmarshal([]byte(line), &e) != nil || e.Key != key {
			continue
		}
		// Check TTL
		ttl := e.TTLHours
		if ttlHours > 0 {
			ttl = ttlHours
		}
		if ttl > 0 && time.Since(e.At).Hours() > float64(ttl) {
			return "", false
		}
		return e.Response, true
	}
	return "", false
}

// Put stores a response in the cache.
func Put(provider, model, input, response string, ttlHours int) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	e := Entry{
		Key: Key(provider, model, input), Provider: provider, Model: model,
		Input: input, Response: response, At: time.Now(), TTLHours: ttlHours,
	}
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}

// List returns up to n most-recent cache entries (newest first).
func List(n int) ([]Entry, error) {
	path, err := filePath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if json.Unmarshal([]byte(line), &e) == nil {
			all = append(all, e)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// Return newest first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if n > 0 && len(all) > n {
		all = all[:n]
	}
	return all, nil
}

// ClearAll removes the cache file.
func ClearAll() error {
	path, err := filePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
