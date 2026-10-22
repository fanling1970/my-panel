package tool

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// MemInfo 系统内存信息
type MemInfo struct{ Usage float64; Used uint64; Total uint64 }

// DiskInfo 磁盘使用信息
type DiskInfo struct{ Usage float64; Used uint64; Total uint64 }

// cpuSnapshot /proc/stat 采样
type cpuSnapshot struct {
	total     uint64
	idleTotal uint64
}

var hostPrefix = "/host"

func init() {
	// 检测宿主机根文件系统挂载：/host/proc 存在说明有 -v /:/host；没有则回退容器自身 /proc
	if _, err := os.Stat("/host/proc/stat"); err != nil {
		hostPrefix = ""
	}
}

func hostPath(p string) string {
	return hostPrefix + p
}

func snapCpu() (cpuSnapshot, error) {
	file, err := os.Open(hostPath("/proc/stat"))
	if err != nil {
		return cpuSnapshot{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return cpuSnapshot{}, fmt.Errorf("cannot read /proc/stat")
	}
	line := scanner.Text()
	var user, nice, system, idle, iowait, irq, softirq uint64
	n, _ := fmt.Sscanf(line, "cpu %d %d %d %d %d %d %d", &user, &nice, &system, &idle, &iowait, &irq, &softirq)
	if n < 4 {
		return cpuSnapshot{}, fmt.Errorf("invalid /proc/stat format")
	}
	total := user + nice + system + idle + iowait + irq + softirq
	return cpuSnapshot{total: total, idleTotal: idle + iowait}, nil
}

// GetCPUUsage 获取宿主机 CPU 使用率（采样两次 /proc/stat 计算差值）
func GetCPUUsage() (float64, error) {
	a, err := snapCpu()
	if err != nil {
		return 0, err
	}
	time.Sleep(600 * time.Millisecond)
	b, err := snapCpu()
	if err != nil {
		return 0, err
	}
	totalDelta := b.total - a.total
	idleDelta := b.idleTotal - a.idleTotal
	if totalDelta == 0 {
		return 0, nil
	}
	usage := 100.0 * (1.0 - float64(idleDelta)/float64(totalDelta))
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return usage, nil
}

// GetMemoryUsage 获取宿主机内存使用率
func GetMemoryUsage() (MemInfo, error) {
	file, err := os.Open(hostPath("/proc/meminfo"))
	if err != nil {
		return MemInfo{}, err
	}
	defer file.Close()

	var total, free, available, buffers, cached uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &total)
		} else if strings.HasPrefix(line, "MemFree:") {
			fmt.Sscanf(line, "MemFree: %d kB", &free)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &available)
		} else if strings.HasPrefix(line, "Buffers:") {
			fmt.Sscanf(line, "Buffers: %d kB", &buffers)
		} else if strings.HasPrefix(line, "Cached:") {
			fmt.Sscanf(line, "Cached: %d kB", &cached)
		}
	}

	if total == 0 {
		return MemInfo{}, fmt.Errorf("cannot read memory info")
	}

	used := total - available
	if available == 0 {
		used = total - free - buffers - cached
	}

	usage := 100.0 * float64(used) / float64(total)
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return MemInfo{Usage: usage, Used: used, Total: total}, nil
}

// GetDiskUsage 获取宿主机根分区磁盘使用率
func GetDiskUsage() (DiskInfo, error) {
	// 优先读宿主根分区（df -k /host），没有 host 挂载时退回到 /
	target := "/"
	if hostPrefix != "" {
		target = "/host"
	}
	cmd := exec.Command("df", "-k", target)
	output, err := cmd.Output()
	if err != nil {
		// 宿主不存在时回退
		cmd2 := exec.Command("df", "-k", "/")
		output, err = cmd2.Output()
		if err != nil {
			return DiskInfo{}, err
		}
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return DiskInfo{}, fmt.Errorf("cannot parse disk info")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return DiskInfo{}, fmt.Errorf("disk info format error")
	}

	total, _ := strconv.ParseUint(fields[1], 10, 64)
	used, _ := strconv.ParseUint(fields[2], 10, 64)
	usageStr := strings.TrimSuffix(fields[4], "%")
	usage, _ := strconv.ParseFloat(usageStr, 64)

	return DiskInfo{Usage: usage, Used: used, Total: total}, nil
}

// HostInfo 主机信息
type HostInfo struct {
	Hostname string
	OS       string
	Kernel   string
	Arch     string
	CPUCores int
	Uptime   int64 // 秒
}

// readFileString 读取文件全部内容（容错，不存在返回空）
func readFileString(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// GetHostInfo 获取宿主机信息（优先读 /host 下的文件，没有则退回到容器自身）
func GetHostInfo() HostInfo {
	info := HostInfo{
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	// 宿主机内核版本
	if k := readFileString(hostPath("/proc/sys/kernel/osrelease")); k != "" {
		info.Kernel = k
	} else {
		info.Kernel = readFileString("/proc/sys/kernel/osrelease")
	}

	// 从 /etc/os-release 解析 PRETTY_NAME（优先宿主的）
	var triedPaths []string
	if hostPrefix != "" {
		triedPaths = append(triedPaths, "/host/etc/os-release")
	}
	triedPaths = append(triedPaths, "/etc/os-release")
	for _, p := range triedPaths {
		if f, err := os.Open(p); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := sc.Text()
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					v := strings.TrimPrefix(line, "PRETTY_NAME=")
					info.OS = strings.Trim(v, `"`)
					break
				}
			}
			f.Close()
			if info.OS != "" {
				break
			}
		}
	}
	if info.OS == "" {
		info.OS = runtime.GOOS
	}

	// /proc/uptime（宿主机的）
	if up := readFileString(hostPath("/proc/uptime")); up != "" {
		fields := strings.Fields(up)
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				info.Uptime = int64(sec)
			}
		}
	} else if up := readFileString("/proc/uptime"); up != "" {
		fields := strings.Fields(up)
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				info.Uptime = int64(sec)
			}
		}
	}

	return info
}
