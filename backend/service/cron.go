package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"

	"rabbit-panel/model"
	"rabbit-panel/repository"
)

// 脚本执行超时时间
const scriptTimeout = 30 * time.Minute

// 单条日志输出上限
const maxLogOutput = 64 * 1024

// CronService 计划任务服务：CRUD + 后台调度
type CronService struct {
	repo       *repository.SQLiteRepository
	dockerRepo repository.IDockerRepository
	compose    *ComposeService

	mu      sync.Mutex
	running map[int64]bool
	stopCh  chan struct{}
	once    sync.Once
}

// NewCronService 创建计划任务服务
func NewCronService(repo *repository.SQLiteRepository, dockerRepo repository.IDockerRepository, compose *ComposeService) *CronService {
	return &CronService{
		repo:       repo,
		dockerRepo: dockerRepo,
		compose:    compose,
		running:    make(map[int64]bool),
		stopCh:     make(chan struct{}),
	}
}

// Start 启动后台调度循环
func (s *CronService) Start() {
	s.once.Do(func() {
		go s.loop()
		log.Println("[Cron] 调度器已启动")
	})
}

// Stop 停止调度循环
func (s *CronService) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}

func (s *CronService) loop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	// 启动后先立即跑一次，避免错过启动期间到期的任务
	s.tick()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *CronService) tick() {
	tasks, err := s.repo.ListCronTasks()
	if err != nil {
		log.Printf("[Cron] 读取任务失败: %v", err)
		return
	}
	now := time.Now()
	for _, t := range tasks {
		if !t.Enabled || t.NextRunAt != nil {
			continue
		}
		// 手动任务没有调度计划，跳过补算
		if t.Schedule == "manual" {
			continue
		}
		// 补算缺失的下次执行时间（新建/升级遗留）
		expr, err := ParseCron(t.Schedule)
		if err != nil {
			log.Printf("[Cron] 任务 %d 表达式无效: %v", t.ID, err)
			continue
		}
		_ = s.repo.SetCronTaskNextRun(t.ID, expr.Next(now))
	}

	for _, t := range tasks {
		if !t.Enabled || t.NextRunAt == nil {
			continue
		}
		if !now.Before(*t.NextRunAt) {
			expr, err := ParseCron(t.Schedule)
			if err != nil {
				continue
			}
			s.execute(t, expr, true)
		}
	}

	s.cleanupIfNeeded()
}

var lastCleanup time.Time

// cleanupIfNeeded 每小时清理一次 7 天前的执行日志
func (s *CronService) cleanupIfNeeded() {
	if time.Since(lastCleanup) < time.Hour {
		return
	}
	lastCleanup = time.Now()
	n, err := s.repo.CleanupCronLogs(time.Now().Add(-7 * 24 * time.Hour).UTC())
	if err != nil {
		log.Printf("[Cron] 清理日志失败: %v", err)
	} else if n > 0 {
		log.Printf("[Cron] 已清理 %d 条过期执行日志", n)
	}
}

// execute 执行任务（异步）。scheduled=true 表示由调度触发，会顺延下次执行时间。
func (s *CronService) execute(t *model.CronTask, expr *cronExpr, scheduled bool) {
	s.mu.Lock()
	if s.running[t.ID] {
		s.mu.Unlock()
		log.Printf("[Cron] 任务 %s 尚在执行，跳过本次触发", t.Name)
		return
	}
	s.running[t.ID] = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, t.ID)
			s.mu.Unlock()
		}()

		startedAt := time.Now()
		logID, _ := s.repo.CreateCronLog(&model.CronLog{
			TaskID:    t.ID,
			TaskName:  t.Name,
			Status:    "running",
			StartedAt: startedAt,
		})

		status, output := s.run(t)

		finishedAt := time.Now()
		if logID > 0 {
			_ = s.repo.FinishCronLog(logID, status, output, finishedAt)
		}
		if scheduled {
			_ = s.repo.UpdateCronTaskRunState(t.ID, startedAt, expr.Next(finishedAt))
		} else {
			// 手动执行：只记上次执行时间，不影响下次执行计划
			_ = s.repo.UpdateCronTaskLastRun(t.ID, startedAt)
		}
		log.Printf("[Cron] 任务 %s 执行%s", t.Name, map[string]string{"success": "成功", "failed": "失败"}[status])
	}()
}

