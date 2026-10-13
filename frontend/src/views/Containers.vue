<template>
  <div class="containers-page">
    <!-- Header with title and actions -->
    <div class="page-header">
      <h2>{{ t('container.title') }}</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog = true">
          <el-icon><Plus /></el-icon>
          {{ t('container.create') }}
        </el-button>
        <el-button @click="showDockerRunDialog = true">
          通过命令创建
        </el-button>
        <el-button :loading="containerStore.loading" @click="handleRefresh">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
      </div>
    </div>

    <!-- 批量操作栏 -->
    <div class="batch-bar">
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('start')">启动</el-button>
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('stop')">停止</el-button>
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('restart')">重启</el-button>
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('kill')">强制停止</el-button>
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('pause')">暂停</el-button>
      <el-button size="default" :disabled="selectedIds.length === 0" @click="batchAction('unpause')">恢复</el-button>
      <el-button size="default" type="danger" :disabled="selectedIds.length === 0" @click="batchAction('remove')">删除</el-button>
    </div>

    <!-- Search and filter bar -->
    <div class="filter-bar">
      <el-input
        v-model="searchInput"
        :placeholder="t('common.search')"
        clearable
        style="width: 300px"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <el-select
        v-model="statusFilterValue"
        :placeholder="t('common.status')"
        style="width: 150px"
        @change="handleStatusFilter"
      >
        <el-option :label="t('common.all')" value="all" />
        <el-option :label="t('container.running')" value="running" />
        <el-option :label="t('container.stopped')" value="exited" />
        <el-option :label="t('container.paused')" value="paused" />
        <el-option :label="t('container.created')" value="created" />
      </el-select>
    </div>

    <!-- Container table -->
    <el-table
      v-loading="containerStore.loading"
      :data="containerStore.paginatedContainers"
      stripe
      style="width: 100%"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="42" />
      <el-table-column
        prop="name"
        :label="t('common.name')"
        min-width="140"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="container-name">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column
        prop="ports"
        label="端口"
        min-width="180"
      >
        <template #default="{ row }">
          <span v-if="row.ports" class="port-tags">
            <a
              v-for="(p, i) in row.ports.split(', ')"
              :key="i"
              class="port-tag"
              :href="portUrl(p)"
              target="_blank"
              rel="noopener"
            >{{ p }}</a>
          </span>
          <span v-else class="cell-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column
        prop="image"
        label="镜像"
        min-width="200"
        show-overflow-tooltip
      />
      <el-table-column
        prop="description"
        label="描述"
        min-width="120"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span v-if="row.description" class="desc-text">{{ row.description }}</span>
          <span v-else class="cell-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column
        prop="state"
        label="状态"
        width="90"
      >
        <template #default="{ row }">
          <el-tag :type="getStateTagType(row.state)" size="small">
            {{ getStateLabel(row.state) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column
        label="CPU"
        width="85"
      >
        <template #default="{ row }">
          <template v-if="row.state === 'running'">
            <span v-if="loadingStats[row.id]">...</span>
            <span v-else-if="hasCpuStats(row.id)">{{ containerStats[row.id]!.cpu_percent.toFixed(1) }}%</span>
            <span v-else>-</span>
          </template>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column
        label="内存"
        min-width="120"
      >
        <template #default="{ row }">
          <template v-if="row.state === 'running'">
            <span v-if="loadingStats[row.id]">...</span>
            <template v-else-if="hasMemoryStats(row.id)">
              <span>{{ formatMemory(containerStats[row.id]!.memory_usage) }}</span>
              <span v-if="containerStats[row.id]!.has_memory_limit" class="memory-limit"> / {{ formatMemory(containerStats[row.id]!.memory_limit) }}</span>
              <span v-else class="memory-limit"> / 无限制</span>
            </template>
            <span v-else>-</span>
          </template>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column
        prop="status"
        label="运行时长"
        min-width="150"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ formatStatusCN(row.status) }}
        </template>
      </el-table-column>
      <el-table-column
        prop="created"
        label="创建时间"
        min-width="160"
        show-overflow-tooltip
      />
      <el-table-column
        :label="t('common.actions')"
        :width="isMobile ? 160 : 200"
        fixed="right"
        align="center"
      >
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="handleOpenTerminal(row.id, row.name)">{{ t('container.terminal') || '终端' }}</el-button>
          <el-button link type="primary" size="small" @click="handleViewLogs(row.id, row.name)">{{ t('container.logs') || '日志' }}</el-button>
          <el-button link type="primary" size="small" @click="handleOpenConfig(row.id, row.name)">{{ t('common.edit') || '编辑' }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPageValue"
        v-model:page-size="pageSizeValue"
        :page-sizes="[10, 20, 50, 100]"
        :total="containerStore.totalContainers"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>

    <!-- Create Container Dialog -->
    <CreateContainerDialog
      v-model:visible="showCreateDialog"
      @created="handleContainerCreated"
    />

    <!-- Docker Run Dialog -->
    <DockerRunDialog
      v-model:visible="showDockerRunDialog"
      @created="handleContainerCreated"
    />

    <!-- Container Logs Dialog -->
    <ContainerLogsDialog
      v-model:visible="showLogsDialog"
      :container-id="selectedContainerId"
      :container-name="selectedContainerName"
    />

    <!-- Container Terminal Dialog -->
    <ContainerTerminal
      v-model:visible="showTerminalDialog"
      :container-id="selectedContainerId"
      :container-name="selectedContainerName"
    />

    <!-- Container File Explorer Dialog -->
    <FileExplorer
      v-model:visible="showFilesDialog"
      :container-id="selectedContainerId"
      :container-name="selectedContainerName"
    />

    <!-- Container Config Dialog -->
    <ContainerConfig
      v-model:visible="showConfigDialog"
      :container-id="selectedContainerId"
      :container-name="selectedContainerName"
      @updated="handleContainerCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, onUnmounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useContainerStore, type ContainerState } from '@/stores/containers'
import { useI18n } from '@/composables/useI18n'
import { containerApi } from '@/api/containers'
import type { ContainerStats } from '@/types'
import CreateContainerDialog from '@/components/container/CreateContainerDialog.vue'
import DockerRunDialog from '@/components/container/DockerRunDialog.vue'
import ContainerLogsDialog from '@/components/container/ContainerLogsDialog.vue'
import ContainerTerminal from '@/components/container/ContainerTerminal.vue'
import FileExplorer from '@/components/container/FileExplorer.vue'
import ContainerConfig from '@/components/container/ContainerConfig.vue'

const { t } = useI18n()
const route = useRoute()
const containerStore = useContainerStore()

// 容器资源统计缓存
const containerStats = reactive<Record<string, ContainerStats | null>>({})
const loadingStats = reactive<Record<string, boolean>>({})

// 格式化内存大小
function formatMemory(bytes: number): string {
  if (bytes === 0) return '-'
  const mb = bytes / 1024 / 1024
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(1)} GB`
  }
  return `${mb.toFixed(0)} MB`
}

function hasCpuStats(containerId: string): boolean {
  const stats = containerStats[containerId]
  return !!stats && typeof stats.cpu_percent === 'number' && Number.isFinite(stats.cpu_percent)
}

function hasMemoryStats(containerId: string): boolean {
  const stats = containerStats[containerId]
  return !!stats && typeof stats.memory_usage === 'number' && Number.isFinite(stats.memory_usage)
}

// 从 "5244:5244" 取宿主机端口（冒号前）
function hostPortOf(portMapping: string): string {
  const idx = portMapping.indexOf(':')
  return idx > 0 ? portMapping.substring(0, idx) : portMapping
}

// 生成端口访问链接：当前主机 + 宿主机端口
function portUrl(portMapping: string): string {
  return `http://${window.location.hostname}:${hostPortOf(portMapping)}`
}

