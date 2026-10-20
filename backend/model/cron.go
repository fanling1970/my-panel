package model

import "time"

// CronTask 计划任务
type CronTask struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"` // script | container | compose
	Schedule  string     `json:"schedule"` // 5 段 cron 表达式
	Shell     string     `json:"shell"`    // script 类型: sh | bash
	Script    string     `json:"script"`   // script 类型: 脚本内容
	Target    string     `json:"target"`   // container/compose 类型: 容器名或项目名
	Action    string     `json:"action"`   // container: start|stop|restart; compose: up|down|restart|pull
	Enabled   bool       `json:"enabled"`
	LastRunAt *time.Time `json:"last_run_at"`
	NextRunAt *time.Time `json:"next_run_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CronLog 计划任务执行日志
type CronLog struct {
	ID         int64      `json:"id"`
	TaskID     int64      `json:"task_id"`
	TaskName   string     `json:"task_name"`
	Status     string     `json:"status"` // running | success | failed
	Output     string     `json:"output"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
