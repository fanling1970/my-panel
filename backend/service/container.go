package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"rabbit-panel/model"
	"rabbit-panel/repository"
)

// ContainerService 容器服务
type ContainerService struct {
	dockerRepo repository.IDockerRepository
	cacheRepo  repository.ICacheRepository
	fileRepo   repository.IFileRepository
}

// NewContainerService 创建容器服务
func NewContainerService(dr repository.IDockerRepository, cr repository.ICacheRepository, fr repository.IFileRepository) *ContainerService {
	return &ContainerService{
		dockerRepo: dr,
		cacheRepo:  cr,
		fileRepo:   fr,
	}
}

// ListContainers 列出容器
func (s *ContainerService) ListContainers(ctx context.Context) ([]model.ContainerInfo, error) {
	// Check cache
	if cached, _, ok := s.cacheRepo.GetContainers(); ok {
		return cached, nil
	}

	containers, err := s.dockerRepo.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := s.convertContainers(containers)
	s.cacheRepo.SetContainers(result)
	return result, nil
}

// ListContainersForce 强制刷新容器列表（不使用缓存）
func (s *ContainerService) ListContainersForce(ctx context.Context) ([]model.ContainerInfo, error) {
	containers, err := s.dockerRepo.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := s.convertContainers(containers)
	s.cacheRepo.SetContainers(result)
	return result, nil
}

// GetContainer 获取单个容器详情
func (s *ContainerService) GetContainer(ctx context.Context, id string) (*model.ContainerInfo, error) {
	containers, err := s.dockerRepo.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if strings.HasPrefix(c.ID, id) || c.ID == id || strings.Contains(c.ID, id) {
			result := s.convertContainers([]types.Container{c})
			return &result[0], nil
		}
	}
	return nil, fmt.Errorf("容器不存在")
}

// InspectContainer 获取容器完整配置
func (s *ContainerService) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return s.dockerRepo.ContainerInspect(ctx, id)
}

// StartContainer 启动容器
func (s *ContainerService) StartContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerStart(ctx, id, container.StartOptions{})
	s.cacheRepo.InvalidateContainers()
	return err
}

// StopContainer 停止容器
func (s *ContainerService) StopContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerStop(ctx, id, nil)
	s.cacheRepo.InvalidateContainers()
	return err
}

// RestartContainer 重启容器
func (s *ContainerService) RestartContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerRestart(ctx, id, nil)
	s.cacheRepo.InvalidateContainers()
	return err
}

// PauseContainer 暂停容器
func (s *ContainerService) PauseContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerPause(ctx, id)
	s.cacheRepo.InvalidateContainers()
	return err
}

// UnpauseContainer 恢复容器
func (s *ContainerService) UnpauseContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerUnpause(ctx, id)
	s.cacheRepo.InvalidateContainers()
	return err
}

// KillContainer 强制终止容器
func (s *ContainerService) KillContainer(ctx context.Context, id string) error {
	err := s.dockerRepo.ContainerKill(ctx, id, "SIGKILL")
	s.cacheRepo.InvalidateContainers()
	return err
}

// RemoveContainer 删除容器
func (s *ContainerService) RemoveContainer(ctx context.Context, id string, force bool) error {
	err := s.dockerRepo.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
	s.cacheRepo.InvalidateContainers()
	return err
}

// RenameContainer 重命名容器
func (s *ContainerService) RenameContainer(ctx context.Context, id, newName string) error {
	return s.dockerRepo.ContainerRename(ctx, id, newName)
}

// GetContainerLogs 获取容器日志
func (s *ContainerService) GetContainerLogs(ctx context.Context, id, tail string, follow bool) (io.ReadCloser, error) {
	tailValue := "100"
	if strings.EqualFold(strings.TrimSpace(tail), "all") {
		tailValue = "all"
	} else if strings.TrimSpace(tail) != "" {
		var tailOpt int64 = 100
		fmt.Sscanf(tail, "%d", &tailOpt)
		tailValue = fmt.Sprintf("%d", tailOpt)
	}
	return s.dockerRepo.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailValue,
		Timestamps: true,
		Follow:     follow,
	})
}

// PullImage 拉取镜像
func (s *ContainerService) PullImage(ctx context.Context, image string) error {
	reader, err := s.dockerRepo.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)
	return nil
}

