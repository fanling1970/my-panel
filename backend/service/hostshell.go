package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// HostShellService 宿主机直通终端服务：
// 当容器以 pid: host + privileged 方式运行时，可直接用 nsenter 进入宿主机命名空间，
// 无需 SSH 账号密码即可获得宿主机 shell。
type HostShellService struct{}

// NewHostShellService 创建宿主机直通终端服务
func NewHostShellService() *HostShellService {
	return &HostShellService{}
}

// HostShellMode 直通模式可用性
type HostShellMode struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// StartSession 启动宿主机直通终端，返回终端主设备、子进程和清理函数
func (s *HostShellService) StartSession(cols, rows uint32) (*os.File, *exec.Cmd, func(), error) {
	nsenter, err := exec.LookPath("nsenter")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("容器内未安装 nsenter")
	}
	if cols == 0 || cols > 500 {
		cols = 80
	}
	if rows == 0 || rows > 500 {
		rows = 24
	}

	// 进入宿主机的 mount/uts/ipc/net/pid 命名空间（不切换 user 命名空间）
	cmd := exec.Command(nsenter,
		"-t", "1",
		"--mount", "--uts", "--ipc", "--net", "--pid", "--wd",
		"/bin/sh",
	)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Dir = "/"

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("启动宿主机 shell 失败: %w", err)
	}

	cleanup := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = ptmx.Close()
		_, _ = cmd.Process.Wait()
		if cmd.Process != nil {
			log.Printf("[HostShell] 会话结束 pid=%d", cmd.Process.Pid)
		}
	}
	return ptmx, cmd, cleanup, nil
}

// Resize 调整直通终端大小
func (s *HostShellService) Resize(ptmx *os.File, cols, rows uint32) error {
	if cols == 0 || rows == 0 || cols > 500 || rows > 500 {
		return nil
	}
	return pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}
