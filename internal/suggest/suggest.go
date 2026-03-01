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

	// ---- git ----
	if strings.HasPrefix(p, "git ") {
		switch {
		case strings.HasPrefix(p, "git st"):
			return remainder(prefix, "git status")
		case strings.HasPrefix(p, "git di"):
			return remainder(prefix, "git diff")
		case strings.HasPrefix(p, "git co"):
			return remainder(prefix, "git commit")
		}
	}

	// ---- docker ----
	if strings.HasPrefix(p, "docker ") {
		switch {
		case strings.HasPrefix(p, "docker ps"):
			return remainder(prefix, "docker ps")
		case strings.HasPrefix(p, "docker im"):
			return remainder(prefix, "docker images")
		case strings.HasPrefix(p, "docker lo"):
			return remainder(prefix, "docker logs -f ")
		case strings.HasPrefix(p, "docker ex"):
			return remainder(prefix, "docker exec -it ")
		}
	}
	if strings.HasPrefix(p, "docker-compose ") || strings.HasPrefix(p, "docker compose ") {
		// Only if a compose file exists (best-effort)
		if hasComposeFile(cwd) {
			switch {
			case strings.HasPrefix(p, "docker compose u"):
				return remainder(prefix, "docker compose up -d")
			case strings.HasPrefix(p, "docker compose d"):
				return remainder(prefix, "docker compose down")
			case strings.HasPrefix(p, "docker compose l"):
				return remainder(prefix, "docker compose logs -f")
			}
		}
	}

	// ---- kubectl ----
	if strings.HasPrefix(p, "kubectl ") {
		switch {
		case strings.HasPrefix(p, "kubectl g"):
			return remainder(prefix, "kubectl get ")
		case strings.HasPrefix(p, "kubectl d"):
			return remainder(prefix, "kubectl describe ")
		case strings.HasPrefix(p, "kubectl a"):
			return remainder(prefix, "kubectl apply -f ")
		case strings.HasPrefix(p, "kubectl l"):
			return remainder(prefix, "kubectl logs -f ")
		case strings.HasPrefix(p, "kubectl c"):
			return remainder(prefix, "kubectl config ")
		}
	}

	// ---- aws ----
	if strings.HasPrefix(p, "aws ") {
		switch {
		case strings.HasPrefix(p, "aws s3 l"):
			return remainder(prefix, "aws s3 ls")
		case strings.HasPrefix(p, "aws st"):
			return remainder(prefix, "aws sts get-caller-identity")
		case strings.HasPrefix(p, "aws ec2 d"):
			return remainder(prefix, "aws ec2 describe-instances")
		}
	}

	// ---- go ----
	if strings.HasPrefix(p, "go ") && fileExists(filepath.Join(cwd, "go.mod")) {
		switch {
		case strings.HasPrefix(p, "go te"):
			return remainder(prefix, "go test ./...")
		case strings.HasPrefix(p, "go ru"):
			return remainder(prefix, "go run .")
		case strings.HasPrefix(p, "go bu"):
			return remainder(prefix, "go build ./...")
		case strings.HasPrefix(p, "go fm"):
			return remainder(prefix, "gofmt -w .")
		}
	}

	// ---- ruby ----
	if strings.HasPrefix(p, "bundle ") && fileExists(filepath.Join(cwd, "Gemfile")) {
		switch {
		case strings.HasPrefix(p, "bundle i"):
			return remainder(prefix, "bundle install")
		case strings.HasPrefix(p, "bundle e"):
			return remainder(prefix, "bundle exec ")
		}
	}
	if strings.HasPrefix(p, "rails ") && fileExists(filepath.Join(cwd, "Gemfile")) {
		if strings.HasPrefix(p, "rails s") {
			return remainder(prefix, "rails server")
		}
	}

	// ---- Elasticsearch ----
	if strings.HasPrefix(p, "curl ") {
		// common ES endpoints
		if strings.Contains(p, "localhost:9200") {
			if strings.HasSuffix(p, "_cat") || strings.Contains(p, "_cat") {
				return ""
			}
		}
	}
	if strings.HasPrefix(p, "es ") {
		// lightweight aliases people often use (user can create real alias)
		if strings.HasPrefix(p, "es ca") {
			return remainder(prefix, "es cat")
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

func hasComposeFile(cwd string) bool {
	// Check common compose filenames in cwd.
	cands := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, name := range cands {
		if fileExists(filepath.Join(cwd, name)) {
			return true
		}
	}
	return false
}

var ErrNotAvailable = errors.New("suggest not available")
