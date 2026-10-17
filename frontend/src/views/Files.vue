<template>
  <div class="files-page">
    <div class="page-header">
      <h2>文件管理</h2>
    </div>
    <div style="display:flex;gap:8px;align-items:center;margin-bottom:16px">
      <el-button @click="goBack">←</el-button>
      <el-button @click="goHome">🏠</el-button>
      <span style="color:#606266">{{ currentPath }}</span>
      <el-button @click="loadFiles">刷新</el-button>
    </div>
    <el-table v-loading="loading" :data="files" stripe style="width:100%" @row-dblclick="openRow">
      <el-table-column label="名称" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" @click="openRow(row)">
            <el-icon v-if="row.isDir" style="margin-right:4px"><Folder /></el-icon>
            <el-icon v-else style="margin-right:4px"><Document /></el-icon>
            {{ row.name }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column label="权限" width="100">
        <template #default="{ row }">{{ row.mode }}</template>
      </el-table-column>
      <el-table-column label="大小" width="100">
        <template #default="{ row }">{{ row.isDir ? '-' : formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="修改时间" width="180">
        <template #default="{ row }">{{ row.modTime }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openRow(row)">打开</el-button>
          <el-button link type="primary" size="small" @click="downloadFile(row)">下载</el-button>
          <el-popconfirm title="确定删除？" @confirm="deleteFile(row)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- 编辑文件 drawer -->
    <el-drawer v-model="showEditor" :title="'编辑 - ' + editingPath" size="80%" :close-on-click-modal="false">
      <el-input v-model="editingContent" type="textarea" :rows="25" style="font-family:monospace" />
      <template #footer>
        <el-button @click="showEditor = false">取消</el-button>
        <el-button type="primary" @click="saveFile">保存</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Folder, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const currentPath = ref('/')
const files = ref<any[]>([])
const loading = ref(false)
const showEditor = ref(false)
const editingPath = ref('')
const editingContent = ref('')
async function loadFiles() {
  loading.value = true
  try {
    const res = await request.get('/fs/list', { params: { path: currentPath.value } })
    files.value = res.data.files || []
  } catch {
    ElMessage.error('加载失败')
  }
  loading.value = false
}

function openRow(row: any) {
  const newPath = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
  if (row.isDir) {
    currentPath.value = newPath
    loadFiles()
  } else {
    editingPath.value = newPath
    showEditor.value = true
    readFile(newPath)
  }
}

async function readFile(path: string) {
  try {
    const res = await request.get('/fs/read', { params: { path } })
    editingContent.value = res.data.content || ''
  } catch {
    ElMessage.error('读取失败')
  }
}

async function saveFile() {
  try {
    await request.post('/fs/write', { path: editingPath.value, content: editingContent.value })
    ElMessage.success('保存成功')
    showEditor.value = false
  } catch {
    ElMessage.error('保存失败')
  }
}

function goBack() {
  if (currentPath.value === '/') return
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  currentPath.value = '/' + parts.join('/')
  loadFiles()
}

function goHome() {
  currentPath.value = '/'
  loadFiles()
}

function downloadFile(row: any) {
  const path = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
  window.open('/api/fs/read?path=' + encodeURIComponent(path))
}

async function deleteFile(row: any) {
  const path = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
  try {
    await request.post('/fs/delete', { path })
    ElMessage.success('删除成功')
    loadFiles()
  } catch {
    ElMessage.error('删除失败')
  }
}

function formatSize(size: number) {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

onMounted(loadFiles)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
</style>
