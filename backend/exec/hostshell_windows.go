//go:build windows

package exec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Windows 回退实现：无 PTY，使用管道直连 cmd.exe。
// shell 不会回显输入，由服务端 LocalEcho 补偿。

type windowsHostProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	pr    *io.PipeReader
	pw    *io.PipeWriter
}

// StartHostShell 启动宿主机 shell（Windows 回退，无 PTY）
func StartHostShell(cols, rows uint16) (hostProcess, error) {
	shell := os.Getenv("COMSPEC")
	if shell == "" {
		shell = "cmd.exe"
	}
	if _, err := exec.LookPath(shell); err != nil {
		return nil, fmt.Errorf("未找到可用 shell: %v", err)
	}

	cmd := exec.Command(shell)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &windowsHostProcess{cmd: cmd, stdin: stdin, pr: pr, pw: pw}, nil
}

func (p *windowsHostProcess) Read(b []byte) (int, error)  { return p.pr.Read(b) }
func (p *windowsHostProcess) Write(b []byte) (int, error) { return p.stdin.Write(b) }
func (p *windowsHostProcess) LocalEcho() bool             { return true }
func (p *windowsHostProcess) Resize(cols, rows uint16) error {
	return nil // 回退实现不支持动态调整大小
}

func (p *windowsHostProcess) Close() error {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.stdin.Close()
	_ = p.pr.Close()
	_ = p.pw.Close()
	return p.cmd.Wait()
}
