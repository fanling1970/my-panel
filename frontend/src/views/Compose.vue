<template>
  <div class="compose-page">
    <div class="page-header">
      <h2>Compose 管理</h2>
      <div class="header-actions">
        <el-button type="primary">创建项目</el-button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Folder } from '@element-plus/icons-vue'
import request from '@/utils/request'

const projects = ref<any[]>([])
const loading = ref(false)

async function fetchProjects() {
  loading.value = true
  try {
    const res = await request.get('/api/compose/projects')
    projects.value = res.data || []
  } catch {}
  loading.value = false
}

function handleRefresh() { fetchProjects() }

async function handleDeleteProject(name: string) {
  try {
    await request.delete('/api/compose/projects/' + name)
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
