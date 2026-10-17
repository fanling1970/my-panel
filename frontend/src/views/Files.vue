<template>
  <div class="files-page">
    <div class="page-header">
      <h2 style="margin-bottom:20px">文件管理</h2>
    </div>
    <div style="display:flex;gap:8px;align-items:center;margin-bottom:16px;padding:10px 16px;background:#f5f7fa;border-radius:6px">
      <el-button @click="goBack">←</el-button>
      <el-button @click="goHome">🏠</el-button>
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: '/host' }" @click="goHome">host</el-breadcrumb-item>
        <el-breadcrumb-item v-for="(seg, i) in pathSegments" :key="i">
          <el-link type="primary" @click="goToSegment(i)">{{ seg }}</el-link>
        </el-breadcrumb-item>
      </el-breadcrumb>
      <el-button @click="loadFiles" style="margin-left:auto">刷新</el-button>
    </div>
    <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap;align-items:center">
      <el-button size="small" @click="newFile">新建文件</el-button>
      <el-button size="small" @click="newFolder">新建文件夹</el-button>
      <el-button size="small" @click="triggerUpload">上传</el-button>
      <el-button size="small" :disabled="!selected.length" @click="startCopy">复制</el-button>
      <el-button size="small" :disabled="!selected.length" @click="startMove">移动</el-button>
      <el-button size="small" :disabled="!selected.length" @click="downloadSelected">下载</el-button>
      <el-button size="small" :disabled="!selected.length" @click="showChmod = true">编辑权限</el-button>
      <el-button size="small" type="danger" plain :disabled="!selected.length" @click="deleteSelected">删除</el-button>
      <template v-if="clipboard.length">
        <el-button size="small" type="primary" @click="paste">粘贴({{ clipboard.length }})</el-button>
        <el-button size="small" type="danger" circle @click="clipboard = []">✕</el-button>
      </template>
      <el-input v-model="search" placeholder="搜索" size="small" style="max-width:200px;margin-left:auto" clearable />
    </div>
    <input ref="fileInput" type="file" style="display:none" @change="uploadFile" />
    <el-table v-loading="loading" :data="filteredFiles" stripe style="width:100%" @selection-change="onSelect" @row-dblclick="openRow">
      <el-table-column type="selection" width="40" />
      <el-table-column label="名称" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" @click="openRow(row)">
            <span v-if="row.isDir" style="margin-right:4px">📁</span>
            <span v-else style="margin-right:4px">📄</span>
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
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openRow(row)">打开</el-button>
          <el-button link type="primary" size="small" @click="renameRow(row)">重命名</el-button>
          <el-popconfirm title="确定删除？" @confirm="deleteSingle(row)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="showEditor" :title="'编辑 - ' + editingPath" size="50%" :close-on-click-modal="false">
      <el-input v-model="editingContent" type="textarea" :rows="25" style="font-family:monospace" />
      <template #footer>
        <el-button @click="showEditor = false">取消</el-button>
        <el-button type="primary" @click="saveFile">保存</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="showChmod" title="编辑权限" width="400px">
      <div style="display:flex;flex-direction:column;gap:12px">
        <div>所有者</div>
        <div style="display:flex;gap:16px;margin-bottom:8px">
          <el-checkbox v-model="perm.ownerRead">读取</el-checkbox>
          <el-checkbox v-model="perm.ownerWrite">写入</el-checkbox>
          <el-checkbox v-model="perm.ownerExec">可执行</el-checkbox>
        </div>
        <div>用户组</div>
        <div style="display:flex;gap:16px;margin-bottom:8px">
          <el-checkbox v-model="perm.groupRead">读取</el-checkbox>
          <el-checkbox v-model="perm.groupWrite">写入</el-checkbox>
          <el-checkbox v-model="perm.groupExec">可执行</el-checkbox>
        </div>
        <div>公共</div>
        <div style="display:flex;gap:16px;margin-bottom:8px">
          <el-checkbox v-model="perm.otherRead">读取</el-checkbox>
          <el-checkbox v-model="perm.otherWrite">写入</el-checkbox>
          <el-checkbox v-model="perm.otherExec">可执行</el-checkbox>
        </div>
        <div style="color:#606266">当前权限：{{ permMode }}</div>
      </div>
      <template #footer>
        <el-button @click="showChmod = false">取消</el-button>
        <el-button type="primary" @click="applyChmod">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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
const clipboard = ref<{ src: string; isCopy: boolean }[]>([])
const showChmod = ref(false)
const perm = ref({
  ownerRead: true, ownerWrite: true, ownerExec: false,
  groupRead: true, groupWrite: false, groupExec: false,
  otherRead: true, otherWrite: false, otherExec: false
})

const permMode = computed(() => {
  const o = (perm.value.ownerRead ? 4 : 0) + (perm.value.ownerWrite ? 2 : 0) + (perm.value.ownerExec ? 1 : 0)
  const g = (perm.value.groupRead ? 4 : 0) + (perm.value.groupWrite ? 2 : 0) + (perm.value.groupExec ? 1 : 0)
  const p = (perm.value.otherRead ? 4 : 0) + (perm.value.otherWrite ? 2 : 0) + (perm.value.otherExec ? 1 : 0)
  return '' + o + g + p
})

const filteredFiles = computed(() => {
  let arr = files.value
  if (search.value) arr = arr.filter(f => f.name.includes(search.value))
  return [...arr].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

const pathSegments = computed(() => {
  if (currentPath.value === '/host') return []
  return currentPath.value.replace('/host', '').split('/').filter(Boolean)
})

function goToSegment(i: number) {
  const segs = pathSegments.value.slice(0, i + 1)
  currentPath.value = '/host/' + segs.join('/')
  loadFiles()
}

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

function startCopy() {
  clipboard.value = selected.value.map(r => ({ src: joinPath(r.name), isCopy: true }))
}

function startMove() {
  clipboard.value = selected.value.map(r => ({ src: joinPath(r.name), isCopy: false }))
}

async function paste() {
  for (const item of clipboard.value) {
    const name = item.src.split('/').pop()
    const dst = currentPath.value + '/' + name
    if (item.isCopy) {
      await request.post('/fs/copy', { src: item.src, dst }).catch(() => {})
    } else {
      await request.post('/fs/move', { src: item.src, dst }).catch(() => {})
    }
  }
  ElMessage.success(clipboard.value[0]?.isCopy ? '复制成功' : '移动成功')
  clipboard.value = []
  loadFiles()
}

async function renameRow(row: any) {
  const { value } = await ElMessageBox.prompt('新名称', '重命名', { inputValue: row.name })
  try {
    await request.post('/fs/rename', { path: joinPath(row.name), name: value })
    loadFiles()
  } catch { ElMessage.error('重命名失败') }
}

async function applyChmod() {
  for (const row of selected.value) {
    await request.post('/fs/chmod', { path: joinPath(row.name), mode: permMode.value }).catch(() => {})
  }
  ElMessage.success('权限已修改')
  showChmod.value = false
  loadFiles()
}

function formatSize(size: number) {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

onMounted(loadFiles)
</script>
