package service

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"rabbit-panel/model"
	"rabbit-panel/repository"
)

// SSHTerminalService 宿主机 SSH 终端服务：配置存取 + SSH 连接
type SSHTerminalService struct {
	repo *repository.SQLiteRepository
}

// NewSSHTerminalService 创建宿主机 SSH 终端服务
func NewSSHTerminalService(repo *repository.SQLiteRepository) *SSHTerminalService {
	return &SSHTerminalService{repo: repo}
}

// GetConfig 获取已保存的连接配置，未配置时返回 nil
func (s *SSHTerminalService) GetConfig() (*model.TerminalHost, error) {
	return s.repo.GetTerminalHost()
}

// SaveConfig 保存连接配置
func (s *SSHTerminalService) SaveConfig(host string, port int, username, password string) error {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	if host == "" {
		return fmt.Errorf("主机地址不能为空")
	}
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if port <= 0 || port > 65535 {
		port = 22
	}
	return s.repo.SaveTerminalHost(host, port, username, password)
}

// TestConnection 测试 SSH 连接是否可用
func (s *SSHTerminalService) TestConnection(host string, port int, username, password string) error {
	client, err := s.dial(host, port, username, password)
	if err != nil {
		return err
	}
	_ = client.Close()
	return nil
}

// StartSession 使用已保存的配置建立 SSH 会话并申请 PTY。
// 返回会话、标准输入、标准输出以及清理函数。
func (s *SSHTerminalService) StartSession(cols, rows uint32) (*ssh.Session, io.WriteCloser, io.Reader, func(), error) {
	cfg, err := s.repo.GetTerminalHost()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("读取终端配置失败: %w", err)
	}
	if cfg == nil {
		return nil, nil, nil, nil, fmt.Errorf("尚未配置宿主机连接信息")
	}

	client, err := s.dial(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("创建 SSH 会话失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}
	if err := session.RequestPty("xterm-256color", int(rows), int(cols), modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("申请终端失败: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("打开输入管道失败: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("打开输出管道失败: %w", err)
	}

	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("启动 shell 失败: %w", err)
	}

	cleanup := func() {
		_ = session.Close()
		_ = client.Close()
	}
	return session, stdin, stdout, cleanup, nil
}

func (s *SSHTerminalService) dial(host string, port int, username, password string) (*ssh.Client, error) {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	if host == "" {
		return nil, fmt.Errorf("主机地址不能为空")
	}
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	if port <= 0 || port > 65535 {
		port = 22
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// 面板连接宿主机场景，跳过主机指纹校验
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		log.Printf("[SSHTerminal] 连接 %s:%d 失败: %v", host, port, err)
		return nil, err
	}
	return client, nil
}
