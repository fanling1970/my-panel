<template>
  <div class="terminal-page">
    <div class="terminal-toolbar">
      <el-tabs
        v-if="sessions.length"
        v-model="activeId"
        type="card"
        class="terminal-tabs"
        @tab-change="onTabChange"
        @tab-remove="closeSession"
      >
        <el-tab-pane
          v-for="s in sessions"
          :key="s.id"
          :name="s.id"
          :label="s.title"
          :closable="true"
        />
      </el-tabs>
      <span v-else class="terminal-tabs-placeholder">未打开任何会话</span>

      <div class="terminal-actions">
        <el-button type="primary" size="small" :icon="Plus" @click="openHostSession">宿主机终端</el-button>
        <el-button size="small" :icon="Setting" @click="openConfigDialog">SSH 配置</el-button>
        <el-tag v-if="hostMode.available" type="success" size="small" effect="plain">直通模式</el-tag>
      </div>
    </div>

    <div class="terminal-body">
      <div
        v-for="s in sessions"
        :key="s.id"
        v-show="s.id === activeId"
        class="terminal-session"
      >
        <div :ref="(el) => setSessionEl(s, el)" class="terminal-wrapper"></div>
        <div v-if="s.status !== 'connected'" class="terminal-overlay">
          <div class="overlay-content">
            <el-icon v-if="s.status === 'connecting'" class="is-loading" :size="32"><Loading /></el-icon>
            <el-icon v-else-if="s.status === 'error'" :size="32" color="#f56c6c"><WarningFilled /></el-icon>
            <el-icon v-else :size="32"><CircleClose /></el-icon>
            <p class="status-text">{{ statusText(s) }}</p>
            <el-button
              v-if="s.status === 'disconnected' || s.status === 'error'"
              type="primary"
              size="small"
              @click="connectSession(s)"
            >
              重新连接
            </el-button>
          </div>
        </div>
      </div>

      <el-empty v-if="!sessions.length" :description="hostMode.available ? '点击右上角「宿主机终端」直接进入宿主机（直通模式，无需 SSH）' : '点击右上角「宿主机终端」打开终端（需先在 SSH 配置中填写宿主机连接信息）'" class="terminal-empty" />
    </div>

    <!-- 宿主机 SSH 连接配置 -->
    <el-dialog
      v-model="showConfig"
      title="宿主机终端连接配置"
      width="420px"
      :close-on-click-modal="false"
      @closed="onConfigClosed"
    >
      <el-form label-width="80px">
        <el-form-item label="主机地址" required>
          <el-input v-model="configForm.host" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="用户名" required>
          <el-input v-model="configForm.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="configForm.password" type="password" show-password placeholder="登录密码" />
        </el-form-item>
        <el-form-item label="端口" required>
          <el-input-number v-model="configForm.port" :min="1" :max="65535" style="width:100%" />
        </el-form-item>
      </el-form>
      <el-alert v-if="configError" type="error" :closable="false" style="margin-bottom:8px">
        <template #title>
          <div>{{ configError }}</div>
          <div style="margin-top:6px;font-size:12px">
            排查建议：确认宿主机已开启 SSH 服务且端口正确；面板运行在容器内，需保证容器网络可访问宿主机
            {{ configForm.host }}:{{ configForm.port }}（可尝试宿主机局域网 IP）。
            若容器已开启 pid: host + privileged，可直接使用「宿主机终端」的直通模式，无需 SSH。
          </div>
        </template>
      </el-alert>
      <template #footer>
        <el-button @click="showConfig = false">取消</el-button>
        <el-button :loading="testing" @click="testConnection">连接测试</el-button>
        <el-button type="primary" :loading="saving" @click="saveAndConnect">保存并连接</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Setting, Loading, CircleClose, WarningFilled } from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import terminalApi, { type SSHHostConfig, type HostShellMode } from '@/api/terminal'

type SessionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

interface Session {
  id: string
  title: string
  /** 连接方式：direct=nsenter 直通，ssh=SSH */
  mode: 'direct' | 'ssh'
  status: SessionStatus
  errorMessage?: string
  el: HTMLElement | null
  term: Terminal | null
  fitAddon: FitAddon | null
  ws: WebSocket | null
  reconnectTimer?: number
}

let sessionSeq = 0

const sessions = ref<Session[]>([])
const activeId = ref('')
const hostConfig = ref<SSHHostConfig | null>(null)
/** 宿主机直通模式（pid: host + nsenter 可用时为 true） */
const hostMode = ref<HostShellMode>({ available: false })

const showConfig = ref(false)
const testing = ref(false)
const saving = ref(false)
const configError = ref('')
/** 保存后是否自动打开终端 */
let connectAfterSave = false

