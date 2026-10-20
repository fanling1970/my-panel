//go:build !windows

package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixHostProcess struct {
	cmd *exec.Cmd
	tty *os.File
}

// StartHostShell 启动宿主机 shell（PTY）
func StartHostShell(cols, rows uint16) (hostProcess, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		for _, c := range []string{"bash", "sh"} {
			if p, err := exec.LookPath(c); err == nil {
				shell = p
				break
			}
		}
	}
	if shell == "" {
		return nil, fmt.Errorf("未找到可用 shell")
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	return &unixHostProcess{cmd: cmd, tty: tty}, nil
}

func (p *unixHostProcess) Read(b []byte) (int, error)  { return p.tty.Read(b) }
func (p *unixHostProcess) Write(b []byte) (int, error) { return p.tty.Write(b) }
func (p *unixHostProcess) LocalEcho() bool             { return false }

func (p *unixHostProcess) Resize(cols, rows uint16) error {
	return pty.Setsize(p.tty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (p *unixHostProcess) Close() error {
	if p.tty != nil {
		_ = p.tty.Close()
	}
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return p.cmd.Wait()
}
