<template>
  <div class="compose-page">
    <div class="page-header">
      <h2>任务管理</h2>
      <div class="header-actions">
        <el-button type="primary" class="btn-create" @click="openCreate">创建任务</el-button>
        <el-button class="btn-outline" :loading="loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>
    <el-input v-model="search" placeholder="搜索" style="max-width:320px;margin-bottom:16px" clearable @input="page = 1">
      <template #prefix><el-icon><Search /></el-icon></template>
    </el-input>
    <el-table v-loading="loading" :data="pagedProjects" stripe style="width:100%">
      <el-table-column label="名称" min-width="150">
        <template #default="{ row }"><el-link type="primary" @click="openDetail(row)">{{ row.name }}</el-link></template>
      </el-table-column>
      <el-table-column label="描述" min-width="120">
        <template #default="{ row }"><span>{{ row.description || '-' }}</span></template>
      </el-table-column>
      <el-table-column label="端口" min-width="150">
        <template #default="{ row }">
          <template v-if="row.ports">
            <template v-for="(p, i) in row.ports.split(', ')" :key="i">
              <el-link type="primary" class="mono" :href="portUrl(p)" target="_blank" style="margin-right:6px">{{ p }}</el-link>
            </template>
          </template>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="目录" min-width="140">
        <template #default="{ row }">
          <el-button
            link
            type="primary"
            size="small"
            @click="goToProjectDir(row.name)"
          >
            打开目录
          </el-button>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.status === 'running'" type="success" size="small" effect="plain">运行中</el-tag>
          <el-tag v-else-if="row.status === 'partial'" type="warning" size="small" effect="plain">部分运行</el-tag>
          <el-tag v-else-if="row.status === 'paused'" type="warning" size="small" effect="plain">已暂停</el-tag>
          <el-tag v-else-if="row.status === 'stopped'" type="danger" size="small" effect="plain">已停止</el-tag>
          <el-tag v-else-if="row.status === 'unknown'" type="info" size="small" effect="plain">未知</el-tag>
          <el-tag v-else type="warning" size="small" effect="plain">等待部署</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180">
        <template #default="{ row }"><span>{{ row.created_at || '-' }}</span></template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <div style="display:flex;flex-direction:row;gap:4px">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="primary" size="small" @click="openLogs(row)">日志</el-button>
            <el-button link type="primary" size="small" @click="openConsole(row)">控制台</el-button>
            <el-popconfirm title="删除将同时停止并删除该项目部署的容器，确定？" @confirm="handleDeleteProject(row.name)">
              <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div style="display:flex;justify-content:flex-end;margin-top:20px">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredProjects.length"
        layout="total, sizes, prev, pager, next, jumper"
      />
    </div>

    <!-- 项目日志抽屉 -->
    <el-drawer v-model="showLogs" :title="'日志 - ' + logsProject" size="55%">
      <div style="margin-bottom:12px">
        <el-button size="small" @click="loadLogs">刷新</el-button>
      </div>
      <pre class="logs-box">{{ logsText }}</pre>
    </el-drawer>

    <!-- 项目控制台抽屉 -->
    <el-drawer v-model="showConsole" :title="'控制台 - ' + consoleProject" size="60%" @close="closeConsole">
      <div ref="consoleEl" class="console-box"></div>
    </el-drawer>

    <!-- 创建任务 drawer -->
    <el-drawer v-model="showCreate" title="创建任务" size="60%" :close-on-click-modal="false">
      <h4 style="margin-bottom:8px">名称 <span style="color:#f56c6c">*</span></h4>
      <el-input v-model="form.name" placeholder="标识 Compose 创建的容器组" style="margin-bottom:16px" />
      <h4 style="margin-bottom:8px">描述</h4>
      <el-input v-model="form.description" placeholder="请输入" style="margin-bottom:16px" />
      <h4 style="margin-bottom:12px">Yaml 来源</h4>
      <el-radio-group v-model="form.source" style="margin-bottom:12px">
        <el-radio value="text">文本 Yaml</el-radio>
      </el-radio-group>
      <div style="display:flex;gap:12px;margin-bottom:12px">
        <el-button class="btn-outline" @click="importLocal">导入本地 Compose 文件</el-button>
        <el-button class="btn-outline" @click="showRunDrawer = true">通过 docker run 命令</el-button>
      </div>
      <el-input v-model="form.yaml" type="textarea" :rows="12" placeholder="services:&#10;  xxx:&#10;    image: xxx" style="font-family:monospace" @input="autoName" />
      <template #footer>
        <el-button class="btn-outline" @click="showCreate = false">取消</el-button>
        <el-button class="btn-primary-solid" @click="handleCreate">确定</el-button>
      </template>
    </el-drawer>

    <!-- docker run drawer -->
    <el-drawer v-model="showRunDrawer" title="docker run 命令" size="60%" :close-on-click-modal="false">
      <el-input v-model="runCmd" type="textarea" :rows="15" placeholder="请输入 docker run ... 命令" style="font-family:monospace" />
      <template #footer>
        <el-button class="btn-outline" @click="showRunDrawer = false">取消</el-button>
        <el-button class="btn-primary-solid" @click="convertRun">确定</el-button>
      </template>
    </el-drawer>

    <!-- 任务管理 drawer -->
    <el-drawer v-model="showDetail" :title="'任务管理 ' + currentProject" size="60%" :close-on-click-modal="false">
      <h4 class="section-title"><el-icon><Setting /></el-icon> 容器管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:24px;flex-wrap:wrap">
        <el-button class="btn-primary-solid" :loading="deploying" @click="handleDeploy">{{ isRunning ? '更新' : '启动' }}</el-button>
        <el-popconfirm title="停止并删除该项目全部容器？" @confirm="handleAction('down')">
          <template #reference>
            <el-button class="btn-danger-solid" :disabled="!hasContainers">删除容器</el-button>
          </template>
        </el-popconfirm>
        <el-button class="btn-outline" :disabled="!hasContainers" @click="handleAction('restart')">重启</el-button>
        <el-button class="btn-outline" :disabled="!isRunning && !isPaused" @click="handleAction('stop')">停止</el-button>
        <el-button v-if="isPaused" class="btn-outline" @click="handleAction('unpause')">恢复</el-button>
        <el-button v-else class="btn-outline" :disabled="!isRunning" @click="handleAction('pause')">暂停</el-button>
        <el-button class="btn-outline" :disabled="!hasContainers" @click="openLogs({ name: currentProject })">日志</el-button>
        <el-button class="btn-outline" :disabled="!isRunning" @click="openConsole({ name: currentProject })">控制台</el-button>
      </div>

      <h4 class="section-title"><el-icon><Setting /></el-icon> 任务管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:24px">
        <el-button class="btn-primary-solid" @click="openEdit({name: currentProject})">编辑任务</el-button>
        <el-button class="btn-outline" @click="openProjectDir">目录</el-button>
        <el-button class="btn-outline" @click="handleRefresh">刷新</el-button>
      </div>

      <h4 class="section-title"><el-icon><List /></el-icon> 容器列表</h4>
      <el-table :data="projectContainers" size="small" style="margin-bottom:24px">
        <el-table-column label="标识" min-width="140">
          <template #default="{ row }">{{ row.name }}</template>
        </el-table-column>
        <el-table-column label="端口" min-width="140">
          <template #default="{ row }">
            <template v-if="row.ports">
              <template v-for="(p, i) in row.ports.split(', ')" :key="i">
                <el-link type="primary" class="mono" :href="portUrl(p)" target="_blank" style="margin-right:6px">{{ p }}</el-link>
              </template>
            </template>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.state === 'running'" type="success" size="small" effect="plain">运行中</el-tag>
            <el-tag v-else-if="row.state === 'paused'" type="warning" size="small" effect="plain">已暂停</el-tag>
            <el-tag v-else type="danger" size="small" effect="plain">已停止</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110">
          <template #default="{ row }">
            <el-button link type="primary" size="small" :disabled="row.state !== 'running'" @click="openContainerConsole(row)">终端</el-button>
            <el-button link type="primary" size="small" @click="openLogs({ name: currentProject })">日志</el-button>
          </template>
        </el-table-column>
        <template #empty><el-empty description="暂无容器，请先启动项目" :image-size="60" /></template>
      </el-table>

      <h4 class="section-title"><el-icon><Plus /></el-icon> 环境变量</h4>
      <el-table :data="envVars" size="small">
        <el-table-column label="变量名" min-width="220">
          <template #default="{ row }"><el-input :model-value="row.key" disabled /></template>
        </el-table-column>
        <el-table-column label="变量值" min-width="220">
          <template #default="{ row }"><el-input :model-value="row.value" disabled /></template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <!-- 部署进度弹窗 -->
    <el-dialog v-model="showDeployDialog" title="部署容器" width="500px" :close-on-click-modal="false">
      <div style="margin-bottom:16px">
        <div style="display:flex;justify-content:space-between;margin-bottom:4px">
          <span>拉取镜像</span><span>{{ pullStatus }}</span>
        </div>
        <el-progress :percentage="pullProgress" />
      </div>
      <div style="margin-bottom:16px">
        <div style="display:flex;justify-content:space-between;margin-bottom:4px">
          <span>部署容器</span><span>{{ deployStatus }}</span>
        </div>
        <el-progress :percentage="deployProgress" />
      </div>
      <div v-if="deployDone" style="background:#f0f9eb;padding:12px;border-radius:4px;color:#67c23a">部署完成</div>
      <template #footer>
        <el-button type="primary" v-if="deployDone" @click="showDeployDialog = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Search, Setting, List, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import request from '@/utils/request'
