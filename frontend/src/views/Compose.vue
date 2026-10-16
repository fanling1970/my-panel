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
      <el-table-column label="描述" min-width="120"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="端口" min-width="150"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="目录" width="80">
        <template #default="{ row }"><el-icon @click="openDir(row)" style="cursor:pointer"><Folder /></el-icon></template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.status === 'running'" type="success" size="small" effect="plain">运行中</el-tag>
          <el-tag v-else type="warning" size="small" effect="plain">等待部署</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openCreate(row)">编辑</el-button>
          <el-popconfirm title="确定删除？" @confirm="handleDeleteProject(row.name)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建项目 drawer -->
    <el-drawer v-model="showCreate" title="Compose Yaml" size="60%">
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
        <el-button @click="dockerRunMode">通过 docker run 命令</el-button>
      </div>
      <el-input v-model="form.yaml" type="textarea" :rows="12" placeholder="services:&#10;  xxx:&#10;    image: xxx" style="font-family:monospace" @input="autoName" />
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">确定</el-button>
      </template>
    </el-drawer>

    <!-- 任务管理 drawer -->
    <el-drawer v-model="showDetail" :title="'任务管理 ' + currentProject" size="60%">
      <h4 style="margin-bottom:12px">容器管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:24px">
        <el-button type="primary" :loading="deploying" @click="handleDeploy">启动</el-button>
        <el-button>控制台</el-button>
        <el-button type="danger" plain>删除</el-button>
      </div>
      <h4 style="margin-bottom:12px">任务管理</h4>
      <div style="display:flex;gap:12px;margin-bottom:12px">
        <el-button type="primary" @click="openCreate({name: currentProject})">编辑任务</el-button>
        <el-button>目录</el-button>
        <el-button @click="handleRefresh">刷新</el-button>
        <el-button>同步</el-button>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Folder } from '@element-plus/icons-vue'
import request from '@/utils/request'

const projects = ref<any[]>([])
const loading = ref(false)
const showCreate = ref(false)
const showDetail = ref(false)
const currentProject = ref('')
const deploying = ref(false)
const form = ref({ name: '', description: '', source: 'text', yaml: '' })

function autoName() {
  const m = form.value.yaml.match(/container_name:\s*(\S+)/)
  if (m && m[1]) form.value.name = m[1]
}

function openCreate(row?: any) {
  if (row) {
    form.value.name = row.name
  }
  showCreate.value = true
}

function openDetail(row: any) {
  currentProject.value = row.name
  showDetail.value = true
}

function openDir(row: any) {
  window.open('/api/compose/file?project=' + row.name)
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

function dockerRunMode() {
  ElMessageBox.prompt('请输入 docker run 命令', 'docker run 命令', { inputType: 'textarea' })
    .then(({ value }) => { form.value.yaml = value })
    .catch(() => {})
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
    form.value = { name: '', description: '', source: 'text', yaml: '' }
    fetchProjects()
  } catch {}
}

async function handleDeploy() {
  deploying.value = true
  try {
    await request.post('/compose/action', { name: currentProject.value, action: 'up' })
    ElMessage.success('部署完成')
    fetchProjects()
  } catch {}
  deploying.value = false
}

async function handleDeleteProject(name: string) {
  try {
    await request.post('/compose/delete', { name })
    fetchProjects()
  } catch {}
}

import { ElMessage, ElMessageBox } from 'element-plus'
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
