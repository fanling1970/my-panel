<template>
  <div class="volumes-page">
    <!-- Header with title and actions -->
    <div class="page-header">
      <h2>存储卷管理</h2>
      <div class="header-actions">
        <el-button @click="handlePrune">自动清理</el-button>
        <el-button type="primary" @click="showCreateDialog = true">创建存储卷</el-button>
        <el-button :loading="volumeStore.loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <!-- Search bar -->
    <div class="filter-bar">
      <el-input
        v-model="searchInput"
        placeholder="搜索"
        clearable
        style="width: 300px"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <!-- Volume table -->
    <el-table
      v-loading="volumeStore.loading"
      :data="volumeStore.paginatedVolumes"
      stripe
      style="width: 100%"
    >
      <el-table-column prop="name" label="卷名称" min-width="200">
        <template #default="{ row }"><span class="volume-name">{{ row.name }}</span></template>
      </el-table-column>
      <el-table-column prop="driver" label="驱动" width="100">
        <template #default="{ row }"><el-tag size="small" type="primary">{{ row.driver }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="mountpoint" label="挂载点" min-width="250">
        <template #default="{ row }"><span class="mountpoint">{{ row.mountpoint }}</span></template>
      </el-table-column>
      <el-table-column label="使用者" width="150">
        <template #default="{ row }">
          <span v-if="row.containers && row.containers.length > 0">{{ row.containers.join(', ') }}</span>
          <span v-else style="color:#909399">未使用</span>
        </template>
      </el-table-column>
      <el-table-column prop="created" label="创建时间" width="180">
        <template #default="{ row }"><span>{{ formatDate(row.created) }}</span></template>
      </el-table-column>
      <el-table-column label="大小" width="100">
        <template #default="{ row }"><span>{{ row.size || '-' }}</span></template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-popconfirm v-if="!row.containers || row.containers.length === 0" title="确定删除？" @confirm="handleRemove(row.name)">
            <template #reference><el-button type="danger" size="small" link>删除</el-button></template>
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
        :total="volumeStore.totalVolumes"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>

    <!-- Create Volume Dialog -->
    <CreateVolumeDialog
      v-model:visible="showCreateDialog"
      @created="handleVolumeCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useVolumeStore } from '@/stores/volumes'
import CreateVolumeDialog from '@/components/volumes/CreateVolumeDialog.vue'

const volumeStore = useVolumeStore()

// Local state for v-model bindings
const searchInput = ref('')
const currentPageValue = ref(volumeStore.currentPage)
const pageSizeValue = ref(volumeStore.pageSize)

// Dialog visibility
const showCreateDialog = ref(false)

// Sync local state with store
watch(() => volumeStore.searchQuery, (val) => { searchInput.value = val })
watch(() => volumeStore.currentPage, (val) => { currentPageValue.value = val })
watch(() => volumeStore.pageSize, (val) => { pageSizeValue.value = val })

// Format date helper
function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString()
}

// Handlers
function handleSearch(value: string) {
  volumeStore.setSearch(value)
}

function handlePageChange(page: number) {
  volumeStore.setPage(page)
}

function handlePageSizeChange(size: number) {
  volumeStore.setPageSize(size)
}

async function handleRefresh() {
  await volumeStore.fetchVolumes()
}

async function handleRemove(name: string) {
  try {
    await volumeStore.removeVolume(name)
    ElMessage.success('删除成功')
  } catch {
    // Error is handled by request interceptor
  }
}

async function handlePrune() {
  try {
    await ElMessageBox.confirm('将删除所有未使用的存储卷，是否确认？', '自动清理')
    const result = await volumeStore.pruneVolumes()
    const count = result.volumesDeleted?.length || 0
    ElMessage.success(`清理完成，删除 ${count} 个卷`)
  } catch {}
}

function handleVolumeCreated() {
  volumeStore.fetchVolumes()
}

// Fetch volumes on mount
onMounted(() => {
  volumeStore.fetchVolumes()
})
</script>

<style scoped>
.volumes-page {
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

.volume-name {
  font-family: monospace;
  font-size: 13px;
}

.mountpoint {
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* Responsive styles */
@media (max-width: 767px) {
  .volumes-page {
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
