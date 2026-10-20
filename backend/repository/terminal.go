package repository

import (
	"database/sql"
	"errors"
	"time"

	"rabbit-panel/model"
)

// === Terminal Host（宿主机 SSH 配置）===

// GetTerminalHost 获取宿主机 SSH 连接配置（单条），无配置时返回 nil
func (r *SQLiteRepository) GetTerminalHost() (*model.TerminalHost, error) {
	row := r.db.QueryRow(
		`SELECT id, host, port, username, password, created_at, updated_at FROM terminal_hosts ORDER BY id ASC LIMIT 1`)
	var h model.TerminalHost
	if err := row.Scan(&h.ID, &h.Host, &h.Port, &h.Username, &h.Password, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}

// SaveTerminalHost 保存（新增或更新）宿主机 SSH 连接配置
func (r *SQLiteRepository) SaveTerminalHost(host string, port int, username, password string) error {
	existing, err := r.GetTerminalHost()
	if err != nil {
		return err
	}
	if existing != nil {
		_, err = r.db.Exec(
			`UPDATE terminal_hosts SET host = ?, port = ?, username = ?, password = ?, updated_at = ? WHERE id = ?`,
			host, port, username, password, time.Now(), existing.ID)
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO terminal_hosts (host, port, username, password, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		host, port, username, password, time.Now(), time.Now())
	return err
}
