package modules

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// BatteryModule shows battery percentage (macOS and Linux).
type BatteryModule struct{}

func (BatteryModule) Name() string { return "battery" }

func (BatteryModule) Render(ctx Context) (string, bool) {
	switch runtime.GOOS {
	case "darwin":
		return batteryMacOS()
	case "linux":
		return batteryLinux()
	}
	return "", false
}

func batteryMacOS() (string, bool) {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return "", false
	}
	// Look for a line like: "100%; charging" or "85%; discharging"
	for _, line := range strings.Split(string(out), "\n") {
		idx := strings.Index(line, "%")
		if idx < 0 {
			continue
		}
		// Find the number before '%'
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
		icon := batteryIcon(pct, strings.Contains(line, "charging"))
		return fmt.Sprintf("%s%d%%", icon, pct), true
	}
	return "", false
}

func batteryLinux() (string, bool) {
	out, err := exec.Command("cat", "/sys/class/power_supply/BAT0/capacity").Output()
	if err != nil {
		// Try BAT1
		out, err = exec.Command("cat", "/sys/class/power_supply/BAT1/capacity").Output()
		if err != nil {
			return "", false
		}
	}
	pct, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return "", false
	}
	statusOut, _ := exec.Command("cat", "/sys/class/power_supply/BAT0/status").Output()
	charging := strings.Contains(strings.ToLower(string(statusOut)), "charging")
	icon := batteryIcon(pct, charging)
	return fmt.Sprintf("%s%d%%", icon, pct), true
}

func batteryIcon(pct int, charging bool) string {
	if charging {
		return "⚡"
	}
	switch {
	case pct >= 80:
		return "🔋"
	case pct >= 40:
		return "🔋"
	default:
		return "🪫"
	}
}
