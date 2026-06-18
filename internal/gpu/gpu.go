package gpu

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stats holds GPU/accelerator utilisation for one device.
type Stats struct {
	Kind        string // "nvidia" | "amd" | "apple" | ""
	Name        string
	UtilPct     int // 0-100, -1 if unavailable
	MemUsedMiB  int
	MemTotalMiB int
}

// ── cache ─────────────────────────────────────────────────────────────────────

var (
	cacheMu  sync.Mutex
	cached   *Stats
	cachedAt time.Time
)

// Get returns GPU stats, cached for ttl. Returns nil when no GPU is detected.
func Get(ttl time.Duration) *Stats {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cached != nil && time.Since(cachedAt) < ttl {
		return cached
	}
	s := collect()
	cached = s
	cachedAt = time.Now()
	return s
}

// ── collection ────────────────────────────────────────────────────────────────

func collect() *Stats {
	// 1. NVIDIA (any OS)
	if s := collectNvidia(); s != nil {
		return s
	}
	// 2. AMD (Linux / macOS)
	if s := collectAMD(); s != nil {
		return s
	}
	// 3. Apple Silicon unified memory (macOS only)
	if runtime.GOOS == "darwin" {
		return collectAppleSilicon()
	}
	return nil
}

// ── NVIDIA ────────────────────────────────────────────────────────────────────

func collectNvidia() *Stats {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	parts := strings.Split(line, ",")
	if len(parts) < 4 {
		return nil
	}
	util, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	used, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	total, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	if total == 0 {
		return nil
	}
	return &Stats{
		Kind:        "nvidia",
		Name:        strings.TrimSpace(parts[0]),
		UtilPct:     util,
		MemUsedMiB:  used,
		MemTotalMiB: total,
	}
}

// ── AMD ───────────────────────────────────────────────────────────────────────

func collectAMD() *Stats {
	out, err := exec.Command("rocm-smi",
		"--showuse", "--showmemuse", "--csv",
	).Output()
	if err != nil {
		return nil
	}
	// rocm-smi CSV header: device,GPU use (%),VRAM Total Memory (B),VRAM Total Used Memory (B),...
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[1:] { // skip header
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		util, err1 := strconv.Atoi(strings.TrimSpace(parts[1]))
		totalB, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		usedB, err3 := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || totalB == 0 {
			continue
		}
		return &Stats{
			Kind:        "amd",
			Name:        "AMD GPU",
			UtilPct:     util,
			MemUsedMiB:  int(usedB / (1024 * 1024)),
			MemTotalMiB: int(totalB / (1024 * 1024)),
		}
	}
	return nil
}

// ── Apple Silicon (unified memory) ───────────────────────────────────────────

func collectAppleSilicon() *Stats {
	// Total physical memory via sysctl
	totalOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return nil
	}
	totalBytes, err := strconv.ParseInt(strings.TrimSpace(string(totalOut)), 10, 64)
	if err != nil || totalBytes == 0 {
		return nil
	}

	// Page size
	pageSizeOut, err := exec.Command("sysctl", "-n", "hw.pagesize").Output()
	if err != nil {
		return nil
	}
	pageSize, err := strconv.ParseInt(strings.TrimSpace(string(pageSizeOut)), 10, 64)
	if err != nil || pageSize == 0 {
		pageSize = 16384 // M-chip default
	}

	// vm_stat for active + wired pages
	vmOut, err := exec.Command("vm_stat").Output()
	if err != nil {
		return nil
	}
	activePages := parseVMStatLine(string(vmOut), "Pages active:")
	wiredPages := parseVMStatLine(string(vmOut), "Pages wired down:")
	if activePages == 0 && wiredPages == 0 {
		return nil
	}

	usedBytes := (activePages + wiredPages) * pageSize
	totalMiB := int(totalBytes / (1024 * 1024))
	usedMiB := int(usedBytes / (1024 * 1024))
	if usedMiB > totalMiB {
		usedMiB = totalMiB
	}

	return &Stats{
		Kind:        "apple",
		Name:        "Unified Memory",
		UtilPct:     -1, // GPU util not available without sudo
		MemUsedMiB:  usedMiB,
		MemTotalMiB: totalMiB,
	}
}

func parseVMStatLine(vmOutput, label string) int64 {
	for _, line := range strings.Split(vmOutput, "\n") {
		if strings.Contains(line, label) {
			// Format: "Pages active:    12345."
			parts := strings.Fields(line)
			for _, p := range parts {
				p = strings.TrimSuffix(p, ".")
				if n, err := strconv.ParseInt(p, 10, 64); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

// ── formatting ────────────────────────────────────────────────────────────────

// Label returns a short prompt-friendly string like "GPU 45% 8G/16G" or "MEM 10G/16G".
func (s *Stats) Label() string {
	if s == nil {
		return ""
	}
	used := mibToGiB(s.MemUsedMiB)
	total := mibToGiB(s.MemTotalMiB)

	switch s.Kind {
	case "apple":
		return "MEM " + used + "/" + total
	default:
		if s.UtilPct >= 0 {
			return "GPU " + strconv.Itoa(s.UtilPct) + "% " + used + "/" + total
		}
		return "GPU " + used + "/" + total
	}
}

func mibToGiB(mib int) string {
	if mib == 0 {
		return "0G"
	}
	gib := float64(mib) / 1024.0
	if gib < 10 {
		// Show one decimal for small values: 1.5G
		s := strconv.FormatFloat(gib, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
		return s + "G"
	}
	return strconv.Itoa(int(gib+0.5)) + "G"
}
