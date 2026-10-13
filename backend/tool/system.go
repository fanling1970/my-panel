package tool

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// MemInfo 系统内存信息
type MemInfo struct{ Usage float64; Used uint64; Total uint64 }

// DiskInfo 磁盘使用信息
type DiskInfo struct{ Usage float64; Used uint64; Total uint64 }

// GetCPUUsage 获取 CPU 使用率（仅 Linux）
func GetCPUUsage() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("cannot read /proc/stat")
	}

	line := scanner.Text()
	var user, nice, system, idle, iowait, irq, softirq uint64
	n, _ := fmt.Sscanf(line, "cpu %d %d %d %d %d %d %d", &user, &nice, &system, &idle, &iowait, &irq, &softirq)
	if n < 4 {
		return 0, fmt.Errorf("invalid /proc/stat format")
	}

	total := user + nice + system + idle + iowait + irq + softirq
	idleTotal := idle + iowait

	if total == 0 {
		return 0, nil
	}

	usage := 100.0 * (1.0 - float64(idleTotal)/float64(total))
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

// GetMemoryUsage 获取内存使用率（仅 Linux）
func GetMemoryUsage() (MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
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

// GetDiskUsage 获取磁盘使用率（仅 Linux）
func GetDiskUsage() (DiskInfo, error) {
	cmd := exec.Command("df", "-k", "/")
	output, err := cmd.Output()
	if err != nil {
		return DiskInfo{}, err
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

// GetHostInfo 获取主机信息（Linux 容器内有效，其他平台容错返回空值）
func GetHostInfo() HostInfo {
	info := HostInfo{
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	info.Kernel = readFileString("/proc/sys/kernel/osrelease")

	// 从 /etc/os-release 解析 PRETTY_NAME
	if f, err := os.Open("/etc/os-release"); err == nil {
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
	}
	if info.OS == "" {
		info.OS = runtime.GOOS
	}

	// /proc/uptime 第一个字段是秒
	if up := readFileString("/proc/uptime"); up != "" {
		fields := strings.Fields(up)
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				info.Uptime = int64(sec)
			}
		}
	}

	return info
}
