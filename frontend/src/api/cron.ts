import request from '@/utils/request'

export interface CronTask {
  id: number
  name: string
  type: 'script' | 'container' | 'compose'
  schedule: string
  shell?: string
  script?: string
  target?: string
  action?: string
  enabled: boolean
  last_run_at?: string | null
  next_run_at?: string | null
  created_at?: string
  updated_at?: string
}

export interface CronLog {
  id: number
  task_id: number
  task_name: string
  status: 'running' | 'success' | 'failed'
  output: string
  started_at: string
  finished_at?: string | null
}

export const cronApi = {
  /** 获取全部计划任务 */
  async list(): Promise<CronTask[]> {
    const res = await request.get<CronTask[]>('/cron/tasks')
    return res.data || []
  },

  /** 创建计划任务 */
  async create(task: Partial<CronTask>): Promise<CronTask> {
    const res = await request.post<CronTask>('/cron/tasks', task)
    return res.data
  },

  /** 更新计划任务 */
  async update(task: Partial<CronTask>): Promise<CronTask> {
    const res = await request.put<CronTask>('/cron/tasks', task)
    return res.data
  },

  /** 删除计划任务 */
  async remove(id: number): Promise<void> {
    await request.post('/cron/tasks/delete', { id })
  },

  /** 立即执行一次 */
  async run(id: number): Promise<void> {
    await request.post('/cron/tasks/run', { id })
  },

  /** 查询执行日志 */
  async logs(taskId: number, limit = 100): Promise<CronLog[]> {
    const res = await request.get<CronLog[]>('/cron/logs', {
      params: { task_id: taskId, limit },
    })
    return res.data || []
  },

  /** 清空执行日志（task_id 为空时清空全部） */
  async clearLogs(taskId?: number): Promise<void> {
    await request.post('/cron/logs/clear', { task_id: taskId ?? 0 })
  },
}

export default cronApi