const configForm = reactive({
  host: '127.0.0.1',
  port: 22,
  username: 'root',
  password: '',
})

const statusText = (s: Session) => {
  switch (s.status) {
    case 'connecting': return '正在连接...'
    case 'disconnected': return '连接已断开'
    case 'error': return s.errorMessage || '连接出错'
    default: return ''
  }
}

function setSessionEl(s: Session, el: any) {
  s.el = el instanceof HTMLElement ? el : null
}

/** 点击「宿主机终端」：优先直通模式，其次已配置的 SSH，都没有则弹配置框 */
function openHostSession() {
  if (hostMode.value.available) {
    createHostSession('direct')
    return
  }
  if (hostConfig.value) {
    createHostSession('ssh')
    return
  }
  connectAfterSave = true
  openConfigDialog()
}

function createHostSession(mode: 'direct' | 'ssh') {
  // 同一宿主机只保留一个会话
  const existing = sessions.value[0]
  if (existing) {
    activeId.value = existing.id
    onTabChange(existing.id)
    return
  }
  const id = `t${++sessionSeq}`
  sessions.value.push({
    id,
    title: mode === 'direct' ? '宿主机（直通）' : '宿主机',
    mode,
    status: 'connecting',
    el: null,
    term: null,
    fitAddon: null,
    ws: null,
  })
  activeId.value = id
  nextTick(() => {
    const s = sessions.value.find((x) => x.id === id)
    if (!s) return
    initTerminal(s)
    connectSession(s)
  })
}

function openConfigDialog() {
  const cfg = hostConfig.value
  if (cfg) {
    configForm.host = cfg.host
    configForm.port = cfg.port || 22
    configForm.username = cfg.username
    configForm.password = cfg.password || ''
  }
  showConfig.value = true
}

function onConfigClosed() {
  connectAfterSave = false
}

function validateConfigForm(): boolean {
  if (!configForm.host.trim()) { ElMessage.warning('请输入主机地址'); return false }
  if (!configForm.username.trim()) { ElMessage.warning('请输入用户名'); return false }
  if (!configForm.port || configForm.port < 1 || configForm.port > 65535) { ElMessage.warning('请输入有效端口'); return false }
  return true
}

/** 从 axios 错误中提取后端返回的错误信息 */
function extractError(e: any): string {
  return e?.response?.data?.error || e?.message || '连接失败'
}

async function testConnection() {
  if (!validateConfigForm()) return
  testing.value = true
  configError.value = ''
  try {
    await terminalApi.testConfig({ ...configForm })
    ElMessage.success('连接成功')
  } catch (e: any) {
    configError.value = extractError(e)
  } finally {
    testing.value = false
  }
}

async function saveAndConnect() {
  if (!validateConfigForm()) return
  saving.value = true
  configError.value = ''
  try {
    await terminalApi.saveConfig({ ...configForm })
    hostConfig.value = { ...configForm }
    showConfig.value = false
    ElMessage.success('配置已保存')
    if (connectAfterSave) {
      createHostSession('ssh')
    }
  } catch (e: any) {
    configError.value = extractError(e)
  } finally {
    saving.value = false
  }
}

function initTerminal(s: Session) {
  if (!s.el || s.term) return

  const term = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: terminalTheme,
    allowProposedApi: true,
  })
  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())
  term.open(s.el)
  fitAddon.fit()

  term.onData((data) => {
    if (s.ws && s.ws.readyState === WebSocket.OPEN) {
      s.ws.send(data)
    }
  })

  term.onResize(({ cols, rows }) => {
    if (s.ws && s.ws.readyState === WebSocket.OPEN) {
      s.ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
  })

  s.term = term
  s.fitAddon = fitAddon
}

function connectSession(s: Session) {
  if (!s.term) return
  disconnect(s)
  s.status = 'connecting'
  s.errorMessage = ''
  s.term.clear()

  const cols = s.term.cols
  const rows = s.term.rows
  const ws = new WebSocket(s.mode === 'direct' ? terminalApi.hostDirectUrl(cols, rows) : terminalApi.sshUrl(cols, rows))
  s.ws = ws

  ws.onopen = () => {
    s.status = 'connected'
    s.term?.focus()
    // 延迟发送 resize，等待后端完成会话初始化
    setTimeout(() => {
      if (s.term && s.ws && s.ws.readyState === WebSocket.OPEN) {
        s.ws.send(JSON.stringify({ type: 'resize', cols: s.term.cols, rows: s.term.rows }))
      }
    }, 150)
  }

  ws.onmessage = (event) => {
    if (event.data instanceof Blob) {
      event.data.text().then((text) => s.term?.write(text))
    } else if (typeof event.data === 'string') {
      try {
        const message = JSON.parse(event.data)
        if (message.type === 'error') {
          s.errorMessage = message.data || '连接出错'
          s.status = 'error'
          return
        }
      } catch {
        // 普通文本输出
      }
      s.term?.write(event.data)
    }
  }

  ws.onerror = () => {
    s.errorMessage = '连接出错，请确认面板服务正常'
    s.status = 'error'
  }

  ws.onclose = (event) => {
    if (s.status === 'connected') {
      s.status = 'disconnected'
      if (!event.wasClean) s.errorMessage = '连接已断开'
    }
  }
}

