//go:build !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Detect 检测是否具备直通宿主机的条件
func (s *HostShellService) Detect() HostShellMode {
	nsenterPath, err := exec.LookPath("nsenter")
	if err != nil {
		return HostShellMode{Available: false, Reason: "容器内未安装 nsenter"}
	}

	// pid: host 开启时，容器内 /proc/1 是宿主机的 init 进程，其 mount 命名空间
	// 与容器自身的 mount 命名空间不同；未开启 pid: host 时，/proc/1 是容器自己
	// 的 init，mount 命名空间与容器自身相同。
	// 注意：不能对比 /proc/1/ns/pid 与 /proc/self/ns/pid——两种情况下二者都处于
	// 同一 PID 命名空间（pid: host 时同属宿主机，未开启时同属容器），对比恒为相同。
	same, err := sameNamespace("/proc/1/ns/mnt", "/proc/self/ns/mnt")
	if err != nil {
		return HostShellMode{Available: false, Reason: "无法读取命名空间信息: " + err.Error()}
	}
	if same {
		return HostShellMode{Available: false, Reason: "容器未开启 pid: host，无法直通宿主机"}
	}

	// 再确认宿主机根文件系统可见（/bin 或 /etc 任一可读即可，兼容精简固件）
	rootOK := false
	for _, p := range []string{"/proc/1/root/bin", "/proc/1/root/etc", "/proc/1/root/sbin"} {
		if _, err := os.Stat(p); err == nil {
			rootOK = true
			break
		}
	}
	if !rootOK {
		return HostShellMode{Available: false, Reason: "无法访问宿主机文件系统（缺少权限？）"}
	}

	// 真实探测：进入宿主机 PID/mount 命名空间执行 true，把实际失败原因暴露给前端
	probe := exec.Command(nsenterPath, "-t", "1", "--pid", "--mount", "--", "true")
	if out, perr := probe.CombinedOutput(); perr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = perr.Error()
		}
		return HostShellMode{Available: false, Reason: "nsenter 探测失败: " + msg}
	}

	return HostShellMode{Available: true}
}

// sameNamespace 比较两个命名空间文件的 inode 是否相同
func sameNamespace(a, b string) (bool, error) {
	fa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	sa, okA := fa.Sys().(*syscall.Stat_t)
	sb, okB := fb.Sys().(*syscall.Stat_t)
	if !okA || !okB {
		return false, fmt.Errorf("无法解析命名空间信息")
	}
	return sa.Ino == sb.Ino, nil
}
