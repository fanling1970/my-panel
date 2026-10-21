<template>
  <div class="images-page">
    <!-- Header with title and actions -->
    <div class="page-header">
      <h2>镜像管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showPullDialog = true">拉取镜像</el-button>
        <el-button @click="showImportDialog = true">导入镜像</el-button>
        <el-button @click="showBuildDialog = true">构建镜像</el-button>
        <el-button @click="handlePrune">清理构建缓存</el-button>
        <el-button :loading="imageStore.loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <!-- Search bar -->
    <div class="filter-bar">
      <el-input v-model="searchInput" placeholder="请输入镜像名称" clearable style="width: 300px" @input="handleSearch">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
    </div>

    <!-- Image table -->
    <el-table v-loading="imageStore.loading" :data="imageStore.paginatedImages" stripe style="width: 100%" @sort-change="handleSortChange">
      <el-table-column prop="id" label="ID" width="140" sortable="custom">
        <template #default="{ row }">
          <el-link type="primary">{{ row.id.substring(0, 12) }}</el-link>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.used_by && row.used_by.length > 0" type="success" size="small">已使用</el-tag>
          <el-tag v-else type="info" size="small">未使用</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="tag" label="标签" min-width="200">
        <template #default="{ row }">
          <el-tag size="small" style="background:#f0f5ff;border-color:#d6e4ff;color:#1d6fd8">{{ row.name }}:{{ row.tag }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="size" label="大小" width="120" sortable="custom" />
      <el-table-column prop="created" label="时间" width="180" sortable="custom" />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="handleExport(row)">导出</el-button>
          <el-popconfirm :title="'确定删除该镜像？'" @confirm="handleRemoveImage(row.id)">
            <template #reference>
              <el-button type="danger" size="small" link>删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPageValue"
        v-model:page-size="pageSizeValue"
        :page-sizes="[10, 20, 50, 100]"
        :total="imageStore.totalImages"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>

    <!-- Build Image Dialog -->
    <BuildImageDialog
      v-model:visible="showBuildDialog"
      @built="handleImageBuilt"
    />

    <!-- 拉取镜像 Drawer -->
    <el-drawer v-model="showPullDialog" title="拉取远程镜像" size="50%">
      <div style="font-size:13px;color:#606262;line-height:2;margin-bottom:24px">
        <div>• 镜像名称未包含仓库地址时，默认从 hub.docker.io 下载。</div>
        <div>• 拉取非 hub.docker.io 镜像时，请填写完整镜像地址。</div>
      </div>
      <h4 style="margin-bottom:12px;font-size:16px">拉取镜像</h4>
      <div style="display:flex;gap:12px;margin-bottom:12px">
        <el-select v-model="pullArch" placeholder="默认当前架构" style="width:200px">
          <el-option label="amd64" value="amd64" />
          <el-option label="arm64" value="arm64" />
        </el-select>
        <el-select v-model="pullRegistry" placeholder="选择镜像仓库，默认为 docker.io" style="flex:1">
          <el-option label="docker.io" value="docker.io" />
          <el-option v-for="r in registries" :key="r.id" :label="r.name" :value="r.url" />
        </el-select>
      </div>
      <div style="display:flex;gap:12px;margin-bottom:16px">
        <el-input v-model="pullImage" placeholder="请输入镜像地址" style="flex:1" />
        <el-button type="primary" :loading="pulling" @click="handlePull">拉取镜像</el-button>
      </div>
      <h4 style="margin-bottom:12px;font-size:16px">拉取进度</h4>
      <el-progress :percentage="pullProgress" />
    </el-drawer>

    <!-- 导入镜像 Drawer -->
    <el-drawer v-model="showImportDialog" title="导入镜像" size="50%">
      <h4 style="margin-bottom:12px;font-size:16px">本地镜像</h4>
      <el-upload action="/api/images/import" :data="uploadData" :show-file-list="true" accept=".tar" drag
        style="margin-bottom:24px" :on-change="handleImportFileChange"
        :on-success="handleImportSuccess" :on-error="handleImportError" :headers="uploadHeaders">
        <div style="padding:24px">
          <div style="font-size:32px;color:#1d6fd8;margin-bottom:8px">⬆</div>
          <div>导入镜像包，最多支持同时上传 10 个。（仅限导出镜像生成的 .tar 文件）</div>
        </div>
      </el-upload>
      <h4 style="margin-bottom:12px;font-size:16px">远程地址</h4>
      <div style="display:flex;gap:8px">
        <el-input v-model="remoteImportUrl" placeholder="输入远程镜像地址导入" style="flex:1" @keyup.enter="handleImportUrl" />
        <el-button type="primary" :loading="importingUrl" :disabled="!remoteImportUrl.trim()" @click="handleImportUrl">导入</el-button>
      </div>
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="showImportDialog = false">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useImageStore, type SortField } from '@/stores/images'
import BuildImageDialog from '@/components/image/BuildImageDialog.vue'
import request, { getToken } from '@/utils/request'
import registryApi from '@/api/registry'
import type { RegistryInfo } from '@/types'

