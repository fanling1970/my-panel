//go:build !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Detect 检测是否具备直通宿主机的条件
func (s *HostShellService) Detect() HostShellMode {
	if _, err := exec.LookPath("nsenter"); err != nil {
		return HostShellMode{Available: false, Reason: "容器内未安装 nsenter"}
	}
	same, err := sameNamespace("/proc/1/ns/pid", "/proc/self/ns/pid")
	if err != nil {
		return HostShellMode{Available: false, Reason: "无法读取命名空间信息: " + err.Error()}
	}
	if same {
		return HostShellMode{Available: false, Reason: "容器未开启 pid: host，无法直通宿主机"}
	}
	// 再确认宿主机根文件系统可见
	if _, err := os.Stat("/proc/1/root/bin"); err != nil {
		return HostShellMode{Available: false, Reason: "无法访问宿主机文件系统"}
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