import terminalApi from '@/api/terminal'

const router = useRouter()

const projects = ref<any[]>([])
const search = ref('')
const filteredProjects = computed(() => {
  if (!search.value) return projects.value
  return projects.value.filter(p => p.name.includes(search.value))
})
// 前端本地分页切片
const pagedProjects = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredProjects.value.slice(start, start + pageSize.value)
})
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const showCreate = ref(false)
const showDetail = ref(false)
const showRunDrawer = ref(false)
const currentProject = ref('')
// 任务管理抽屉状态
const projectStatus = ref('')
const projectContainers = ref<any[]>([])
const isRunning = computed(() => projectStatus.value === 'running' || projectStatus.value === 'partial')
const isPaused = computed(() => projectStatus.value === 'paused')
const hasContainers = computed(() => projectContainers.value.length > 0)
const envVars = computed(() => [{ key: 'COMPOSE_PROJECT_NAME', value: currentProject.value }])
const deploying = ref(false)
const showDeployDialog = ref(false)
const pullProgress = ref(0)
const deployProgress = ref(0)
const pullStatus = ref('等待中')
const deployStatus = ref('等待中')
const deployDone = ref(false)
const runCmd = ref('')
const form = ref({ name: '', description: '', source: 'text', yaml: '' })
const showLogs = ref(false)
const logsProject = ref('')
const logsText = ref('')