const imageStore = useImageStore()
const searchInput = ref('')
const currentPageValue = ref(imageStore.currentPage)
const pageSizeValue = ref(imageStore.pageSize)
const showBuildDialog = ref(false)
const showPullDialog = ref(false)
const showImportDialog = ref(false)
const pullImage = ref('')
const pullArch = ref('')
const pullRegistry = ref('')
const remoteImportUrl = ref('')
const pulling = ref(false)
const pullProgress = ref(0)
const registries = ref<RegistryInfo[]>([])
const importingUrl = ref(false)

/** 上传附加参数：从文件名推断镜像 repo:tag（导出文件名规则 repo_tag.tar，/ 被浏览器转为 _） */
const uploadData = ref<{ name: string }>({ name: '' })
const uploadHeaders = computed(() => ({ Authorization: `Bearer ${getToken()}` }))

/** 从 tar 文件名推断镜像名，如 cqshushu_iptv-spider_latest.tar -> cqshushu/iptv-spider:latest */
function inferImageName(filename: string): string {
  const base = filename.replace(/\.tar$/i, '')
  const tagIdx = base.lastIndexOf('_')
  if (tagIdx <= 0) return ''
  const tag = base.slice(tagIdx + 1)
  let repo = base.slice(0, tagIdx)
  const slashIdx = repo.indexOf('_')
  if (slashIdx > 0) repo = repo.slice(0, slashIdx) + '/' + repo.slice(slashIdx + 1)
  if (!/^[a-z0-9]+([._-][a-z0-9]+)*(\/[a-z0-9]+([._-][a-z0-9]+)*)*$/.test(repo)) return ''
  if (!/^[\w][\w.-]{0,127}$/.test(tag)) return ''
  return `${repo}:${tag}`
}

function handleImportFileChange(file: any) {
  uploadData.value = { name: inferImageName(file?.name || '') }
}

function handleImportSuccess() {
  ElMessage.success('导入成功')
  imageStore.fetchImages(true)
}

function handleImportError() {
  ElMessage.error('导入失败，请确认文件为 docker save 导出的 .tar 包')
}

/** 从远程地址导入镜像 */
async function handleImportUrl() {
  const name = remoteImportUrl.value.trim()
  if (!name) return
  importingUrl.value = true
  try {
    await request.post('/images/pull', { name })
    ElMessage.success('导入完成')
    remoteImportUrl.value = ''
    showImportDialog.value = false
    imageStore.fetchImages(true)
  } catch { /* 错误已由拦截器提示 */ } finally {
    importingUrl.value = false
  }
}

/** 加载自定义镜像仓库列表 */
async function loadRegistries() {
  try {
    registries.value = await registryApi.list()
  } catch { /* 忽略，下拉仅显示 docker.io */ }
}

