<template>
  <div class="compose-page">
    <div class="page-header">
      <h2>Compose 管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="openCreate">创建项目</el-button>
        <el-button :loading="loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>
    <el-table v-loading="loading" :data="projects" stripe style="width:100%">
      <el-table-column label="名称" min-width="150">
        <template #default="{ row }"><el-link type="primary" @click="openDetail(row)">{{ row.name }}</el-link></template>
      </el-table-column>
      <el-table-column label="描述" min-width="120">
        <template #default="{ row }"><span>{{ row.description || '-' }}</span></template>
      </el-table-column>
      <el-table-column label="端口" min-width="150"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.status === 'running'" type="success" size="small" effect="plain">运行中</el-tag>
          <el-tag v-else type="warning" size="small" effect="plain">等待部署</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <div style="display:flex;flex-direction:row;gap:4px">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="primary" size="small">日志</el-button>
            <el-button link type="primary" size="small">控制台</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDeleteProject(row.name)">
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
        :total="projects.length"
        layout="total, sizes, prev, pager, next, jumper"
      />
    </div>

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
        <el-button @click="importLocal">导入本地 Compose 文件</el-button>
        <el-button @click="showRunDrawer = true">通过 docker run 命令</el-button>
      </div>
      <el-input v-model="form.yaml" type="textarea" :rows="12" placeholder="services:&#10;  xxx:&#10;    image: xxx" style="font-family:monospace" @input="autoName" />
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">确定</el-button>
      </template>
    </el-drawer>

    <!-- docker run drawer -->
    <el-drawer v-model="showRunDrawer" title="docker run 命令" size="60%" :close-on-click-modal="false">
      <el-input v-model="runCmd" type="textarea" :rows="15" placeholder="请输入 docker run ... 命令" style="font-family:monospace" />
      <template #footer>
        <el-button @click="showRunDrawer = false">取消</el-button>
        <el-button type="primary" @click="convertRun">确定</el-button>
      </template>
    </el-drawer>

    <!-- 任务管理 drawer -->
    <el-drawer v-model="showDetail" :title="'任务管理 ' + currentProject" size="60%" :close-on-click-modal="false">
      <h4 style="margin-bottom:12px">容器管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:24px;flex-wrap:wrap">
        <el-button type="primary" :loading="deploying" @click="handleDeploy">{{ projectRunning ? '更新' : '启动' }}</el-button>
        <el-button @click="handleAction('restart')">重启</el-button>
        <el-button @click="handleAction('stop')">停止</el-button>
        <el-button @click="handleAction('pause')">暂停</el-button>
        <el-button>控制台</el-button>
        <el-button @click="handleAction('logs')">日志</el-button>
        <el-popconfirm title="确认要删除容器？" @confirm="handleAction('down')">
          <template #reference><el-button type="danger" plain>删除容器</el-button></template>
        </el-popconfirm>
      </div>
      <h4 style="margin-bottom:12px">任务管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:12px">
        <el-button type="primary" @click="openEdit({name: currentProject})">编辑任务</el-button>
        <el-button @click="handleRefresh">刷新</el-button>
      </div>
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
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

const projects = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const showCreate = ref(false)
const showDetail = ref(false)
const showRunDrawer = ref(false)
const currentProject = ref('')
const projectRunning = ref(false)
const deploying = ref(false)
const showDeployDialog = ref(false)
const pullProgress = ref(0)
const deployProgress = ref(0)
const pullStatus = ref('等待中')
const deployStatus = ref('等待中')
const deployDone = ref(false)
const runCmd = ref('')
const form = ref({ name: '', description: '', source: 'text', yaml: '' })

function autoName() {
  const m = form.value.yaml.match(/container_name:\s*(\S+)/)
  if (m && m[1]) form.value.name = m[1]
}

function openCreate() {
  form.value = { name: '', description: '', source: 'text', yaml: '' }
  showCreate.value = true
}

async function openEdit(row: any) {
  form.value = { name: row.name, description: '', source: 'text', yaml: '' }
  try {
    const res = await request.get('/compose/file?project=' + row.name)
    form.value.yaml = res.data || ''
  } catch {}
  showCreate.value = true
}

function openDetail(row: any) {
  currentProject.value = row.name
  projectRunning.value = row.status === 'running'
  showDetail.value = true
}

async function handleAction(action: string) {
  try {
    await fetch('/api/compose/action', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project: currentProject.value, action })
    })
    if (action === 'down') {
      projectRunning.value = false
    }
    setTimeout(() => fetchProjects(), 1000)
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
  for (const m of volMatches) volumes.push(`- ${m[1]}`)

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

async function fetchProjects() {
  loading.value = true
  try {
    const res = await request.get('/compose/list')
    projects.value = res.data || []
  } catch {}
  loading.value = false
}

function handleRefresh() { fetchProjects() }

async function handleCreate() {
  if (!form.value.name) return
  try {
    await request.post('/compose/create', { name: form.value.name, content: form.value.yaml })
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
    projectRunning.value = true
    fetchProjects()
  } catch {
    pullStatus.value = '失败'
    deployStatus.value = '失败'
  }
  deploying.value = false
}

async function handleDeleteProject(name: string) {
  try {
    await request.post('/compose/delete', { name })
    fetchProjects()
  } catch {}
}

onMounted(() => { fetchProjects() })
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
</style>