// 控制台终端状态
const showConsole = ref(false)
const consoleProject = ref('')
const consoleEl = ref<HTMLElement | null>(null)
let consoleTerm: Terminal | null = null
let consoleWs: WebSocket | null = null

/** 端口映射转链接：8855:80 -> http://当前主机:8855 */
function portUrl(p: string): string {
  const hostPort = p.trim().split(':')[0] ?? ''
  return `${window.location.protocol}//${window.location.hostname}:${hostPort}`
}

/** 打开项目控制台：连接项目第一个运行中容器的终端 */
async function openConsole(row: any) {
  consoleProject.value = row.name
  let containers: any[] = []
  try {
    const res: any = await request.get('/compose/status', { params: { project: row.name } })
    containers = res?.data?.containers || res?.containers || []
  } catch {}
  const target = containers.find((c: any) => c.state === 'running') || containers[0]
  if (!target) {
    ElMessage.warning('该项目暂无容器，请先启动')
    return
  }
  showConsole.value = true
  nextTick(() => initConsole(target.name))
}

function initConsole(containerName: string) {
  if (!consoleEl.value) return
  closeConsole()
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: { background: '#1e1e1e', foreground: '#d4d4d4' },
  })
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(consoleEl.value)
  fit.fit()
  consoleTerm = term

  const ws = new WebSocket(terminalApi.containerUrl(containerName))
  consoleWs = ws
  term.onData((d) => { if (ws.readyState === WebSocket.OPEN) ws.send(d) })
  term.onResize(({ cols, rows }) => {
    if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  })
  ws.onopen = () => {
    setTimeout(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }, 150)
    term.focus()
  }
  ws.onmessage = (e) => {
    if (e.data instanceof Blob) {
      e.data.text().then((t: string) => term.write(t))
    } else if (typeof e.data === 'string') {
      try {
        const m = JSON.parse(e.data)
        if (m.type === 'error') {
          term.write(`\r\n\x1b[31m${m.data}\x1b[0m\r\n`)
          return
        }
      } catch { /* 普通文本输出 */ }
      term.write(e.data)
    }
  }
}