// Sync local state with store
watch(() => imageStore.searchQuery, (val) => { searchInput.value = val })
watch(() => imageStore.currentPage, (val) => { currentPageValue.value = val })
watch(() => imageStore.pageSize, (val) => { pageSizeValue.value = val })

// Handlers
function handleSearch(value: string) {
  imageStore.setSearch(value)
}

function handleSortChange({ prop, order }: { prop: string; order: string | null }) {
  if (prop && order) {
    const field = prop as SortField
    // Map Element Plus sort order to our format
    imageStore.sortField = field
    imageStore.sortOrder = order === 'ascending' ? 'asc' : 'desc'
  }
}

function handlePageChange(page: number) {
  imageStore.setPage(page)
}

function handlePageSizeChange(size: number) {
  imageStore.setPageSize(size)
}

async function handleRefresh() {
  await imageStore.fetchImages(true)
}

async function handleRemoveImage(id: string) {
  try {
    await imageStore.removeImage(id)
    ElMessage.success('删除成功')
  } catch {
    // Error is handled by request interceptor
  }
}

function handleImageBuilt() {
  imageStore.fetchImages(true)
}

async function handlePrune() {
  try {
    await ElMessageBox.confirm('将清理未使用的镜像缓存，是否继续？', '提示', { type: 'warning' })
  } catch { return }
  try {
    await request.post('/images/prune')
    ElMessage.success('清理完成')
    imageStore.fetchImages(true)
  } catch {}
}

async function handlePull() {
  if (!pullImage.value) return
  // 镜像名未自带仓库地址时，拼接所选仓库地址（去掉协议和尾斜杠）
  let name = pullImage.value.trim()
  if (pullRegistry.value && pullRegistry.value !== 'docker.io') {
    const host = pullRegistry.value.replace(/^https?:\/\//, '').replace(/\/+$/, '')
    const first = name.split('/')[0] ?? ''
    const hasRegistry = first.includes('.') || first.includes(':') || first === 'localhost'
    if (host && !hasRegistry) {
      name = `${host}/${name}`
    }
  }
  pulling.value = true
  pullProgress.value = 0
  try {
    await request.post('/images/pull', { name })
    ElMessage.success('拉取完成')
    showPullDialog.value = false
    pullImage.value = ''
    imageStore.fetchImages(true)
  } catch (e: any) {
    ElMessage.error('拉取失败：' + (e?.response?.data?.error || e?.message || ''))
  } finally {
    pulling.value = false
  }
}

async function handleExport(row: any) {
  try {
    await ElMessageBox.confirm('导出镜像文件后可通过面板导入恢复，是否确认？', '提示')
  } catch { return }
  const name = (row.name + '_' + row.tag).replace(/[^a-zA-Z0-9._-]/g, '_') || row.id.slice(0, 12)
  window.open('/api/images/export?id=' + row.id + '&name=' + encodeURIComponent(name), '_blank')
}

// Fetch images on mount
onMounted(() => {
  imageStore.fetchImages()
  loadRegistries()
})
</script>

<style scoped>
.images-page {
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

.filter-bar {
  display: flex;
  gap: 15px;
  margin-bottom: 20px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

.image-id {
  font-family: monospace;
  font-size: 12px;
}

/* Responsive styles */
@media (max-width: 767px) {
  .images-page {
    padding: 12px;
  }
  
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
  }
  
  .header-actions .el-button {
    flex: 1;
    min-width: 80px;
  }
  
  .filter-bar {
    flex-direction: column;
  }
  
  .filter-bar .el-input {
    width: 100% !important;
  }
  
  .pagination-wrapper {
    justify-content: center;
  }
}

/* Hide columns on smaller screens */
:deep(.hidden-xs-only) {
  @media (max-width: 767px) {
    display: none !important;
  }
}

:deep(.hidden-sm-and-down) {
  @media (max-width: 991px) {
    display: none !important;
  }
}

:deep(.hidden-md-and-down) {
  @media (max-width: 1199px) {
    display: none !important;
  }
}
</style>
