package monitor

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// CPUMetrics contains process and system CPU utilization stats.
type CPUMetrics struct {
	ProcessCPUPercent float64 `json:"process_cpu_percent"`
	SystemCPUPercent  float64 `json:"system_cpu_percent"`
	NumCPU            int     `json:"num_cpu"`
	NumGoroutine      int     `json:"num_goroutine"`
	NumCgoCall        int64   `json:"num_cgo_call"`
}

// MemoryMetrics contains memory allocation and GC statistics.
type MemoryMetrics struct {
	Alloc          uint64  `json:"alloc_bytes"`
	AllocHuman     string  `json:"alloc_human"`
	TotalAlloc     uint64  `json:"total_alloc_bytes"`
	Sys            uint64  `json:"sys_bytes"`
	SysHuman       string  `json:"sys_human"`
	HeapAlloc      uint64  `json:"heap_alloc_bytes"`
	HeapAllocHuman string  `json:"heap_alloc_human"`
	HeapSys        uint64  `json:"heap_sys_bytes"`
	HeapIdle       uint64  `json:"heap_idle_bytes"`
	HeapInuse      uint64  `json:"heap_inuse_bytes"`
	HeapReleased   uint64  `json:"heap_released_bytes"`
	HeapObjects    uint64  `json:"heap_objects"`
	StackInuse     uint64  `json:"stack_inuse_bytes"`
	StackSys       uint64  `json:"stack_sys_bytes"`
	NumGC          uint32  `json:"num_gc"`
	LastGC         string  `json:"last_gc"`
	GCCPUFraction  float64 `json:"gc_cpu_fraction"`
}

type procStat struct {
	user   uint64
	nice   uint64
	system uint64
	idle   uint64
	iowait uint64
	irq    uint64
	soft   uint64
	steal  uint64
}

func (s procStat) total() uint64 {
	return s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.soft + s.steal
}

func (s procStat) work() uint64 {
	return s.user + s.nice + s.system + s.irq + s.soft + s.steal
}

type cpuSampler struct {
	mu sync.RWMutex

	lastSampleTime   time.Time
	lastProcessUTime time.Duration
	lastProcessSTime time.Duration
	lastProcStat     *procStat

	processPercent float64
	systemPercent  float64
}

var globalCPUSampler = &cpuSampler{
	lastSampleTime: time.Now(),
}

func init() {
	globalCPUSampler.sample()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			globalCPUSampler.sample()
		}
	}()
}

func (s *cpuSampler) sample() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	wallDelta := now.Sub(s.lastSampleTime).Seconds()
	if wallDelta <= 0 {
		return
	}

	numCPU := float64(runtime.NumCPU())
	if numCPU <= 0 {
		numCPU = 1
	}

	// 1. Process CPU usage via syscall.Getrusage
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
		utime := time.Duration(rusage.Utime.Sec)*time.Second + time.Duration(rusage.Utime.Usec)*time.Microsecond
		stime := time.Duration(rusage.Stime.Sec)*time.Second + time.Duration(rusage.Stime.Usec)*time.Microsecond

		if s.lastProcessUTime != 0 || s.lastProcessSTime != 0 {
			procDelta := (utime - s.lastProcessUTime).Seconds() + (stime - s.lastProcessSTime).Seconds()
			s.processPercent = (procDelta / (wallDelta * numCPU)) * 100.0
			if s.processPercent < 0 {
				s.processPercent = 0
			}
		}
		s.lastProcessUTime = utime
		s.lastProcessSTime = stime
	}

	// 2. System CPU usage via /proc/stat if running on Linux
	if stat, err := readProcStat(); err == nil {
		if s.lastProcStat != nil {
			totalDelta := float64(stat.total() - s.lastProcStat.total())
			workDelta := float64(stat.work() - s.lastProcStat.work())
			if totalDelta > 0 {
				s.systemPercent = (workDelta / totalDelta) * 100.0
				if s.systemPercent < 0 {
					s.systemPercent = 0
				}
				if s.systemPercent > 100 {
					s.systemPercent = 100
				}
			}
		}
		s.lastProcStat = stat
	} else {
		// Fallback: system percent mirrors process percent if /proc/stat is unavailable
		s.systemPercent = s.processPercent
	}

	s.lastSampleTime = now
}

func readProcStat() (*procStat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				break
			}
			stat := &procStat{}
			stat.user, _ = strconv.ParseUint(fields[1], 10, 64)
			stat.nice, _ = strconv.ParseUint(fields[2], 10, 64)
			stat.system, _ = strconv.ParseUint(fields[3], 10, 64)
			stat.idle, _ = strconv.ParseUint(fields[4], 10, 64)
			if len(fields) > 5 {
				stat.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
			}
			if len(fields) > 6 {
				stat.irq, _ = strconv.ParseUint(fields[6], 10, 64)
			}
			if len(fields) > 7 {
				stat.soft, _ = strconv.ParseUint(fields[7], 10, 64)
			}
			if len(fields) > 8 {
				stat.steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}
			return stat, nil
		}
	}
	return nil, fmt.Errorf("cpu line not found in /proc/stat")
}

// GetCPUMetrics returns current CPU utilization and Goroutine counts.
func GetCPUMetrics() CPUMetrics {
	globalCPUSampler.mu.RLock()
	procPct := globalCPUSampler.processPercent
	sysPct := globalCPUSampler.systemPercent
	globalCPUSampler.mu.RUnlock()

	return CPUMetrics{
		ProcessCPUPercent: round2(procPct),
		SystemCPUPercent:  round2(sysPct),
		NumCPU:            runtime.NumCPU(),
		NumGoroutine:      runtime.NumGoroutine(),
		NumCgoCall:        runtime.NumCgoCall(),
	}
}

// GetMemoryMetrics returns current runtime memory statistics.
func GetMemoryMetrics() MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	lastGCTime := "never"
	if m.LastGC > 0 {
		lastGCTime = time.Unix(0, int64(m.LastGC)).Format(time.RFC3339)
	}

	return MemoryMetrics{
		Alloc:          m.Alloc,
		AllocHuman:     FormatBytes(m.Alloc),
		TotalAlloc:     m.TotalAlloc,
		Sys:            m.Sys,
		SysHuman:       FormatBytes(m.Sys),
		HeapAlloc:      m.HeapAlloc,
		HeapAllocHuman: FormatBytes(m.HeapAlloc),
		HeapSys:        m.HeapSys,
		HeapIdle:       m.HeapIdle,
		HeapInuse:      m.HeapInuse,
		HeapReleased:   m.HeapReleased,
		HeapObjects:    m.HeapObjects,
		StackInuse:     m.StackInuse,
		StackSys:       m.StackSys,
		NumGC:          m.NumGC,
		LastGC:         lastGCTime,
		GCCPUFraction:  m.GCCPUFraction,
	}
}

// FormatBytes formats byte counts into human-readable strings (KB, MB, GB).
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func round2(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}