function closeConsole() {
  consoleWs?.close()
  consoleWs = null
  consoleTerm?.dispose()
  consoleTerm = null
}

function openLogs(row: any) {
  logsProject.value = row.name
  showLogs.value = true
  loadLogs()
}

/** 拉取 compose 项目日志（后端 SSE 一次性输出，汇总 data: 行） */
async function loadLogs() {
  logsText.value = '加载中...'
  try {
    const resp = await fetch('/api/compose/action', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project: logsProject.value, action: 'logs' }),
    })
    const raw = await resp.text()
    const lines = raw
      .split('\n')
      .filter((l) => l.startsWith('data:'))
      .map((l) => l.slice(5).replace(/^ /, ''))
    logsText.value = lines.join('\n') || '（暂无日志）'
  } catch {
    logsText.value = '日志加载失败'
  }
}

function autoName() {
  const m = form.value.yaml.match(/container_name:\s*(\S+)/)
  if (m && m[1]) form.value.name = m[1]
}

function openCreate() {
  form.value = { name: '', description: '', source: 'text', yaml: '' }
  showCreate.value = true
}

async function openEdit(row: any) {
  form.value = { name: row.name, description: row.description || '', source: 'text', yaml: '' }
  try {
    const res = await request.get('/compose/file?project=' + row.name)
    form.value.yaml = res.data || ''
  } catch {}
  showCreate.value = true
}

function openDetail(row: any) {
  currentProject.value = row.name
  showDetail.value = true
  fetchProjectDetail()
}

/** 拉取项目详情：状态 + 容器列表 */
async function fetchProjectDetail() {
  try {
    const res: any = await request.get('/compose/status', { params: { project: currentProject.value } })
    const data = res?.data || res || {}
    projectStatus.value = data.status || 'unknown'
    projectContainers.value = data.containers || []
  } catch {
    projectStatus.value = 'unknown'
    projectContainers.value = []
  }
}

/** 打开项目文件目录（文件管理页） */
function openProjectDir() {
  const row = projects.value.find((p: any) => p.name === currentProject.value)
  router.push({ path: '/files', query: { path: row?.dir } })
}

/** 从列表行打开项目目录 */
function goToProjectDir(dir: string) {
  if (!dir) return
  router.push({ path: '/files', query: { path: dir } })
}

/** 打开单个容器的终端 */
function openContainerConsole(row: any) {
  consoleProject.value = row.name
  showConsole.value = true
  nextTick(() => initConsole(row.name))
}

async function handleAction(action: string) {
  try {
    const resp = await fetch('/api/compose/action', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project: currentProject.value, action })
    })
    const raw = await resp.text()
    // SSE 响应中检查错误输出
    if (raw.includes('exit status') || raw.includes('"error"')) {
      ElMessage.error('操作失败，请查看日志')
    } else {
      ElMessage.success('操作完成')
    }
    setTimeout(() => {
      fetchProjects()
      fetchProjectDetail()
    }, 1000)
  } catch {}
}

function importLocal() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.yml,.yaml'
  input.onchange = () => {
    const file = input.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => { form.value.yaml = reader.result as string }
    reader.readAsText(file)
  }
  input.click()
}