// run 同步执行一次任务，返回状态与输出
func (s *CronService) run(t *model.CronTask) (string, string) {
	switch t.Type {
	case "container":
		if err := s.runContainerAction(t.Target, t.Action); err != nil {
			return "failed", err.Error()
		}
		return "success", fmt.Sprintf("容器 %s %s 成功", t.Target, t.Action)
	case "compose":
		var buf bytes.Buffer
		if err := s.compose.ExecuteAction(t.Target, t.Action, &buf); err != nil {
			return "failed", truncateOutput(buf.String() + "\n" + err.Error())
		}
		return "success", truncateOutput(buf.String())
	default: // script
		shell := t.Shell
		if shell != "bash" {
			shell = "sh"
		}
		ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, shell, "-c", t.Script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "failed", truncateOutput(string(out) + "\n" + err.Error())
		}
		return "success", truncateOutput(string(out))
	}
}

func (s *CronService) runContainerAction(name, action string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	timeout := 10
	switch action {
	case "start":
		return s.dockerRepo.ContainerStart(ctx, name, container.StartOptions{})
	case "stop":
		return s.dockerRepo.ContainerStop(ctx, name, &timeout)
	case "restart":
		return s.dockerRepo.ContainerRestart(ctx, name, &timeout)
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}
}

// === 对外 CRUD ===

// ListTasks 获取全部任务
func (s *CronService) ListTasks() ([]*model.CronTask, error) {
	return s.repo.ListCronTasks()
}

// GetTask 获取单个任务
func (s *CronService) GetTask(id int64) (*model.CronTask, error) {
	return s.repo.GetCronTask(id)
}

// CreateTask 创建任务
func (s *CronService) CreateTask(t *model.CronTask) (int64, error) {
	if err := s.validateTask(t); err != nil {
		return 0, err
	}
	if t.Schedule == "manual" {
		// 手动任务：不设下次执行时间
		t.NextRunAt = nil
	} else {
		expr, _ := ParseCron(t.Schedule)
		next := expr.Next(time.Now())
		t.NextRunAt = &next
	}
	return s.repo.CreateCronTask(t)
}

// UpdateTask 更新任务
func (s *CronService) UpdateTask(t *model.CronTask) error {
	if t.ID <= 0 {
		return fmt.Errorf("无效的任务 ID")
	}
	if err := s.validateTask(t); err != nil {
		return err
	}
	if t.Schedule == "manual" {
		t.NextRunAt = nil
	} else {
		expr, _ := ParseCron(t.Schedule)
		next := expr.Next(time.Now())
		t.NextRunAt = &next
	}
	return s.repo.UpdateCronTask(t)
}

// DeleteTask 删除任务（同时清理其日志）
func (s *CronService) DeleteTask(id int64) error {
	if err := s.repo.DeleteCronTask(id); err != nil {
		return err
	}
	return s.repo.ClearCronLogs(id)
}

// RunNow 立即执行一次
func (s *CronService) RunNow(id int64) error {
	t, err := s.repo.GetCronTask(id)
	if err != nil || t == nil {
		return fmt.Errorf("任务不存在")
	}
	var expr *cronExpr
	if t.Schedule != "manual" {
		e, err := ParseCron(t.Schedule)
		if err != nil {
			return err
		}
		expr = e
	}
	s.execute(t, expr, false)
	return nil
}

// TaskLogs 查询执行日志
func (s *CronService) TaskLogs(taskID int64, limit int) ([]*model.CronLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListCronLogs(taskID, limit)
}

// ClearLogs 清空执行日志
func (s *CronService) ClearLogs(taskID int64) error {
	return s.repo.ClearCronLogs(taskID)
}

func (s *CronService) validateTask(t *model.CronTask) error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return fmt.Errorf("任务名称不能为空")
	}
	t.Schedule = strings.TrimSpace(t.Schedule)
	if t.Schedule == "manual" {
		// 手动任务：仅通过「执行」触发，不校验 cron 表达式
	} else if _, err := ParseCron(t.Schedule); err != nil {
		return fmt.Errorf("无效的执行计划: %v", err)
	}
	t.Type = strings.TrimSpace(t.Type)
	switch t.Type {
	case "script":
		if strings.TrimSpace(t.Script) == "" {
			return fmt.Errorf("脚本内容不能为空")
		}
	case "container":
		if strings.TrimSpace(t.Target) == "" {
			return fmt.Errorf("请选择目标容器")
		}
		if t.Action != "start" && t.Action != "stop" && t.Action != "restart" {
			return fmt.Errorf("不支持的容器操作: %s", t.Action)
		}
	case "compose":
		if strings.TrimSpace(t.Target) == "" {
			return fmt.Errorf("请选择 Compose 项目")
		}
		switch t.Action {
		case "up", "down", "restart", "pull":
		default:
			return fmt.Errorf("不支持的 Compose 操作: %s", t.Action)
		}
	default:
		return fmt.Errorf("不支持的任务类型: %s", t.Type)
	}
	return nil
}

