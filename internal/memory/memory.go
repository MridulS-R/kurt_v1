package memory

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kurt_v1/internal/history"
)

// Command is a single shell command entry in the memory log.
type Command struct {
	At         time.Time `json:"at"`
	Cmd        string    `json:"cmd"`
	CWD        string    `json:"cwd"`
	GitBranch  string    `json:"git_branch,omitempty"`
	ExitCode   int       `json:"exit_code"`
	DurationMs int64     `json:"duration_ms"`
}

const maxCommands = 10000

var mu sync.Mutex

func filePath() (string, error) {
	d, err := history.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "memory.jsonl"), nil
}

// Log appends a command to the memory log.
func Log(cmd, cwd, gitBranch string, exitCode int, durationMs int64) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()

	path, err := filePath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	e := Command{
		At: time.Now(), Cmd: cmd, CWD: cwd,
		GitBranch: gitBranch, ExitCode: exitCode, DurationMs: durationMs,
	}
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}

// SearchOpts controls how Search filters results.
type SearchOpts struct {
	FailedOnly bool
	CWDFilter  string // only commands run in this directory
	Since      time.Time
	Limit      int // 0 = default 20
}

// Search returns commands matching query (case-insensitive substring), newest first.
func Search(query string, opts SearchOpts) ([]Command, error) {
	all, err := readAll()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	var out []Command
	// Iterate newest-first
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		c := all[i]
		if opts.FailedOnly && c.ExitCode == 0 {
			continue
		}
		if opts.CWDFilter != "" && c.CWD != opts.CWDFilter {
			continue
		}
		if !opts.Since.IsZero() && c.At.Before(opts.Since) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Cmd), q) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func readAll() ([]Command, error) {
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

	var all []Command
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 128*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var c Command
		if json.Unmarshal([]byte(line), &c) == nil {
			all = append(all, c)
			if len(all) > maxCommands {
				all = all[len(all)-maxCommands:]
			}
		}
	}
	return all, scanner.Err()
}
