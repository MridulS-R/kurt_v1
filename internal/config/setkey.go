package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetKey writes a single key=value into the TOML config file.
// It handles dot-notation keys (e.g. "think.provider") by mapping to TOML
// sections. Only a curated set of common keys is supported.
func SetKey(path, key, value string) error {
	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Read existing lines (or start fresh)
	var lines []string
	if f, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		f.Close()
	}

	section, field, tomlVal, err := parseKeyValue(key, value)
	if err != nil {
		return err
	}

	lines = upsertTOML(lines, section, field, tomlVal)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	return w.Flush()
}

// parseKeyValue maps a dot-notation key to (section, field, tomlValue).
func parseKeyValue(key, value string) (section, field, tomlVal string, err error) {
	// Quote strings, leave booleans/numbers bare
	quote := func(v string) string {
		switch strings.ToLower(v) {
		case "true", "false":
			return v
		}
		// Check if it looks numeric
		if len(v) > 0 {
			allNum := true
			for _, c := range v {
				if c < '0' || c > '9' {
					allNum = false
					break
				}
			}
			if allNum {
				return v
			}
		}
		return fmt.Sprintf("%q", v)
	}

	switch key {
	case "style":
		return "", "style", quote(value), nil
	case "prompt.two_line":
		return "prompt", "two_line", quote(value), nil
	case "think.provider":
		return "think", "provider", quote(value), nil
	case "think.model":
		return "think", "model", quote(value), nil
	case "think.base_url":
		return "think", "base_url", quote(value), nil
	case "think.host":
		return "think", "host", quote(value), nil
	case "perf.git_ttl_ms":
		return "perf", "git_ttl_ms", value, nil
	case "rprompt.enabled":
		return "rprompt", "enabled", quote(value), nil
	case "rprompt.time_format":
		return "rprompt", "time_format", quote(value), nil
	case "module.dir.enabled":
		return "module.dir", "enabled", quote(value), nil
	case "module.git.enabled":
		return "module.git", "enabled", quote(value), nil
	case "module.duration.enabled":
		return "module.duration", "enabled", quote(value), nil
	case "module.duration.min_ms":
		return "module.duration", "min_ms", value, nil
	case "module.gpu.enabled":
		return "module.gpu", "enabled", quote(value), nil
	case "module.venv.enabled":
		return "module.venv", "enabled", quote(value), nil
	case "module.conda.enabled":
		return "module.conda", "enabled", quote(value), nil
	case "module.node.enabled":
		return "module.node", "enabled", quote(value), nil
	case "module.kube.enabled":
		return "module.kube", "enabled", quote(value), nil
	case "module.battery.enabled":
		return "module.battery", "enabled", quote(value), nil
	case "module.python.enabled":
		return "module.python", "enabled", quote(value), nil
	case "module.cloud.enabled":
		return "module.cloud", "enabled", quote(value), nil
	case "module.time.enabled":
		return "module.time", "enabled", quote(value), nil
	case "module.time.format":
		return "module.time", "format", quote(value), nil
	}
	return "", "", "", fmt.Errorf("unsupported key %q — run 'kurt config view' to see settable keys", key)
}

// upsertTOML finds or creates the section and sets the field.
func upsertTOML(lines []string, section, field, tomlVal string) []string {
	assignment := field + " = " + tomlVal
	sectionHeader := ""
	if section != "" {
		sectionHeader = "[" + section + "]"
	}

	// Top-level key (no section)
	if section == "" {
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), field+" ") ||
				strings.HasPrefix(strings.TrimSpace(l), field+"=") {
				lines[i] = assignment
				return lines
			}
		}
		return append(lines, assignment)
	}

	// Find existing section
	inSection := false
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == sectionHeader {
			inSection = true
			continue
		}
		if inSection {
			// End of section
			if strings.HasPrefix(trimmed, "[") {
				// Insert before next section
				before := append([]string{}, lines[:i]...)
				before = append(before, assignment)
				return append(before, lines[i:]...)
			}
			// Check if key exists in section
			if strings.HasPrefix(trimmed, field+" ") || strings.HasPrefix(trimmed, field+"=") {
				lines[i] = assignment
				return lines
			}
		}
	}

	if inSection {
		// Append at end of file (we're still in the last section)
		return append(lines, assignment)
	}

	// Section doesn't exist yet — append it
	lines = append(lines, "", sectionHeader, assignment)
	return lines
}