/** 把 shell 写法的主机路径转成 compose 可用的相对路径：$(pwd)/data -> ./data */
function normalizeVolume(spec: string): string {
  const parts: string[] = spec.split(':')
  if (parts.length < 2) return spec
  let src: string = parts[0] ?? ''
  // $(pwd) / ${PWD} / $PWD 在 compose 中不会被展开，等价写法是相对项目目录的 .
  if (/\$\((pwd|PWD)\)/.test(src)) {
    src = '.' + src.replace(/^\$\((pwd|PWD)\)/, '')
  } else if (/\$\{(PWD|pwd)\}/.test(src)) {
    src = '.' + src.replace(/^\$\{(PWD|pwd)\}/, '')
  } else if (/^\$PWD/.test(src)) {
    src = '.' + src.replace(/^\$PWD/, '')
  }
  if (src === '') src = '.'
  parts[0] = src
  return parts.join(':')
}

function convertRun() {
  const cmd = runCmd.value
  const name = cmd.match(/--name\s+(\S+)/)?.[1] || ''
  const image = cmd.trim().split(/\s+/).pop() || ''
  const ports: string[] = []
  const envs: string[] = []
  const volumes: string[] = []
  const restart = cmd.match(/--restart\s+(\S+)/)?.[1] || ''

  const portMatches = cmd.matchAll(/-p\s+(\S+)/g)
  for (const m of portMatches) ports.push(`"${m[1]}"`)

  const envMatches = cmd.matchAll(/-e\s+(\S+)/g)
  for (const m of envMatches) envs.push(`${m[1]}`)

  const volMatches = cmd.matchAll(/-v\s+(\S+)/g)
  for (const m of volMatches) volumes.push(`- ${normalizeVolume(m[1] ?? '')}`)

  let yaml = 'services:\n  app:\n'
  if (name) yaml += `    container_name: ${name}\n`
  if (restart) yaml += `    restart: ${restart}\n`
  yaml += `    image: ${image}\n`
  if (ports.length) {
    yaml += '    ports:\n'
    ports.forEach(p => yaml += `      - ${p}\n`)
  }
  if (envs.length) {
    yaml += '    environment:\n'
    envs.forEach(e => yaml += `      - ${e}\n`)
  }
  if (volumes.length) {
    yaml += '    volumes:\n'
    volumes.forEach(v => yaml += `      ${v}\n`)
  }
  form.value.yaml = yaml
  if (name) form.value.name = name
  showRunDrawer.value = false
  runCmd.value = ''
}

async function fetchProjects(silent = false) {
  if (!silent) loading.value = true
  try {
    const res = await request.get('/compose/list', { suppressErrorMessage: true })
    projects.value = res.data || []
  } catch {}
  if (!silent) loading.value = false
}

function handleRefresh() { fetchProjects() }

// 自动刷新：静默轮询，避免打开页面等待和刷新闪烁
let pollTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  page.value = 1
  fetchProjects()
  pollTimer = setInterval(() => fetchProjects(true), 2000)
})
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

async function handleCreate() {
  if (!form.value.name) return
  try {
    await request.post('/compose/create', { name: form.value.name, content: form.value.yaml, description: form.value.description })
    showCreate.value = false
    showDetail.value = true
    currentProject.value = form.value.name
    fetchProjects()
  } catch {}
}

async function handleDeploy() {
  showDeployDialog.value = true
  deployDone.value = false
  pullProgress.value = 0
  deployProgress.value = 0
  pullStatus.value = '拉取中...'
  deployStatus.value = '等待中...'
  deploying.value = true
  try {
    await request.post('/compose/action', { project: currentProject.value, action: 'up' })
    pullProgress.value = 100
    pullStatus.value = '完成'
    deployProgress.value = 100
    deployStatus.value = '完成'
    deployDone.value = true
    fetchProjects()
    fetchProjectDetail()
  } catch {
    pullStatus.value = '失败'
    deployStatus.value = '失败'
  }
  deploying.value = false
}

async function handleDeleteProject(name: string) {
  try {
    await request.post('/compose/delete', { project: name })
    ElMessage.success('已删除项目及其容器')
    fetchProjects()
  } catch {}
}
</script>

<style scoped>
.compose-page {
  margin: -20px;
  padding: 12px 20px 20px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.header-actions {
  display: flex;
  gap: 10px;
}
.logs-box {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 4px;
  max-height: calc(100vh - 200px);
  overflow: auto;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
}
.console-box {
  height: calc(100vh - 140px);
  background: #1e1e1e;
  border-radius: 4px;
  padding: 8px;
}
.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 15px;
}
.mono {
  font-family: monospace;
}
</style>
