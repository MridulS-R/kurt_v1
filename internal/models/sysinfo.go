package models

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SysInfo struct {
	TotalRAMGB     float64
	AvailableRAMGB float64
	CPUCores       int
	CPUBrand       string
	IsAppleSilicon bool
	HasGPU         bool // Metal on Apple Silicon, or discrete GPU
}

func GetSysInfo() SysInfo {
	info := SysInfo{
		IsAppleSilicon: runtime.GOARCH == "arm64" && runtime.GOOS == "darwin",
	}
	// Apple Silicon always has Metal GPU (unified memory)
	info.HasGPU = info.IsAppleSilicon

	// Total RAM
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			info.TotalRAMGB = float64(n) / (1 << 30)
		}
	}

	// CPU cores
	if out, err := exec.Command("sysctl", "-n", "hw.logicalcpu").Output(); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			info.CPUCores = n
		}
	}

	// CPU brand — works on both Intel and Apple Silicon
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		info.CPUBrand = strings.TrimSpace(string(out))
	}
	if info.CPUBrand == "" {
		if out, err := exec.Command("sysctl", "-n", "hw.model").Output(); err == nil {
			info.CPUBrand = strings.TrimSpace(string(out))
		}
	}

	info.AvailableRAMGB = availableRAMGB()
	return info
}

func availableRAMGB() float64 {
	pageSize := int64(4096)
	if out, err := exec.Command("sysctl", "-n", "hw.pagesize").Output(); err == nil {
		if n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			pageSize = n
		}
	}
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0
	}
	var free, inactive int64
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Pages free:"):
			free = parseVmStatLine(line)
		case strings.HasPrefix(line, "Pages inactive:"):
			inactive = parseVmStatLine(line)
		}
	}
	return float64((free+inactive)*pageSize) / (1 << 30)
}

func parseVmStatLine(line string) int64 {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return 0
	}
	s := strings.TrimSuffix(parts[len(parts)-1], ".")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
