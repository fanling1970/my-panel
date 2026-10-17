<template>
  <div class="files-page">
    <div class="page-header">
      <h2>文件管理</h2>
    </div>
    <div style="display:flex;gap:8px;align-items:center;margin-bottom:12px">
      <el-button @click="goBack">←</el-button>
      <el-button @click="goHome">🏠</el-button>
      <span style="color:#606266">{{ currentPath }}</span>
      <el-button @click="loadFiles">刷新</el-button>
    </div>
    <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
      <el-button size="small" @click="newFile">新建文件</el-button>
      <el-button size="small" @click="newFolder">新建文件夹</el-button>
      <el-button size="small" @click="triggerUpload">上传</el-button>
      <el-button size="small" :disabled="!selected.length" @click="downloadSelected">下载</el-button>
      <el-button size="small" :disabled="!selected.length" @click="chmodSelected">编辑权限</el-button>
      <el-button size="small" type="danger" plain :disabled="!selected.length" @click="deleteSelected">删除</el-button>
      <el-input v-model="search" placeholder="搜索" size="small" style="max-width:200px;margin-left:auto" clearable />
    </div>
    <input ref="fileInput" type="file" style="display:none" @change="uploadFile" />
    <el-table v-loading="loading" :data="filteredFiles" stripe style="width:100%" @selection-change="onSelect" @row-dblclick="openRow">
      <el-table-column type="selection" width="40" />
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
          <el-popconfirm title="确定删除？" @confirm="deleteSingle(row)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

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
import { ref, computed, onMounted } from 'vue'
import { Folder, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const currentPath = ref('/host')
const files = ref<any[]>([])
const selected = ref<any[]>([])
const search = ref('')
const loading = ref(false)
const showEditor = ref(false)
const editingPath = ref('')
const editingContent = ref('')
const fileInput = ref<HTMLInputElement>()

const filteredFiles = computed(() => {
  if (!search.value) return files.value
  return files.value.filter(f => f.name.includes(search.value))
})

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

function onSelect(rows: any[]) { selected.value = rows }

function openRow(row: any) {
  const newPath = joinPath(row.name)
  if (row.isDir) {
    currentPath.value = newPath
    loadFiles()
  } else {
    editingPath.value = newPath
    showEditor.value = true
    readFile(newPath)
  }
}

function joinPath(name: string) {
  return currentPath.value === '/' ? '/' + name : currentPath.value + '/' + name
}

async function readFile(path: string) {
  try {
    const res = await request.get('/fs/read', { params: { path } })
    editingContent.value = res.data.content || ''
  } catch { ElMessage.error('读取失败') }
}

async function saveFile() {
  try {
    await request.post('/fs/write', { path: editingPath.value, content: editingContent.value })
    ElMessage.success('保存成功')
    showEditor.value = false
  } catch { ElMessage.error('保存失败') }
}

function goBack() {
  if (currentPath.value === '/' || currentPath.value === '/host') return
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  currentPath.value = '/' + parts.join('/')
  loadFiles()
}

function goHome() {
  currentPath.value = '/host'
  loadFiles()
}

async function newFile() {
  const { value } = await ElMessageBox.prompt('文件名', '新建文件')
  try {
    await request.post('/fs/touch', { path: currentPath.value, name: value })
    loadFiles()
  } catch { ElMessage.error('创建失败') }
}

async function newFolder() {
  const { value } = await ElMessageBox.prompt('文件夹名', '新建文件夹')
  try {
    await request.post('/fs/mkdir', { path: currentPath.value, name: value })
    loadFiles()
  } catch { ElMessage.error('创建失败') }
}

function triggerUpload() { fileInput.value?.click() }

async function uploadFile(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const file = input.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('path', currentPath.value)
  fd.append('file', file)
  try {
    await request.post('/fs/upload', fd)
    ElMessage.success('上传成功')
    loadFiles()
  } catch { ElMessage.error('上传失败') }
  input.value = ''
}

async function deleteSingle(row: any) {
  const path = joinPath(row.name)
  try {
    await request.post('/fs/delete', { path })
    ElMessage.success('删除成功')
    loadFiles()
  } catch { ElMessage.error('删除失败') }
}

async function deleteSelected() {
  for (const row of selected.value) {
    await request.post('/fs/delete', { path: joinPath(row.name) }).catch(() => {})
  }
  ElMessage.success('删除成功')
  loadFiles()
}

function downloadSelected() {
  for (const row of selected.value) {
    if (!row.isDir) {
      window.open('/api/fs/read?path=' + encodeURIComponent(joinPath(row.name)))
    }
  }
}

async function chmodSelected() {
  const { value } = await ElMessageBox.prompt('权限 (如 755)', '编辑权限')
  for (const row of selected.value) {
    await request.post('/fs/chmod', { path: joinPath(row.name), mode: value }).catch(() => {})
  }
  ElMessage.success('权限已修改')
  loadFiles()
}

function formatSize(size: number) {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

onMounted(loadFiles)
</script>
