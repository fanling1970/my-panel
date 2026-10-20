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

      <el-dropdown trigger="click" popper-class="terminal-session-dropdown" @command="handleNewSession">
        <el-button type="primary" size="small" :icon="Plus">新建会话</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="__host__">宿主机终端</el-dropdown-item>
            <el-dropdown-item divided disabled>容器终端</el-dropdown-item>
            <el-dropdown-item v-for="c in containers" :key="c.value" :command="c.value" :title="c.value">
              {{ c.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
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

      <el-empty v-if="!sessions.length" description="点击右上角「新建会话」打开终端" class="terminal-empty" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Loading, CircleClose, WarningFilled } from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import terminalApi from '@/api/terminal'
import containerApi from '@/api/containers'

type SessionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

interface Session {
  id: string
  title: string
  containerId?: string
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
const containers = ref<{ label: string; value: string }[]>([])

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

function newSession(command: string) {
  const isHost = command === '__host__'
  // 同一容器/宿主机只保留一个会话
  const existing = sessions.value.find((s) => (isHost ? !s.containerId : s.containerId === command))
  if (existing) {
    activeId.value = existing.id
    onTabChange(existing.id)
    return
  }

  const id = `t${++sessionSeq}`
  sessions.value.push({
    id,
    title: isHost ? '宿主机' : command,
    containerId: isHost ? undefined : command,
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

function handleNewSession(command: string) {
  newSession(command)
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
  const url = s.containerId
    ? terminalApi.containerUrl(s.containerId)
    : terminalApi.hostUrl(cols, rows)
  const ws = new WebSocket(url)
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

async function loadContainers() {
  try {
    const res: any = await containerApi.list()
    const arr = Array.isArray(res) ? res : res?.containers || res?.data || []
    containers.value = arr
      .map((c: any) => {
        const name = c.Names?.[0]?.replace('/', '') || c.Name || c.name || ''
        const state = c.State || c.state || ''
        return { label: state ? `${name}（${state}）` : name, value: name }
      })
      .filter((c: { value: string }) => c.value)
  } catch {
    ElMessage.warning('容器列表加载失败')
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

onMounted(() => {
  loadContainers()
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

<style>
/* 下拉菜单渲染在组件树外，需全局样式 */
.terminal-session-dropdown {
  max-height: 320px;
  overflow-y: auto;
}
</style>
