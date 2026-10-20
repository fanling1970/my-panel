package repository

import (
	"time"

	"rabbit-panel/model"
)

// === Cron Task Operations ===

// scanCronTask 从行扫描计划任务记录
func scanCronTask(scan func(dest ...any) error) (*model.CronTask, error) {
	var t model.CronTask
	var enabled int
	var lastRun, nextRun any
	if err := scan(&t.ID, &t.Name, &t.Type, &t.Schedule, &t.Shell, &t.Script, &t.Target, &t.Action, &enabled,
		&lastRun, &nextRun, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	if v, ok := lastRun.(time.Time); ok {
		t.LastRunAt = &v
	}
	if v, ok := nextRun.(time.Time); ok {
		t.NextRunAt = &v
	}
	return &t, nil
}

const cronTaskColumns = `id, name, type, schedule, shell, script, target, action, enabled, last_run_at, next_run_at, created_at, updated_at`

// ListCronTasks 获取所有计划任务
func (r *SQLiteRepository) ListCronTasks() ([]*model.CronTask, error) {
	rows, err := r.db.Query("SELECT " + cronTaskColumns + " FROM cron_tasks ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.CronTask
	for rows.Next() {
		t, err := scanCronTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetCronTask 根据 ID 获取计划任务
func (r *SQLiteRepository) GetCronTask(id int64) (*model.CronTask, error) {
	row := r.db.QueryRow("SELECT "+cronTaskColumns+" FROM cron_tasks WHERE id = ?", id)
	t, err := scanCronTask(row.Scan)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateCronTask 创建计划任务
func (r *SQLiteRepository) CreateCronTask(t *model.CronTask) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO cron_tasks (name, type, schedule, shell, script, target, action, enabled, next_run_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Type, t.Schedule, t.Shell, t.Script, t.Target, t.Action, boolToInt(t.Enabled), t.NextRunAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateCronTask 更新计划任务
func (r *SQLiteRepository) UpdateCronTask(t *model.CronTask) error {
	_, err := r.db.Exec(
		`UPDATE cron_tasks SET name = ?, type = ?, schedule = ?, shell = ?, script = ?, target = ?, action = ?,
		 enabled = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		t.Name, t.Type, t.Schedule, t.Shell, t.Script, t.Target, t.Action, boolToInt(t.Enabled), t.NextRunAt, t.ID,
	)
	return err
}

// SetCronTaskNextRun 仅更新下次执行时间
func (r *SQLiteRepository) SetCronTaskNextRun(id int64, next time.Time) error {
	_, err := r.db.Exec(
		`UPDATE cron_tasks SET next_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		next, id,
	)
	return err
}

// UpdateCronTaskRunState 更新任务执行时间
func (r *SQLiteRepository) UpdateCronTaskRunState(id int64, lastRun, nextRun time.Time) error {
	_, err := r.db.Exec(
		`UPDATE cron_tasks SET last_run_at = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		lastRun, nextRun, id,
	)
	return err
}

// UpdateCronTaskLastRun 仅更新上次执行时间（手动执行用）
func (r *SQLiteRepository) UpdateCronTaskLastRun(id int64, lastRun time.Time) error {
	_, err := r.db.Exec(
		`UPDATE cron_tasks SET last_run_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		lastRun, id,
	)
	return err
}

// DeleteCronTask 删除计划任务
func (r *SQLiteRepository) DeleteCronTask(id int64) error {
	_, err := r.db.Exec("DELETE FROM cron_tasks WHERE id = ?", id)
	return err
}

// === Cron Log Operations ===

// CreateCronLog 创建执行日志，返回日志 ID
func (r *SQLiteRepository) CreateCronLog(log *model.CronLog) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO cron_logs (task_id, task_name, status, output, started_at) VALUES (?, ?, ?, ?, ?)`,
		log.TaskID, log.TaskName, log.Status, log.Output, log.StartedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishCronLog 更新执行日志结果
func (r *SQLiteRepository) FinishCronLog(id int64, status, output string, finishedAt time.Time) error {
	_, err := r.db.Exec(
		`UPDATE cron_logs SET status = ?, output = ?, finished_at = ? WHERE id = ?`,
		status, output, finishedAt, id,
	)
	return err
}

// ListCronLogs 查询执行日志
func (r *SQLiteRepository) ListCronLogs(taskID int64, limit int) ([]*model.CronLog, error) {
	query := `SELECT id, task_id, task_name, status, output, started_at, finished_at FROM cron_logs`
	var args []any
	if taskID > 0 {
		query += " WHERE task_id = ?"
		args = append(args, taskID)
	}
	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.CronLog
	for rows.Next() {
		var l model.CronLog
		var finished any
		if err := rows.Scan(&l.ID, &l.TaskID, &l.TaskName, &l.Status, &l.Output, &l.StartedAt, &finished); err != nil {
			return nil, err
		}
		if v, ok := finished.(time.Time); ok {
			l.FinishedAt = &v
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

// ClearCronLogs 清空执行日志
func (r *SQLiteRepository) ClearCronLogs(taskID int64) error {
	if taskID > 0 {
		_, err := r.db.Exec("DELETE FROM cron_logs WHERE task_id = ?", taskID)
		return err
	}
	_, err := r.db.Exec("DELETE FROM cron_logs")
	return err
}

// CleanupCronLogs 清理过期日志（保留 recent 以内的记录）
func (r *SQLiteRepository) CleanupCronLogs(olderThan time.Time) (int64, error) {
	res, err := r.db.Exec("DELETE FROM cron_logs WHERE started_at < ?", olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
