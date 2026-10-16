<template>
  <div class="compose-page">
    <div class="page-header">
      <h2>Compose 管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreate = true">创建项目</el-button>
        <el-button :loading="loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>
    <el-table v-loading="loading" :data="projects" stripe style="width:100%">
      <el-table-column label="名称" min-width="150">
        <template #default="{ row }"><el-link type="primary">{{ row.name }}</el-link></template>
      </el-table-column>
      <el-table-column label="描述" min-width="120"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="端口" min-width="150"><template #default><span>-</span></template></el-table-column>
      <el-table-column label="目录" width="80"><template #default><el-icon><Folder /></el-icon></template></el-table-column>
      <el-table-column label="状态" width="120">
        <template #default>
          <el-tag type="success" size="small" effect="plain">运行中</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180">
        <template #default><span>-</span></template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small">编辑</el-button>
          <el-popconfirm title="确定删除？" @confirm="handleDeleteProject(row.name)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="showCreate" title="Compose Yaml" size="60%">
      <h4 style="margin-bottom:8px">名称 <span style="color:#f56c6c">*</span></h4>
      <el-input v-model="form.name" placeholder="标识 Compose 创建的容器组" style="margin-bottom:16px" />
      <h4 style="margin-bottom:8px">描述</h4>
      <el-input v-model="form.description" placeholder="请输入" style="margin-bottom:16px" />
      <h4 style="margin-bottom:12px">Yaml 来源</h4>
      <el-radio-group v-model="form.source" style="margin-bottom:16px">
        <el-radio value="text">文本 Yaml</el-radio>
      </el-radio-group>
      <el-input v-model="form.yaml" type="textarea" :rows="15" placeholder="services:&#10;  xxx:&#10;    image: xxx" style="font-family:monospace" />
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">确定</el-button>
      </template>
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
const form = ref({ name: '', description: '', source: 'text', yaml: '' })

async function handleCreate() {
  if (!form.value.name) return
  try {
    await request.post('/api/compose/create', {
      name: form.value.name,
      content: form.value.yaml
    })
    showCreate.value = false
    form.value = { name: '', description: '', source: 'text', yaml: '' }
    fetchProjects()
  } catch {}
}

async function fetchProjects() {
  loading.value = true
  try {
    const res = await request.get('/api/compose/list')
    projects.value = res.data || []
  } catch {}
  loading.value = false
}

function handleRefresh() { fetchProjects() }

async function handleDeleteProject(name: string) {
  try {
    await request.post('/api/compose/delete', { name })
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
