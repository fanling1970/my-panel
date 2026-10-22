import request from '@/utils/request'
import { getToken } from '@/utils/request'

/** 宿主机直通模式可用性 */
export interface HostShellMode {
  available: boolean
  reason?: string
}

/** 宿主机 SSH 连接配置 */
export interface SSHHostConfig {
  id?: number
  host: string
  port: number
  username: string
  password: string
}

/**
 * 构建 WebSocket 地址（自动拼接 token）
 */
function buildWsUrl(path: string, params: Record<string, string> = {}): string {
  const token = getToken()
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const query = new URLSearchParams({ ...params, token: token || '' })
  return `${protocol}//${window.location.host}${path}?${query.toString()}`
}

export const terminalApi = {
  /** 宿主机直通终端 WebSocket 地址（nsenter，无需 SSH） */
  hostDirectUrl(cols = 80, rows = 24): string {
    return buildWsUrl('/api/terminal/host-direct', { cols: String(cols), rows: String(rows) })
  },

  /** 检测宿主机直通模式是否可用 */
  async getHostMode(): Promise<HostShellMode> {
    const res = await request.get<HostShellMode>('/terminal/host-mode', { suppressErrorMessage: true })
    return res.data ?? { available: false }
  },

  /** 宿主机 SSH 终端 WebSocket 地址 */
  sshUrl(cols = 80, rows = 24): string {
    return buildWsUrl('/api/terminal/ssh', { cols: String(cols), rows: String(rows) })
  },

  /** 容器终端 WebSocket 地址 */
  containerUrl(containerId: string): string {
    return buildWsUrl('/api/containers/terminal', { id: containerId })
  },

  /** 获取宿主机 SSH 连接配置，未配置时返回 null */
  async getConfig(): Promise<SSHHostConfig | null> {
    const res = await request.get<SSHHostConfig | null>('/terminal/ssh-config')
    return res.data ?? null
  },

  /** 保存宿主机 SSH 连接配置 */
  async saveConfig(cfg: SSHHostConfig): Promise<void> {
    await request.post('/terminal/ssh-config', cfg, { suppressErrorMessage: true })
  },

  /** 测试 SSH 连接 */
  async testConfig(cfg: SSHHostConfig): Promise<void> {
    await request.post('/terminal/ssh-test', cfg, { suppressErrorMessage: true })
  },
}

export default terminalApi
