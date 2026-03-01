package suggest

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Args struct {
	Buffer string
	CWD    string
}

var (
	cacheMu      sync.Mutex
	cachedLines  []string
	cachedPath   string
	cachedMtime  time.Time
	cachedSize   int64
	cachedLoaded bool
	maxScanLines = 4000
)

func Suggest(a Args) (string, error) {
	buf := a.Buffer
	cwd := a.CWD

	// If buffer is empty, we don't autosuggest anything by default.
	if strings.TrimSpace(buf) == "" {
		return heuristicEmpty(cwd), nil
	}

	// 1) History prefix match
	hit, err := historyPrefix(buf)
	if err == nil && hit != "" {
		return remainder(buf, hit), nil
	}

	// 2) Heuristics (if buffer is short)
	return heuristicPrefix(buf, cwd), nil
}

func remainder(prefix, full string) string {
	if !strings.HasPrefix(full, prefix) {
		return ""
	}
	return strings.TrimPrefix(full, prefix)
}

func historyPrefix(prefix string) (string, error) {
	path, err := zshHistoryPath()
	if err != nil {
		return "", err
	}
	lines, err := loadHistoryLines(path)
	if err != nil {
		return "", err
	}

	// Scan from newest to oldest.
	p := prefix
	for i := len(lines) - 1; i >= 0; i-- {
		cmd := normalizeHistoryLine(lines[i])
		if cmd == "" {
			continue
		}
		if strings.HasPrefix(cmd, p) {
			return cmd, nil
		}
	}
	return "", nil
}

func zshHistoryPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("HISTFILE")); p != "" {
		return p, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".zsh_history"), nil
}

func loadHistoryLines(path string) ([]string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cachedLoaded && cachedPath == path && st.ModTime().Equal(cachedMtime) && st.Size() == cachedSize {
		return cachedLines, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Allow longer lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lines := make([]string, 0, 1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		// keep memory bounded
		if len(lines) > maxScanLines {
			lines = lines[len(lines)-maxScanLines:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	cachedLines = lines
	cachedPath = path
	cachedMtime = st.ModTime()
	cachedSize = st.Size()
	cachedLoaded = true
	return lines, nil
}

func normalizeHistoryLine(line string) string {
	// zsh extended history format:
	// : 1700000000:0;command here
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if strings.HasPrefix(line, ": ") {
		if idx := strings.Index(line, ";"); idx >= 0 {
			return strings.TrimSpace(line[idx+1:])
		}
		return ""
	}
	return line
}

func heuristicEmpty(cwd string) string {
	// If user hasn't typed anything, we can optionally suggest a context starter.
	// Keep conservative: only suggest in git repos.
	if isGitRepo(cwd) {
		return "git status"
	}
	return ""
}

func heuristicPrefix(prefix, cwd string) string {
	p := strings.ToLower(strings.TrimSpace(prefix))
	if strings.HasPrefix(p, "git ") {
		if strings.HasPrefix(p, "git st") {
			return remainder(prefix, "git status")
		}
		if strings.HasPrefix(p, "git di") {
			return remainder(prefix, "git diff")
		}
	}
	if strings.HasPrefix(p, "go ") && fileExists(filepath.Join(cwd, "go.mod")) {
		if strings.HasPrefix(p, "go te") {
			return remainder(prefix, "go test ./...")
		}
	}
	return ""
}

func isGitRepo(cwd string) bool {
	// cheap check: look for .git directory in current or parents
	d := cwd
	for i := 0; i < 6; i++ {
		if fileExists(filepath.Join(d, ".git")) {
			return true
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return false
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

var ErrNotAvailable = errors.New("suggest not available")
