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

    <!-- 拉取镜像 -->
    <el-dialog v-model="showPullDialog" title="拉取远程镜像" width="600px">
      <el-input v-model="pullImage" placeholder="请输入镜像地址，如 nginx:latest" style="margin-bottom:12px" />
      <el-progress v-if="pulling" :percentage="pullProgress" style="margin-bottom:12px" />
      <template #footer>
        <el-button @click="showPullDialog = false">取消</el-button>
        <el-button type="primary" :loading="pulling" @click="handlePull">拉取镜像</el-button>
      </template>
    </el-dialog>

    <!-- 导入镜像 -->
    <el-dialog v-model="showImportDialog" title="导入镜像" width="600px">
      <el-upload action="/api/images/import" :show-file-list="true" accept=".tar">
        <el-button type="primary">选择 tar 文件上传</el-button>
      </el-upload>
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="showImportDialog = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useImageStore, type SortField } from '@/stores/images'
import BuildImageDialog from '@/components/image/BuildImageDialog.vue'
import request from '@/utils/request'

const imageStore = useImageStore()
const searchInput = ref('')
const currentPageValue = ref(imageStore.currentPage)
const pageSizeValue = ref(imageStore.pageSize)
const showBuildDialog = ref(false)
const showPullDialog = ref(false)
const showImportDialog = ref(false)
const pullImage = ref('')
const pulling = ref(false)
const pullProgress = ref(0)

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
  pulling.value = true
  pullProgress.value = 0
  try {
    await request.post('/images/pull', { name: pullImage.value })
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
  window.open('/api/images/export?id=' + row.id, '_blank')
}

// Fetch images on mount
onMounted(() => {
  imageStore.fetchImages()
})
</script>

<style scoped>
.images-page {
  padding: 20px;
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