function disconnect(s: Session) {
  if (s.ws) {
    s.ws.close()
    s.ws = null
  }
  if (s.reconnectTimer) {
    clearTimeout(s.reconnectTimer)
    s.reconnectTimer = undefined
  }
}

function closeSession(id: string | number) {
  const idx = sessions.value.findIndex((s) => s.id === String(id))
  if (idx < 0) return
  const [s] = sessions.value.splice(idx, 1)
  if (!s) return
  disconnect(s)
  s.term?.dispose()
  if (activeId.value === String(id)) {
    const next = sessions.value[idx - 1] || sessions.value[idx] || null
    activeId.value = next ? next.id : ''
    if (next) nextTick(() => fitActive())
  }
}

function onTabChange(id: string | number) {
  activeId.value = String(id)
  nextTick(() => {
    fitActive()
    const s = sessions.value.find((x) => x.id === String(id))
    s?.term?.focus()
  })
}

function fitActive() {
  const s = sessions.value.find((x) => x.id === activeId.value)
  if (s?.fitAddon && s.term) {
    try {
      s.fitAddon.fit()
      if (s.ws && s.ws.readyState === WebSocket.OPEN) {
        s.ws.send(JSON.stringify({ type: 'resize', cols: s.term.cols, rows: s.term.rows }))
      }
    } catch { /* 容器隐藏时 fit 可能失败，忽略 */ }
  }
}

function onWindowResize() {
  fitActive()
}

async function loadConfig() {
  try {
    hostConfig.value = await terminalApi.getConfig()
  } catch {
    // 读取失败视为未配置
    hostConfig.value = null
  }
}

const terminalTheme = {
  background: '#1e1e1e',
  foreground: '#d4d4d4',
  cursor: '#d4d4d4',
  cursorAccent: '#1e1e1e',
  selectionBackground: '#264f78',
  black: '#000000',
  red: '#cd3131',
  green: '#0dbc79',
  yellow: '#e5e510',
  blue: '#2472c8',
  magenta: '#bc3fbc',
  cyan: '#11a8cd',
  white: '#e5e5e5',
  brightBlack: '#666666',
  brightRed: '#f14c4c',
  brightGreen: '#23d18b',
  brightYellow: '#f5f543',
  brightBlue: '#3b8eea',
  brightMagenta: '#d670d6',
  brightCyan: '#29b8db',
  brightWhite: '#e5e5e5',
}

async function loadHostMode() {
  try {
    hostMode.value = await terminalApi.getHostMode()
  } catch {
    hostMode.value = { available: false }
  }
}

onMounted(() => {
  loadConfig()
  loadHostMode()
  window.addEventListener('resize', onWindowResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onWindowResize)
  sessions.value.forEach(disconnect)
})
</script>

<style scoped>
.terminal-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px 16px 16px;
  box-sizing: border-box;
}
.terminal-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.terminal-tabs {
  flex: 1;
  min-width: 0;
}
.terminal-actions {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
}
.terminal-tabs-placeholder {
  flex: 1;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.terminal-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}
.terminal-tabs :deep(.el-tabs__item) {
  height: 34px;
  line-height: 34px;
}
.terminal-body {
  position: relative;
  flex: 1;
  min-height: 0;
  margin-top: 8px;
  background: #1e1e1e;
  border-radius: 6px;
  overflow: hidden;
}
.terminal-session {
  position: absolute;
  inset: 0;
}
.terminal-wrapper {
  width: 100%;
  height: 100%;
  padding: 8px;
  box-sizing: border-box;
}
.terminal-wrapper :deep(.xterm) {
  height: 100%;
}
.terminal-wrapper :deep(.xterm-viewport) {
  overflow-y: auto !important;
}
.terminal-overlay {
  position: absolute;
  inset: 0;
  background: rgba(30, 30, 30, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}
.overlay-content {
  text-align: center;
  color: #d4d4d4;
}
.status-text {
  margin: 14px 0;
  font-size: 14px;
}
.terminal-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.is-loading {
  animation: rotating 2s linear infinite;
}
@keyframes rotating {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
