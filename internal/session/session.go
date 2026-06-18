package session

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Meta struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	CWD       string    `json:"cwd"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	Role      string    `json:"role"` // system | user | assistant
	Content   string    `json:"content"`
	Timestamp time.Time `json:"ts"`
}

func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".local", "share", "kurt", "sessions")
	return d, os.MkdirAll(d, 0700)
}

func sessionDir(id string) (string, error) {
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, id), nil
}

// New creates a new session and persists its metadata.
func New(name, provider, model, cwd string) (*Meta, error) {
	id := generateID()
	if name == "" {
		name = id
	}
	dir, err := sessionDir(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	m := &Meta{
		ID:        id,
		Name:      name,
		Provider:  provider,
		Model:     model,
		CWD:       cwd,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return m, saveMeta(m)
}

func saveMeta(m *Meta) error {
	dir, err := sessionDir(m.ID)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(dir, "meta.json"), b, 0600)
}

func Touch(id string) error {
	m, err := Load(id)
	if err != nil {
		return err
	}
	m.UpdatedAt = time.Now()
	return saveMeta(m)
}

// Append adds a message to the session's history file.
func Append(id string, msg Message) error {
	dir, err := sessionDir(id)
	if err != nil {
		return err
	}
	msg.Timestamp = time.Now()
	b, _ := json.Marshal(msg)
	f, err := os.OpenFile(filepath.Join(dir, "history.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// History loads all messages for a session.
func History(id string) ([]Message, error) {
	dir, err := sessionDir(id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(dir, "history.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var msgs []Message
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m Message
		if json.Unmarshal([]byte(line), &m) == nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, sc.Err()
}

// ClearHistory wipes all messages but keeps the session alive.
func ClearHistory(id string) error {
	dir, err := sessionDir(id)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "history.jsonl"), nil, 0600)
}

// Load finds a session by exact ID, name, or ID prefix.
func Load(idOrName string) (*Meta, error) {
	all, err := List()
	if err != nil {
		return nil, err
	}
	for i := range all {
		m := &all[i]
		if m.ID == idOrName || m.Name == idOrName || strings.HasPrefix(m.ID, idOrName) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("session %q not found", idOrName)
}

// List returns all sessions sorted newest-first.
func List() ([]Meta, error) {
	d, err := DataDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var all []Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(d, e.Name(), "meta.json"))
		if err != nil {
			continue
		}
		var m Meta
		if json.Unmarshal(b, &m) == nil {
			all = append(all, m)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	return all, nil
}

// Destroy permanently deletes a session and all its data.
func Destroy(id string) error {
	dir, err := sessionDir(id)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// DestroyAll permanently deletes every session.
func DestroyAll() error {
	d, err := DataDir()
	if err != nil {
		return err
	}
	all, err := List()
	if err != nil {
		return err
	}
	for _, m := range all {
		os.RemoveAll(filepath.Join(d, m.ID))
	}
	return nil
}

// MsgCount returns the number of messages in a session.
func MsgCount(id string) int {
	msgs, _ := History(id)
	return len(msgs)
}

// ── ID generation ────────────────────────────────────────────────────────────

var adjectives = []string{
	"swift", "dark", "bright", "cold", "warm", "deep", "wild", "calm",
	"sharp", "bold", "clear", "fresh", "quiet", "still", "lost", "bare",
	"raw", "thin", "old", "new", "open", "wide", "fast", "safe", "rare",
	"tiny", "vast", "soft", "dry", "dim",
}
var nouns = []string{
	"river", "moon", "storm", "spark", "field", "rock", "wave", "cloud",
	"flame", "shade", "ridge", "creek", "path", "peak", "grove", "frost",
	"smoke", "stone", "echo", "drift", "bloom", "arc", "tide", "vault",
	"comet", "forge", "lens", "node", "orbit", "pulse",
}

func generateID() string {
	b := make([]byte, 3)
	rand.Read(b)
	adj := adjectives[int(b[0])%len(adjectives)]
	noun := nouns[int(b[1])%len(nouns)]
	suffix := fmt.Sprintf("%02x", b[2])
	return adj + "-" + noun + "-" + suffix
}
