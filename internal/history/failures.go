package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Failure struct {
	Cmd      string    `json:"cmd"`
	CWD      string    `json:"cwd"`
	ExitCode int       `json:"exit_code"`
	At       time.Time `json:"at"`
}

const maxFailures = 2000

var mu sync.Mutex

func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".local", "share", "kurt")
	return d, os.MkdirAll(d, 0700)
}

func failuresPath() (string, error) {
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "failures.jsonl"), nil
}

// LogFailure appends a failed command entry to the failure log.
func LogFailure(cmd, cwd string, exitCode int) error {
	mu.Lock()
	defer mu.Unlock()

	path, err := failuresPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	e := Failure{
		Cmd:      strings.TrimSpace(cmd),
		CWD:      cwd,
		ExitCode: exitCode,
		At:       time.Now(),
	}
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}

// Recent returns the last n failure entries (oldest first).
func Recent(n int) ([]Failure, error) {
	path, err := failuresPath()
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

	var all []Failure
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Failure
		if json.Unmarshal([]byte(line), &e) == nil {
			all = append(all, e)
			if len(all) > maxFailures {
				all = all[len(all)-maxFailures:]
			}
		}
	}
	if len(all) > n {
		return all[len(all)-n:], nil
	}
	return all, nil
}

// Lookup returns recent failures whose command starts with prefix (newest first, up to limit).
func Lookup(cmdPrefix string, limit int) ([]Failure, error) {
	all, err := Recent(maxFailures)
	if err != nil || len(all) == 0 {
		return nil, err
	}
	prefix := strings.ToLower(strings.TrimSpace(cmdPrefix))
	var out []Failure
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(all[i].Cmd)), prefix) {
			out = append(out, all[i])
		}
	}
	return out, nil
}