func truncateOutput(s string) string {
	if len(s) <= maxLogOutput {
		return s
	}
	return s[:maxLogOutput] + "\n...（输出过长已截断）"
}

// === Cron 表达式解析 ===

// cronExpr 解析后的 5 段 cron 表达式（分 时 日 月 周）
type cronExpr struct {
	minute map[int]bool
	hour   map[int]bool
	dom    map[int]bool
	month  map[int]bool
	dow    map[int]bool

	anyDom bool
	anyDow bool
}

// ParseCron 解析并校验 cron 表达式，支持 @hourly/@daily/@weekly/@monthly 快捷方式
func ParseCron(expr string) (*cronExpr, error) {
	switch strings.ToLower(strings.TrimSpace(expr)) {
	case "@hourly":
		expr = "0 * * * *"
	case "@daily", "@midnight":
		expr = "0 0 * * *"
	case "@weekly":
		expr = "0 0 * * 0"
	case "@monthly":
		expr = "0 0 1 * *"
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("需要 5 个字段（分 时 日 月 周），收到 %d 个", len(fields))
	}

	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("分钟字段: %v", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("小时字段: %v", err)
	}
	dom, domAny, err := parseCronFieldAny(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("日字段: %v", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("月字段: %v", err)
	}
	dow, dowAny, err := parseCronFieldAny(fields[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("周字段: %v", err)
	}

	// 规范化周字段：7 视为 0（周日）
	if dow[7] {
		dow[0] = true
		delete(dow, 7)
	}

	return &cronExpr{
		minute: minute,
		hour:   hour,
		dom:    dom,
		month:  month,
		dow:    dow,
		anyDom: domAny,
		anyDow: dowAny,
	}, nil
}

// Next 计算从 from 之后（不含 from）的下一次执行时间
func (e *cronExpr) Next(from time.Time) time.Time {
	t := from.Add(time.Minute).Truncate(time.Minute)
	limit := from.AddDate(5, 0, 0)

	for t.Before(limit) {
		if !e.month[int(t.Month())] {
			// 跳到下月 1 号 0 点
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).AddDate(0, 1, 0)
			continue
		}
		if !e.dayMatch(t) {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
			continue
		}
		if e.hour[t.Hour()] && e.minute[t.Minute()] {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}
}

// Match 判断 t 是否命中
func (e *cronExpr) Match(t time.Time) bool {
	return e.month[int(t.Month())] && e.dayMatch(t) && e.hour[t.Hour()] && e.minute[t.Minute()]
}

// dayMatch 标准cron规则：日与周都受限时二者满足其一即可
func (e *cronExpr) dayMatch(t time.Time) bool {
	domOK := e.anyDom || e.dom[t.Day()]
	dowOK := e.anyDow || e.dow[int(t.Weekday())]
	if !e.anyDom && !e.anyDow {
		return domOK || dowOK
	}
	return domOK && dowOK
}

func parseCronField(field string, min, max int) (map[int]bool, error) {
	m, _, err := parseCronFieldAny(field, min, max)
	return m, err
}

// parseCronFieldAny 解析单个字段，返回取值集合及是否为 * 
func parseCronFieldAny(field string, min, max int) (map[int]bool, bool, error) {
	result := make(map[int]bool)
	any := false
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false, fmt.Errorf("空取值")
		}
		// 步长
		step := 1
		if idx := strings.Index(part, "/"); idx >= 0 {
			v, err := strconv.Atoi(part[idx+1:])
			if err != nil || v <= 0 {
				return nil, false, fmt.Errorf("无效步长 %q", part[idx+1:])
			}
			step = v
			part = part[:idx]
		}
		// 范围
		lo, hi := min, max
		if part != "*" {
			if idx := strings.Index(part, "-"); idx >= 0 {
				a, err1 := strconv.Atoi(part[:idx])
				b, err2 := strconv.Atoi(part[idx+1:])
				if err1 != nil || err2 != nil {
					return nil, false, fmt.Errorf("无效范围 %q", part)
				}
				lo, hi = a, b
			} else {
				v, err := strconv.Atoi(part)
				if err != nil {
					return nil, false, fmt.Errorf("无效数值 %q", part)
				}
				lo, hi = v, v
			}
		} else if step > 1 {
			hi = max
		}
		if lo < min || hi > max || lo > hi {
			return nil, false, fmt.Errorf("取值 %q 超出范围 [%d,%d]", part, min, max)
		}
		if part == "*" && step == 1 {
			any = true
		}
		for v := lo; v <= hi; v += step {
			result[v] = true
		}
	}
	return result, any, nil
}
