package model

import "time"

// TerminalHost 宿主机 SSH 终端连接配置
type TerminalHost struct {
	ID        int64     `json:"id"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
