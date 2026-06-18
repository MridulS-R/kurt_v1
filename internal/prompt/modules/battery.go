package modules

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// BatteryModule shows battery percentage (macOS and Linux).
// ShowThreshold: only show when battery is at or below this %; 0 = always show.
type BatteryModule struct {
	ShowThreshold int
}

func (BatteryModule) Name() string { return "battery" }

func (m BatteryModule) Render(ctx Context) (string, bool) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("pmset", "-g", "batt").Output()
		if err != nil {
			return "", false
		}
		return m.parsePmset(string(out))
	case "linux":
		return batteryLinux(m.ShowThreshold)
	}
	return "", false
}

// parsePmset parses pmset -g batt output and returns the display segment.
func (m BatteryModule) parsePmset(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, "%")
		if idx < 0 {
			continue
		}
		start := idx - 1
		for start > 0 && line[start-1] >= '0' && line[start-1] <= '9' {
			start--
		}
		if start < 0 || start > idx {
			continue
		}
		pctStr := strings.TrimSpace(line[start : idx+1])
		pct, err := strconv.Atoi(strings.TrimRight(pctStr, "%"))
		if err != nil {
			continue
		}
		charging := strings.Contains(line, "charging")
		// Suppress when charging at 100%.
		if charging && pct == 100 {
			return "", false
		}
		// Suppress when above threshold (if threshold is set).
		if m.ShowThreshold > 0 && pct > m.ShowThreshold {
			return "", false
		}
		icon := batteryIcon(pct, charging)
		return fmt.Sprintf("%s%d%%", icon, pct), true
	}
	return "", false
}

func batteryLinux(threshold int) (string, bool) {
	readInt := func(path string) (int, bool) {
		out, err := exec.Command("cat", path).Output()
		if err != nil {
			return 0, false
		}
		n, err := strconv.Atoi(strings.TrimSpace(string(out)))
		return n, err == nil
	}
	pct, ok := readInt("/sys/class/power_supply/BAT0/capacity")
	if !ok {
		pct, ok = readInt("/sys/class/power_supply/BAT1/capacity")
		if !ok {
			return "", false
		}
	}
	statusOut, _ := exec.Command("cat", "/sys/class/power_supply/BAT0/status").Output()
	charging := strings.Contains(strings.ToLower(string(statusOut)), "charging")
	if charging && pct == 100 {
		return "", false
	}
	if threshold > 0 && pct > threshold {
		return "", false
	}
	icon := batteryIcon(pct, charging)
	return fmt.Sprintf("%s%d%%", icon, pct), true
}

func batteryIcon(pct int, charging bool) string {
	if charging {
		return "⚡"
	}
	if pct < 20 {
		return "🪫"
	}
	return "🔋"
}