// 将 Docker 状态文本转为中文：Up 31 hours → 运行 31 小时
function formatStatusCN(status: string): string {
  if (!status) return '-'
  let s = status
  // Exited (143) 7 hours ago → 已停止 7 小时前（退出码 143）
  s = s.replace(/^Exited\s*\((\d+)\)\s*(.+?)\s*ago$/i, '已停止 $2 前（退出码 $1）')
  s = s.replace(/^Up\s+/i, '运行 ')
  s = s.replace(/(\d+)\s*days?/gi, '$1 天')
  s = s.replace(/(\d+)\s*hours?/gi, '$1 小时')
  s = s.replace(/(\d+)\s*minutes?/gi, '$1 分钟')
  s = s.replace(/(\d+)\s*seconds?/gi, '$1 秒')
  s = s.replace(/\(healthy\)/gi, '（健康）')
  return s
}

// 获取容器 stats
async function fetchContainerStats(containerId: string) {
  if (loadingStats[containerId]) return
  loadingStats[containerId] = true
  try {
    const stats = await containerApi.stats(containerId)
    containerStats[containerId] = stats
  } catch {
    containerStats[containerId] = null
  } finally {
    loadingStats[containerId] = false
  }
}

// 批量获取运行中容器的 stats
async function fetchAllRunningStats() {
  const runningContainers = containerStore.containers.filter(c => c.state === 'running')
  // 并行获取，但限制并发数
  const batchSize = 5
  for (let i = 0; i < runningContainers.length; i += batchSize) {
    const batch = runningContainers.slice(i, i + batchSize)
    await Promise.all(batch.map(c => fetchContainerStats(c.id)))
  }
}

// 监听容器列表变化，自动获取 stats
watch(() => containerStore.containers, () => {
  fetchAllRunningStats()
}, { immediate: true })

// Responsive state
const windowWidth = ref(window.innerWidth)
const isMobile = computed(() => windowWidth.value < 768)

