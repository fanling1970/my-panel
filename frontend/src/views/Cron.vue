<template>
  <div class="cron-page">
    <div class="page-header">
      <h2>计划任务</h2>
      <div class="header-actions">
        <el-button type="primary" @click="openCreate">创建计划任务</el-button>
      </div>
    </div>

    <el-table :data="tasks" stripe style="width:100%" v-loading="loading">
      <el-table-column label="任务名称" min-width="160">
        <template #default="{ row }">
          <el-link type="primary" @click="openLogs(row)">{{ row.name }}</el-link>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="typeTagMap[row.type]?.tag || 'info'">
            {{ typeTagMap[row.type]?.label || row.type }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="目标" min-width="150">
        <template #default="{ row }">
          <span v-if="row.type === 'script'" class="muted">{{ scriptBrief(row.script) }}</span>
          <span v-else>{{ row.target }}（{{ actionLabel(row.type, row.action) }}）</span>
        </template>
      </el-table-column>
      <el-table-column label="执行计划" min-width="150">
        <template #default="{ row }">
          <span class="mono">{{ row.schedule }}</span>
          <span class="muted" v-if="scheduleHint(row.schedule)">（{{ scheduleHint(row.schedule) }}）</span>
        </template>
      </el-table-column>
      <el-table-column label="上次执行" width="160">
        <template #default="{ row }">{{ formatTime(row.last_run_at) }}</template>
      </el-table-column>
      <el-table-column label="下次执行" width="160">
        <template #default="{ row }">{{ formatTime(row.next_run_at) }}</template>
      </el-table-column>
      <el-table-column label="启用" width="80">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" :loading="row._toggling" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="230" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" :loading="row._running" @click="runNow(row)">执行</el-button>
          <el-button link type="primary" size="small" @click="openLogs(row)">记录</el-button>
          <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
          <el-popconfirm title="删除后不可恢复，确定？" @confirm="removeTask(row)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无计划任务，点击右上角创建" />
      </template>
    </el-table>

    <!-- 创建 / 编辑抽屉 -->
    <el-drawer v-model="showForm" :title="editingId ? '编辑计划任务' : '创建计划任务'" size="600px" :close-on-click-modal="false">
      <el-form label-position="top">
        <el-form-item label="任务名称" required>
          <el-input v-model="form.name" placeholder="例如: 每天备份数据库" maxlength="50" show-word-limit />
        </el-form-item>

        <el-form-item label="任务类型" required>
          <el-radio-group v-model="form.type">
            <el-radio-button value="script">脚本</el-radio-button>
            <el-radio-button value="container">容器操作</el-radio-button>
            <el-radio-button value="compose">Compose 操作</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.type === 'script'">
          <el-form-item label="Shell 环境">
            <el-select v-model="form.shell" style="width:200px">
              <el-option label="sh" value="sh" />
              <el-option label="bash" value="bash" />
            </el-select>
          </el-form-item>
          <el-form-item label="执行脚本" required>
            <el-input
              v-model="form.script"
              type="textarea"
              :rows="10"
              placeholder="#!/bin/sh&#10;echo hello"
              style="font-family:monospace"
            />
          </el-form-item>
        </template>

        <template v-else-if="form.type === 'container'">
          <el-form-item label="目标容器" required>
            <el-select v-model="form.target" filterable placeholder="请选择容器" style="width:100%">
              <el-option v-for="c in containers" :key="c.value" :label="c.label" :value="c.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="操作" required>
            <el-radio-group v-model="form.action">
              <el-radio-button value="start">启动</el-radio-button>
              <el-radio-button value="stop">停止</el-radio-button>
              <el-radio-button value="restart">重启</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="Compose 项目" required>
            <el-select v-model="form.target" filterable placeholder="请选择项目" style="width:100%">
              <el-option v-for="p in composeProjects" :key="p" :label="p" :value="p" />
            </el-select>
          </el-form-item>
          <el-form-item label="操作" required>
            <el-radio-group v-model="form.action">
              <el-radio-button value="up">启动</el-radio-button>
              <el-radio-button value="down">停止</el-radio-button>
              <el-radio-button value="restart">重启</el-radio-button>
              <el-radio-button value="pull">拉取镜像</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </template>

        <el-form-item label="执行计划" required>
          <el-radio-group v-model="form.scheduleMode" style="margin-bottom:8px">
            <el-radio-button value="simple">简单模式</el-radio-button>
            <el-radio-button value="cron">Cron 表达式</el-radio-button>
          </el-radio-group>

          <template v-if="form.scheduleMode === 'simple'">
            <div class="simple-schedule">
              <el-select v-model="form.freq" style="width:110px" @change="form.weekdays = [1]">
                <el-option label="每天" value="day" />
                <el-option label="每周" value="week" />
                <el-option label="每月" value="month" />
              </el-select>
              <el-select v-if="form.freq === 'week'" v-model="form.weekdays" multiple collapse-tags style="width:220px">
                <el-option v-for="(label, idx) in weekdayLabels" :key="idx" :label="label" :value="idx" />
              </el-select>
              <el-input-number v-if="form.freq === 'month'" v-model="form.dayOfMonth" :min="1" :max="31" style="width:110px" />
              <el-time-picker v-model="form.time" format="HH:mm" placeholder="时间" style="width:130px" />
            </div>
            <div class="hint" v-if="simpleCronPreview">对应表达式: <span class="mono">{{ simpleCronPreview }}</span></div>
          </template>

          <template v-else>
            <el-input v-model="form.schedule" placeholder="分 时 日 月 周，例如: 0 8 * * *" class="mono" />
            <div class="hint">
              支持 <span class="mono">*/n</span> 步长、<span class="mono">a-b</span> 范围、<span class="mono">a,b</span> 列表及
              <span class="mono">@daily</span> 等快捷方式
            </div>
          </template>
        </el-form-item>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showForm = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-drawer>

    <!-- 执行记录抽屉 -->
    <el-drawer v-model="showLogs" :title="`执行记录 - ${logsTask?.name || ''}`" size="55%">
      <div style="margin-bottom:12px">
        <el-button size="small" @click="refreshLogs">刷新</el-button>
        <el-popconfirm title="清空该任务全部执行记录？" @confirm="clearLogs">
          <template #reference><el-button size="small" type="danger" plain>清空记录</el-button></template>
        </el-popconfirm>
      </div>
      <el-table :data="logs" stripe size="small" v-loading="logsLoading">
        <el-table-column label="开始时间" width="170">
          <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="110">
          <template #default="{ row }">{{ duration(row) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="输出">
          <template #default="{ row }">
            <el-link type="primary" @click="showOutput(row)">查看输出</el-link>
          </template>
        </el-table-column>
        <template #empty><el-empty description="暂无执行记录" /></template>
      </el-table>
    </el-drawer>

    <!-- 输出对话框 -->
    <el-dialog v-model="showOutputDialog" title="执行输出" width="70%">
      <pre class="output-box">{{ outputText || '（无输出）' }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import cronApi, { type CronTask, type CronLog } from '@/api/cron'
import containerApi from '@/api/containers'
import composeApi from '@/api/compose'

interface TaskRow extends CronTask {
  _running?: boolean
  _toggling?: boolean
}

const tasks = ref<TaskRow[]>([])
const loading = ref(false)
const showForm = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const containers = ref<{ label: string; value: string }[]>([])
const composeProjects = ref<string[]>([])

const form = ref({
  name: '',
  type: 'script' as TaskRow['type'],
  shell: 'sh',
  script: '',
  target: '',
  action: 'start',
  scheduleMode: 'simple' as 'simple' | 'cron',
  freq: 'day',
  weekdays: [1] as number[],
  dayOfMonth: 1,
  time: null as Date | null,
  schedule: '0 8 * * *',
  enabled: true,
})

const showLogs = ref(false)
const logsTask = ref<TaskRow | null>(null)
const logs = ref<CronLog[]>([])
const logsLoading = ref(false)

const showOutputDialog = ref(false)
const outputText = ref('')

const typeTagMap: Record<string, { label: string; tag: 'primary' | 'warning' | 'success' }> = {
  script: { label: '脚本', tag: 'primary' },
  container: { label: '容器', tag: 'success' },
  compose: { label: 'Compose', tag: 'warning' },
}

const weekdayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

function actionLabel(type: string, action?: string): string {
  if (type === 'container') return { start: '启动', stop: '停止', restart: '重启' }[action || ''] || action || ''
  if (type === 'compose') return { up: '启动', down: '停止', restart: '重启', pull: '拉取镜像' }[action || ''] || action || ''
  return ''
}

function statusLabel(status: string): string {
  return { running: '执行中', success: '成功', failed: '失败' }[status] || status
}

function scriptBrief(script?: string): string {
  if (!script) return '-'
  const oneLine = script.replace(/\s+/g, ' ').trim()
  return oneLine.length > 40 ? oneLine.slice(0, 40) + '…' : oneLine
}

function formatTime(t?: string | null): string {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function duration(row: CronLog): string {
  if (!row.finished_at) return '-'
  const ms = new Date(row.finished_at).getTime() - new Date(row.started_at).getTime()
  if (isNaN(ms) || ms < 0) return '-'
  if (ms < 1000) return '<1s'
  const s = Math.round(ms / 1000)
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m${s % 60}s`
}

/** 简单模式转 cron 表达式 */
const simpleCronPreview = computed(() => {
  const f = form.value
  const time = f.time
  if (!time) return ''
  const h = time.getHours()
  const m = time.getMinutes()
  if (f.freq === 'day') return `${m} ${h} * * *`
  if (f.freq === 'week') {
    if (!f.weekdays.length) return ''
    return `${m} ${h} * * ${[...f.weekdays].sort().join(',')}`
  }
  return `${m} ${h} ${f.dayOfMonth} * *`
})

/** cron 表达式的人话提示 */
function scheduleHint(expr: string): string {
  switch (expr) {
    case '@hourly': return '每小时'
    case '@daily': return '每天 00:00'
    case '@weekly': return '每周日 00:00'
    case '@monthly': return '每月 1 号 00:00'
  }
  const parts = expr.split(/\s+/)
  if (parts.length !== 5) return ''
  const [m = '', h = '', dom = '', mon = '', dow = ''] = parts
  if (m.startsWith('*/') && h === '*' && dom === '*' && mon === '*' && dow === '*') return `每 ${m.slice(2)} 分钟`
  if (h.startsWith('*/') && m !== '*' && dom === '*' && mon === '*' && dow === '*') return `每 ${h.slice(2)} 小时的第 ${m} 分`
  if (dom === '*' && mon === '*' && dow === '*' && !h.includes('*') && !m.includes('*')) return `每天 ${h.padStart(2, '0')}:${m.padStart(2, '0')}`
  if (dom === '*' && mon === '*' && dow !== '*' && !h.includes('*') && !m.includes('*')) {
    const days = dow.split(',').map((d) => weekdayLabels[Number(d)] ?? d).join('、')
    return `每周${days === '每周' ? '' : ''} ${days} ${h.padStart(2, '0')}:${m.padStart(2, '0')}`
  }
  if (dom !== '*' && mon === '*' && dow === '*' && !h.includes('*') && !m.includes('*')) return `每月 ${dom} 号 ${h.padStart(2, '0')}:${m.padStart(2, '0')}`
  return ''
}

async function loadTasks() {
  loading.value = true
  try {
    tasks.value = (await cronApi.list()) as TaskRow[]
  } finally {
    loading.value = false
  }
}

async function loadOptions() {
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
  } catch { /* 忽略，下拉为空 */ }

  try {
    composeProjects.value = (await composeApi.list()).map((p: any) => p.Name || p.name)
  } catch { /* 忽略 */ }
}

function resetForm() {
  form.value = {
    name: '',
    type: 'script',
    shell: 'sh',
    script: '',
    target: '',
    action: 'start',
    scheduleMode: 'simple',
    freq: 'day',
    weekdays: [1],
    dayOfMonth: 1,
    time: null,
    schedule: '0 8 * * *',
    enabled: true,
  }
}

function openCreate() {
  editingId.value = null
  resetForm()
  showForm.value = true
}

function openEdit(row: TaskRow) {
  editingId.value = row.id
  const f = form.value
  f.name = row.name
  f.type = row.type
  f.shell = row.shell || 'sh'
  f.script = row.script || ''
  f.target = row.target || ''
  f.action = row.action || 'start'
  f.enabled = row.enabled
  f.scheduleMode = 'cron'
  f.schedule = row.schedule
  showForm.value = true
}

async function save() {
  const f = form.value
  const schedule = f.scheduleMode === 'simple' ? simpleCronPreview.value : f.schedule.trim()
  if (!f.name.trim()) { ElMessage.warning('请输入任务名称'); return }
  if (!schedule) { ElMessage.warning('请设置执行计划'); return }
  if (f.type === 'script' && !f.script.trim()) { ElMessage.warning('请输入脚本内容'); return }
  if (f.type !== 'script' && !f.target) { ElMessage.warning(f.type === 'container' ? '请选择容器' : '请选择 Compose 项目'); return }

  const payload: Partial<CronTask> = {
    name: f.name.trim(),
    type: f.type,
    schedule,
    shell: f.shell,
    script: f.script,
    target: f.target,
    action: f.type === 'script' ? '' : f.action,
    enabled: f.enabled,
  }

  saving.value = true
  try {
    if (editingId.value) {
      await cronApi.update({ id: editingId.value, ...payload })
      ElMessage.success('已保存')
    } else {
      await cronApi.create(payload)
      ElMessage.success('已创建')
    }
    showForm.value = false
    await loadTasks()
  } finally {
    saving.value = false
  }
}

async function runNow(row: TaskRow) {
  row._running = true
  try {
    await cronApi.run(row.id)
    ElMessage.success('已触发执行，可在执行记录中查看结果')
    setTimeout(loadTasks, 1500)
  } finally {
    row._running = false
  }
}

async function toggleEnabled(row: TaskRow) {
  row._toggling = true
  try {
    const updated = await cronApi.update({ ...row, id: row.id, enabled: !row.enabled })
    row.enabled = updated.enabled
    row.next_run_at = updated.next_run_at
  } catch {
    // 失败保持原状态
  } finally {
    row._toggling = false
  }
}

async function removeTask(row: TaskRow) {
  await cronApi.remove(row.id)
  ElMessage.success('已删除')
  await loadTasks()
}

async function openLogs(row: TaskRow) {
  logsTask.value = row
  showLogs.value = true
  await refreshLogs()
}

async function refreshLogs() {
  if (!logsTask.value) return
  logsLoading.value = true
  try {
    logs.value = await cronApi.logs(logsTask.value.id)
  } finally {
    logsLoading.value = false
  }
}

async function clearLogs() {
  if (!logsTask.value) return
  await cronApi.clearLogs(logsTask.value.id)
  ElMessage.success('已清空')
  await refreshLogs()
}

function showOutput(row: CronLog) {
  outputText.value = row.output || ''
  showOutputDialog.value = true
}

onMounted(() => {
  loadTasks()
  loadOptions()
})
</script>

<style scoped>
.cron-page {
  padding: 16px;
}
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
}
.mono {
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.simple-schedule {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.output-box {
  background: #1e1e1e;
  color: #0f0;
  padding: 12px;
  border-radius: 4px;
  max-height: 55vh;
  overflow: auto;
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
  font-size: 13px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
