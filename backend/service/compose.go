package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"rabbit-panel/model"
	"rabbit-panel/repository"
)

// ComposeService Docker Compose 服务
type ComposeService struct {
	dockerRepo repository.IDockerRepository
	fileRepo   repository.IFileRepository
}

// NewComposeService 创建 Compose 服务
func NewComposeService(dr repository.IDockerRepository, fr repository.IFileRepository) *ComposeService {
	return &ComposeService{
		dockerRepo: dr,
		fileRepo:   fr,
	}
}

// ListProjects 列出 Compose 项目（合并 docker label 和本地目录）
func (s *ComposeService) ListProjects() ([]model.ComposeProject, error) {
	ctx := context.Background()
	projectSet := map[string]string{}

	// 从 docker 容器 label 读取
	containers, err := s.dockerRepo.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		for _, c := range containers {
			if p, ok := c.Labels["com.docker.compose.project"]; ok && p != "" {
				projectSet[p] = "running"
			}
		}
	}

	// 从本地 compose_projects 目录读取
	if localDirs, err := s.fileRepo.ListComposeProjects(); err == nil {
		for _, name := range localDirs {
			if _, exists := projectSet[name]; !exists {
				projectSet[name] = "created"
			}
		}
	}

	result := []model.ComposeProject{}
	for name := range projectSet {
		p := s.projectFrom(name, containers)
		p.Description = s.fileRepo.GetComposeDescription(name)
		p.CreatedAt = s.fileRepo.GetComposeCreatedAt(name)
		// 汇总项目端口映射
		portSet := map[string]bool{}
		ports := []string{}
		for _, c := range p.Containers {
			for _, port := range strings.Split(c.Ports, ", ") {
				port = strings.TrimSpace(port)
				if port != "" && !portSet[port] {
					portSet[port] = true
					ports = append(ports, port)
				}
			}
		}
		p.Ports = strings.Join(ports, ", ")
		if p.Ports == "" {
			// host 网络等场景下兜底解析 compose 文件端口
			p.Ports = parseComposePorts(s.fileRepo.GetComposeProjectDir(name))
		}
		// 列表不返回容器明细
		p.Containers = nil
		result = append(result, *p)
	}
	// 按项目名称英文字母排序
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

// FetchProjectStatus 获取单个 Compose 项目状态
func (s *ComposeService) FetchProjectStatus(name string) (*model.ComposeProject, error) {
	return s.projectFromContainers(name), nil
}

// CreateProject 创建 Compose 项目
func (s *ComposeService) CreateProject(name, content, description string) error {
	if err := s.fileRepo.CreateComposeProject(name, content); err != nil {
		return err
	}
	return s.fileRepo.SaveComposeDescription(name, description)
}

// GetProjectDescription 获取项目描述
func (s *ComposeService) GetProjectDescription(name string) string {
	return s.fileRepo.GetComposeDescription(name)
}

// GetProjectFile 获取项目 Compose 文件
func (s *ComposeService) GetProjectFile(name string) (string, error) {
	return s.fileRepo.GetComposeFile(name)
}

// SaveProjectFile 保存项目 Compose 文件
func (s *ComposeService) SaveProjectFile(name, content string) error {
	return s.fileRepo.SaveComposeFile(name, content)
}

// DeleteProject 删除项目：先停止并删除项目容器，再删除 compose 文件目录
func (s *ComposeService) DeleteProject(name string) error {
	// 项目目录存在时，先执行 compose down 停止并删除容器/网络
	if s.fileRepo.ComposeProjectExists(name) {
		if err := s.ExecuteAction(name, "down", io.Discard); err != nil {
			log.Printf("[Compose] 项目 %s down 失败，尝试强制删除容器: %v", name, err)
		}
	}
	// 兜底：强制删除仍带该项目 label 的容器（down 失败或目录已丢失时）
	containers, err := s.dockerRepo.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err == nil {
		for _, c := range containers {
			if c.Labels["com.docker.compose.project"] == name {
				_ = s.dockerRepo.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true})
			}
		}
	}
	return s.fileRepo.DeleteComposeProject(name)
}

// ExecuteAction 执行 Compose 操作（up, down, restart, stop, pause, unpause, pull, logs）
func (s *ComposeService) ExecuteAction(name, action string, writer io.Writer) error {
	dir := s.fileRepo.GetComposeProjectDir(name)
	composeFile := filepath.Base(resolveComposeFile(dir))
	var cmd *exec.Cmd

	switch action {
	case "up":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	case "down":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "down")
	case "restart":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "restart")
	case "stop":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "stop")
	case "pause":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "pause")
	case "unpause":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "unpause")
	case "pull":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "pull")
	case "logs":
		cmd = exec.Command("docker", "compose", "-f", composeFile, "logs", "--tail=50")
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	cmd.Dir = dir
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

// getProjectStatus 获取项目状态。新建项目还没有容器时记为已停止。
func (s *ComposeService) getProjectStatus(name string) string {
	return s.projectFromContainers(name).Status
}

func (s *ComposeService) projectFromContainers(name string) *model.ComposeProject {
	containers, err := s.dockerRepo.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return &model.ComposeProject{Name: name, Status: "unknown"}
	}
	return s.projectFrom(name, containers)
}

// projectFrom 用已获取的容器列表构建项目信息，避免列表页逐个查询 Docker（N+1 拖慢响应）
func (s *ComposeService) projectFrom(name string, containers []types.Container) *model.ComposeProject {
	project := &model.ComposeProject{
		Name:   name,
		Status: "stopped",
	}

	running := 0
	paused := 0
	// host 网络等场景下 docker 不返回端口，预先解析 compose 文件端口兜底
	fallbackPorts := ""
	for _, container := range containers {
		if container.Labels["com.docker.compose.project"] != name {
			continue
		}
		if container.State == "running" {
			running++
		}
		if container.State == "paused" {
			paused++
		}
		containerName := ""
		if len(container.Names) > 0 {
			containerName = strings.TrimPrefix(container.Names[0], "/")
		}
		ports := make([]string, 0, len(container.Ports))
		for _, port := range container.Ports {
			if port.PublicPort != 0 {
				ports = append(ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
			}
		}
		id := container.ID
		if len(id) > 12 {
			id = id[:12]
		}
		portsStr := strings.Join(ports, ", ")
		if portsStr == "" {
			if fallbackPorts == "" {
				fallbackPorts = parseComposePorts(s.fileRepo.GetComposeProjectDir(name))
			}
			portsStr = fallbackPorts
		}
		project.Containers = append(project.Containers, model.ComposeContainer{
			ID:      id,
			Name:    containerName,
			Service: container.Labels["com.docker.compose.service"],
			State:   container.State,
			Status:  container.Status,
			Ports:   portsStr,
		})
	}

	total := len(project.Containers)
	switch {
	case total == 0:
		project.Status = "created"
	case running == total:
		project.Status = "running"
	case paused == total:
		project.Status = "paused"
	case running > 0:
		project.Status = "partial"
	default:
		project.Status = "stopped"
	}
	return project
}

func resolveComposeFile(dir string) string {
	yml := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(yml); err == nil {
		return yml
	}
	yaml := filepath.Join(dir, "docker-compose.yaml")
	if _, err := os.Stat(yaml); err == nil {
		return yaml
	}
	return yml
}