function handleResize() {
  windowWidth.value = window.innerWidth
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

// Local state for v-model bindings
const searchInput = ref('')
const statusFilterValue = ref<ContainerState>(containerStore.statusFilter)
const currentPageValue = ref(containerStore.currentPage)
const pageSizeValue = ref(containerStore.pageSize)

// Dialog visibility
const showCreateDialog = ref(false)
const showDockerRunDialog = ref(false)
const showLogsDialog = ref(false)
const showTerminalDialog = ref(false)
const showFilesDialog = ref(false)
const showConfigDialog = ref(false)

// Selected container for logs
const selectedContainerId = ref('')
const selectedContainerName = ref('')

// 批量操作选中的容器
const selectedIds = ref<string[]>([])

function handleSelectionChange(rows: { id: string }[]) {
  selectedIds.value = rows.map((r) => r.id)
}

async function batchAction(action: 'start' | 'stop' | 'restart' | 'remove' | 'pause' | 'unpause' | 'kill') {
  if (selectedIds.value.length === 0) return
  const ids = [...selectedIds.value]
  for (const id of ids) {
    try {
      await containerStore.containerAction(id, action)
    } catch {
      // 单个失败继续
    }
  }
  selectedIds.value = []
  ElMessage.success(t('common.success'))
  await containerStore.fetchContainers()
}

// Sync local state with store
watch(() => containerStore.searchQuery, (val) => { searchInput.value = val })
watch(() => containerStore.statusFilter, (val) => { statusFilterValue.value = val })
watch(() => containerStore.currentPage, (val) => { currentPageValue.value = val })
watch(() => containerStore.pageSize, (val) => { pageSizeValue.value = val })

// Handlers
function handleSearch(value: string) {
  containerStore.setSearch(value)
}

function handleStatusFilter(value: ContainerState) {
  containerStore.setStatusFilter(value)
}

function handlePageChange(page: number) {
  containerStore.setPage(page)
}

function handlePageSizeChange(size: number) {
  containerStore.setPageSize(size)
}

async function handleRefresh() {
  await containerStore.fetchContainers()
}

function handleViewLogs(id: string, name: string) {
  selectedContainerId.value = id
  selectedContainerName.value = name
  showLogsDialog.value = true
}

function handleOpenTerminal(id: string, name: string) {
  selectedContainerId.value = id
  selectedContainerName.value = name
  showTerminalDialog.value = true
}

function handleOpenConfig(id: string, name: string) {
  selectedContainerId.value = id
  selectedContainerName.value = name
  showConfigDialog.value = true
}

function handleContainerCreated() {
  containerStore.fetchContainers()
}

// Get tag type based on container state
function getStateTagType(state: string): 'success' | 'danger' | 'warning' | 'info' {
  switch (state) {
    case 'running':
      return 'success'
    case 'exited':
      return 'danger'
    case 'paused':
      return 'warning'
    default:
      return 'info'
  }
}

// Get localized state label
function getStateLabel(state: string): string {
  switch (state) {
    case 'running':
      return t('container.running')
    case 'exited':
      return t('container.stopped')
    case 'paused':
      return t('container.paused')
    case 'created':
      return t('container.created')
    default:
      return state
  }
}

// Fetch containers on mount and handle query params
onMounted(async () => {
  await containerStore.fetchContainers()
  
  // 处理从 Compose 页面跳转过来的操作
  const action = route.query.action as string
  const containerName = route.query.container as string
  
  if (action && containerName) {
    // 等待容器列表加载完成后查找容器
    const container = containerStore.containers.find(c => c.name === containerName)
    if (container) {
      selectedContainerId.value = container.id
      selectedContainerName.value = container.name
      switch (action) {
        case 'terminal':
          showTerminalDialog.value = true
          break
        case 'files':
          showFilesDialog.value = true
          break
        case 'logs':
          showLogsDialog.value = true
          break
      }
    } else {
      ElMessage.warning(`容器 ${containerName} 未找到`)
    }
  }
})
</script>

<style scoped>
.containers-page {
  padding: 12px 20px 20px;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.containers-page :deep(.el-table) {
  flex: 1;
  min-height: 0;
}

.page-header,
.batch-bar,
.filter-bar,
.pagination-wrapper {
  flex-shrink: 0;
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

.batch-bar {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 12px 14px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.container-name {
  color: var(--el-color-primary);
  font-weight: 500;
}

.port-tags {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
}
.port-tag {
  display: inline-block;
  padding: 1px 8px;
  font-size: 12px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
  text-decoration: none;
}
.port-tag:hover {
  background: var(--el-color-primary-light-7);
}
.cell-muted { color: #c0c4cc; }
.desc-text { color: #606266; }

/* dpanel 风格表头 + 更大更深的字体 */
:deep(.el-table) {
  font-size: 14px;
  color: #303133;
}
:deep(.el-table th.el-table__cell) {
  background-color: #f5f7fa !important;
  color: #303133;
  font-weight: 600;
  font-size: 14px;
}
:deep(.el-table .cell) {
  line-height: 1.7;
  padding-top: 14px;
  padding-bottom: 14px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

.memory-limit {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

/* Responsive styles */
@media (max-width: 767px) {
  .containers-page {
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
  
  .filter-bar .el-input,
  .filter-bar .el-select {
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