// ContainerUpdateStatus 单个容器的更新状态
type ContainerUpdateStatus struct {
	ContainerID string `json:"container_id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	HasUpdate   bool   `json:"has_update"`
	Error       string `json:"error,omitempty"`
}

// CheckUpdates 遍历所有容器，对比本地与远端镜像 digest
func (s *ContainerService) CheckUpdates(ctx context.Context) ([]ContainerUpdateStatus, error) {
	containers, err := s.dockerRepo.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	result := make([]ContainerUpdateStatus, 0, len(containers))
	seenImages := map[string]bool{}
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		imageRef := c.Image
		status := ContainerUpdateStatus{ContainerID: c.ID, Name: name, Image: imageRef}

		// 同一镜像只查一次远端
		if !seenImages[imageRef] {
			seenImages[imageRef] = true
			localInspect, _, err := s.dockerRepo.ImageInspectWithRaw(ctx, imageRef)
			if err != nil {
				status.Error = err.Error()
				result = append(result, status)
				continue
			}
			remote, err := s.dockerRepo.DistributionInspect(ctx, imageRef, "")
			if err != nil {
				status.Error = err.Error()
				result = append(result, status)
				continue
			}
			// 本地 digest 列表里是否包含远端 digest
			remoteDigest := remote.Descriptor.Digest.String()
			found := false
			for _, d := range localInspect.RepoDigests {
				if strings.Contains(d, remoteDigest) {
					found = true
					break
				}
			}
			status.HasUpdate = !found
		}
		result = append(result, status)
	}
	return result, nil
}

// convertContainers 转换 Docker 容器类型到模型
func (s *ContainerService) convertContainers(containers []types.Container) []model.ContainerInfo {
	result := make([]model.ContainerInfo, 0, len(containers))
	// compose 项目元数据缓存（描述 + 端口），避免重复读文件
	type composeMeta struct {
		desc  string
		ports string
	}
	metaCache := map[string]*composeMeta{}
	// 待并行补全 host 网络端口的容器
	var pendingHostPorts []portTask
	getMeta := func(project string) *composeMeta {
		if m, ok := metaCache[project]; ok {
			return m
		}
		m := &composeMeta{}
		if s.fileRepo != nil {
			m.desc = s.fileRepo.GetComposeDescription(project)
			m.ports = parseComposePorts(s.fileRepo.GetComposeProjectDir(project))
		}
		metaCache[project] = m
		return m
	}

	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		ports := ""
		var portMappings []string
		for _, p := range c.Ports {
			if p.PublicPort != 0 {
				portMappings = append(portMappings, fmt.Sprintf("%d:%d", p.PublicPort, p.PrivatePort))
			}
		}
		if len(portMappings) > 0 {
			ports = strings.Join(portMappings, ", ")
		}

		description := ""
		project := ""
		if c.Labels != nil {
			if d, ok := c.Labels["my-panel.description"]; ok {
				description = d
			}
			project = c.Labels["com.docker.compose.project"]
		}
		// compose 项目容器：描述/端口兜底读取项目元数据（host 网络时 Ports 为空）
		if project != "" {
			m := getMeta(project)
			if description == "" {
				description = m.desc
			}
			if ports == "" {
				ports = m.ports
			}
		}
		// 端口仍为空且是 host 网络：用 ExposedPorts / 镜像元数据 / PORT 环境变量兜底显示
		if ports == "" && c.HostConfig.NetworkMode == "host" {
			pendingHostPorts = append(pendingHostPorts, portTask{index: len(result), id: c.ID})
		}

		result = append(result, model.ContainerInfo{
			ID:          c.ID[:12],
			Name:        name,
			Image:       c.Image,
			Status:      c.Status,
			Ports:       ports,
			Created:     time.Unix(c.Created, 0).Format("2006-01-02 15:04:05"),
			State:       c.State,
			Description: description,
		})
	}

	// host 网络容器的端口兜底需要额外 inspect，改为并行执行，避免串行拖慢列表
	if len(pendingHostPorts) > 0 {
		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		for _, t := range pendingHostPorts {
			wg.Add(1)
			go func(t portTask) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if p := s.hostNetworkPorts(t.id); p != "" {
					result[t.index].Ports = p
				}
			}(t)
		}
		wg.Wait()
	}
	return result
}

// portTask 需要补全端口的容器（索引对应结果位置）
type portTask struct {
	index int
	id    string
}

// hostNetworkPorts 为 host 网络的容器推断端口：
// 容器创建时固化的 ExposedPorts → 镜像 ExposedPorts → 环境变量 PORT（如 my-panel）
func (s *ContainerService) hostNetworkPorts(containerID string) string {
	ctx := context.Background()
	info, err := s.dockerRepo.ContainerInspect(ctx, containerID)
	if err != nil {
		return ""
	}
	if info.HostConfig == nil || !info.HostConfig.NetworkMode.IsHost() {
		return ""
	}

	formatPorts := func(exposed nat.PortSet) string {
		if len(exposed) == 0 {
			return ""
		}
		var eps []string
		for ep := range exposed {
			port := strings.SplitN(string(ep), "/", 2)[0]
			if port != "" {
				eps = append(eps, port+":"+port)
			}
		}
		sort.Strings(eps)
		return strings.Join(eps, ", ")
	}

	// 1. 容器自身配置
	if info.Config != nil {
		if p := formatPorts(info.Config.ExposedPorts); p != "" {
			return p
		}
		// 3. 环境变量 PORT（容器创建早于镜像 EXPOSE 声明时的兜底）
		for _, env := range info.Config.Env {
			if strings.HasPrefix(env, "PORT=") {
				if port := strings.TrimSpace(strings.TrimPrefix(env, "PORT=")); port != "" {
					return port + ":" + port
				}
			}
		}
	}
	// 2. 镜像元数据（容器配置中未固化 ExposedPorts 时）
	if img, _, err := s.dockerRepo.ImageInspectWithRaw(ctx, info.Image); err == nil && img.Config != nil {
		if p := formatPorts(img.Config.ExposedPorts); p != "" {
			return p
		}
	}
	return ""
}

// parseComposePorts 从项目 compose 文件解析端口映射（"宿主端口:容器端口"），用于 host 网络等场景
func parseComposePorts(projectDir string) string {
	data, err := os.ReadFile(resolveComposeFile(projectDir))
	if err != nil {
		return ""
	}
	var ports []string
	inPorts := false
	portRe := regexp.MustCompile(`["']?(\d+):(\d+)["']?`)
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if strings.HasPrefix(trimmed, "ports:") {
			inPorts = true
			continue
		}
		if inPorts {
			// 缩进回到 service 级别说明 ports 段结束
			if !strings.HasPrefix(trimmed, "-") && trimmed != "" && indent <= 4 {
				inPorts = false
				continue
			}
			if m := portRe.FindStringSubmatch(trimmed); m != nil {
				ports = append(ports, m[1]+":"+m[2])
			}
		}
	}
	return strings.Join(ports, ", ")
}
