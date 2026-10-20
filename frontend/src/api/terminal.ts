import { getToken } from '@/utils/request'

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
  /** 宿主机终端 WebSocket 地址 */
  hostUrl(cols = 80, rows = 24): string {
    return buildWsUrl('/api/terminal/host', { cols: String(cols), rows: String(rows) })
  },

  /** 容器终端 WebSocket 地址 */
  containerUrl(containerId: string): string {
    return buildWsUrl('/api/containers/terminal', { id: containerId })
  },
}

export default terminalApi
