<template>
  <div class="cron-page">
    <div class="page-header">
      <h2>计划任务</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreate = true">创建计划任务</el-button>
        <el-button type="primary" @click="showTimer = true">创建定时任务</el-button>
      </div>
    </div>
    <el-tabs v-model="tab">
      <el-tab-pane label="计划任务" name="task">
        <el-table :data="tasks" stripe style="width:100%">
          <el-table-column label="描述" prop="name" />
          <el-table-column label="执行容器" prop="container" />
          <el-table-column label="执行计划" prop="schedule" />
          <el-table-column label="下次时间" prop="next" />
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="runTask(row)">执行</el-button>
              <el-button link type="primary" size="small" @click="editTask(row)">编辑</el-button>
              <el-popconfirm title="确定删除？" @confirm="deleteTask(row)">
                <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="定时任务" name="timer">
        <el-table :data="timers" stripe style="width:100%">
          <el-table-column label="任务名称" prop="name" />
          <el-table-column label="目标" prop="target" />
          <el-table-column label="类型" prop="type" />
          <el-table-column label="Cron表达式" prop="cron" />
          <el-table-column label="执行规则" prop="rule" />
          <el-table-column label="下次执行" prop="next" />
          <el-table-column label="启用" width="80">
            <template #default="{ row }"><el-switch v-model="row.enabled" /></template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default>
              <el-button link type="primary" size="small">编辑</el-button>
              <el-button link type="danger" size="small">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="showCreate" title="添加计划任务" size="60%" :close-on-click-modal="false">
      <h4 style="margin-bottom:8px">名称 <span style="color:#f56c6c">*</span></h4>
      <el-input v-model="form.name" placeholder="计划任务名称" style="margin-bottom:16px" />
      <h4 style="margin-bottom:8px">触发类型</h4>
      <el-radio-group v-model="form.trigger" style="margin-bottom:16px">
        <el-radio value="period">周期</el-radio>
        <el-radio value="manual">手动</el-radio>
        <el-radio value="event">事件</el-radio>
      </el-radio-group>
      <h4 style="margin-bottom:8px">执行周期</h4>
      <div style="display:flex;gap:8px;margin-bottom:16px">
        <el-select v-model="form.period" style="width:120px">
          <el-option label="每周" value="week" />
          <el-option label="每天" value="day" />
        </el-select>
        <el-input v-model="form.hour" placeholder="时" style="width:100px" />
        <el-input v-model="form.minute" placeholder="分" style="width:100px" />
      </div>
      <h4 style="margin-bottom:8px">脚本执行环境</h4>
      <el-select v-model="form.shell" style="width:200px;margin-bottom:16px">
        <el-option label="sh" value="sh" />
        <el-option label="bash" value="bash" />
      </el-select>
      <h4 style="margin-bottom:8px">执行脚本</h4>
      <el-input v-model="form.script" type="textarea" :rows="8" style="font-family:monospace;margin-bottom:16px" />
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="saveTask">确定</el-button>
      </template>
    </el-drawer>

    <el-drawer v-model="showTimer" title="新增定时任务" size="50%" :close-on-click-modal="false">
      <h4 style="margin-bottom:8px">任务名称</h4>
      <el-input v-model="timerForm.name" placeholder="例如: 每天早8点启动" style="margin-bottom:16px" />
      <h4 style="margin-bottom:8px">目标类型</h4>
      <el-radio-group v-model="timerForm.targetType" style="margin-bottom:16px">
        <el-radio-button value="container">容器</el-radio-button>
        <el-radio-button value="compose">Compose项目</el-radio-button>
      </el-radio-group>
      <h4 style="margin-bottom:8px">选择容器</h4>
      <el-select v-model="timerForm.target" placeholder="请选择容器" style="width:100%;margin-bottom:16px">
        <el-option v-for="c in containers" :key="c" :label="c" :value="c" />
      </el-select>
      <h4 style="margin-bottom:8px">操作类型</h4>
      <el-radio-group v-model="timerForm.action" style="margin-bottom:16px">
        <el-radio-button value="start">启动</el-radio-button>
        <el-radio-button value="stop">停止</el-radio-button>
        <el-radio-button value="restart">重启</el-radio-button>
        <el-radio-button value="update">更新</el-radio-button>
      </el-radio-group>
      <h4 style="margin-bottom:8px">执行频率</h4>
      <el-radio-group v-model="timerForm.freq" style="margin-bottom:16px">
        <el-radio-button value="day">每天</el-radio-button>
        <el-radio-button value="week">每周</el-radio-button>
        <el-radio-button value="month">每月</el-radio-button>
      </el-radio-group>
      <h4 style="margin-bottom:8px">执行时间</h4>
      <el-time-picker v-model="timerForm.time" placeholder="选择时间" style="margin-bottom:16px" />
      <h4 style="margin-bottom:8px">启用</h4>
      <el-switch v-model="timerForm.enabled" style="margin-bottom:16px" />
      <template #footer>
        <el-button @click="showTimer = false">取消</el-button>
        <el-button type="primary" @click="saveTimer">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { containerApi } from '@/api/containers'

const tab = ref('task')
const tasks = ref<any[]>([])
const timers = ref<any[]>([])
const showCreate = ref(false)
const showTimer = ref(false)
const form = ref({ name: '', trigger: 'period', period: 'week', hour: '', minute: '', shell: 'sh', script: '' })
const timerForm = ref({ name: '', targetType: 'container', target: '', action: 'start', freq: 'day', time: '', enabled: true })
const containers = ref<string[]>([])

async function loadContainers() {
  try {
    const res = await containerApi.list()
    containers.value = (res || []).map((c: any) => c.Names?.[0]?.replace('/', '') || c.Name || '')
  } catch {}
}

function saveTask() {
  tasks.value.push({ name: form.value.name, container: '-', schedule: '手动', next: '-' })
  showCreate.value = false
  form.value = { name: '', trigger: 'period', period: 'week', hour: '', minute: '', shell: 'sh', script: '' }
}

function runTask(row: any) {
  row.next = '执行中...'
  setTimeout(() => { row.next = new Date().toLocaleString() }, 1000)
}

function editTask(row: any) {
  form.value.name = row.name
  showCreate.value = true
}

function deleteTask(row: any) {
  tasks.value = tasks.value.filter(t => t !== row)
}

function saveTimer() {
  timers.value.push({
    name: timerForm.value.name,
    target: timerForm.value.target,
    type: timerForm.value.action,
    cron: '-',
    rule: '-',
    next: '-',
    enabled: timerForm.value.enabled
  })
  showTimer.value = false
}

onMounted(loadContainers)
</script>
