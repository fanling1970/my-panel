<template>
  <div class="networks-page">
    <!-- Header with title and actions -->
    <div class="page-header">
      <h2>网络管理</h2>
      <div class="header-actions">
        <el-button @click="handlePrune">自动清理</el-button>
        <el-button type="primary" @click="showCreateDialog = true">创建网络</el-button>
        <el-button :loading="networkStore.loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <!-- Search bar -->
    <div class="filter-bar">
      <el-input v-model="searchInput" placeholder="搜索" clearable style="width: 300px" @input="handleSearch">
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
    </div>

    <!-- Network table -->
    <el-table v-loading="networkStore.loading" :data="networkStore.paginatedNetworks" stripe style="width: 100%">
      <el-table-column label="名称" min-width="180">
        <template #default="{ row }">
          <el-link type="primary">{{ row.name }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="driver" label="类型" width="100">
        <template #default="{ row }"><span>{{ row.driver }}</span></template>
      </el-table-column>
      <el-table-column label="驱动" width="100">
        <template #default="{ row }"><el-tag size="small" type="primary">{{ row.driver }}</el-tag></template>
      </el-table-column>
      <el-table-column label="IPAM Driver" width="120">
        <template #default><span>default</span></template>
      </el-table-column>
      <el-table-column label="IPv4" min-width="160">
        <template #default="{ row }">
          <div v-if="row.subnet">{{ row.subnet }}<br/>{{ row.gateway || '' }}</div>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="IPv6" width="80">
        <template #default><span>-</span></template>
      </el-table-column>
      <el-table-column label="创建时间" width="180">
        <template #default="{ row }"><span>{{ row.created || '-' }}</span></template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="handleViewDetails(row)">网络详情</el-button>
          <el-popconfirm v-if="!networkStore.isProtectedNetwork(row.name)" title="确定删除？" @confirm="handleRemoveNetwork(row.id)">
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
        :total="networkStore.totalNetworks"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>

    <!-- Create Network Dialog -->
    <CreateNetworkDialog
      v-model:visible="showCreateDialog"
      @created="handleNetworkCreated"
    />

    <!-- Network Detail Dialog -->
    <NetworkDetailDialog
      v-model:visible="showDetailDialog"
      :network-id="selectedNetworkId"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useNetworkStore } from '@/stores/networks'
import request from '@/utils/request'
import type { NetworkInfo } from '@/types'
import CreateNetworkDialog from '@/components/network/CreateNetworkDialog.vue'
import NetworkDetailDialog from '@/components/network/NetworkDetailDialog.vue'

const networkStore = useNetworkStore()

// Local state for v-model bindings
const searchInput = ref('')
const currentPageValue = ref(networkStore.currentPage)
const pageSizeValue = ref(networkStore.pageSize)

// Dialog visibility
const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
const selectedNetworkId = ref('')

// Sync local state with store
watch(() => networkStore.searchQuery, (val) => { searchInput.value = val })
watch(() => networkStore.currentPage, (val) => { currentPageValue.value = val })
watch(() => networkStore.pageSize, (val) => { pageSizeValue.value = val })

// Handlers
function handleSearch(value: string) {
  networkStore.setSearch(value)
}

function handlePageChange(page: number) {
  networkStore.setPage(page)
}

function handlePageSizeChange(size: number) {
  networkStore.setPageSize(size)
}

async function handleRefresh() {
  await networkStore.fetchNetworks()
}

async function handlePrune() {
  try {
    await ElMessageBox.confirm('将删除所有未使用的网络，是否确认？', '自动清理')
    await request.post('/api/networks/prune')
    ElMessage.success('清理完成')
    networkStore.fetchNetworks()
  } catch {}
}

async function handleRemoveNetwork(id: string) {
  try {
    await networkStore.removeNetwork(id)
    ElMessage.success('删除成功')
  } catch {
    // Error is handled by request interceptor
  }
}

function handleViewDetails(network: NetworkInfo) {
  selectedNetworkId.value = network.id
  showDetailDialog.value = true
}

function handleNetworkCreated() {
  networkStore.fetchNetworks()
}

// Fetch networks on mount
onMounted(() => {
  networkStore.fetchNetworks()
})
</script>

<style scoped>
.networks-page {
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

.network-id {
  font-family: monospace;
  font-size: 12px;
}

/* Responsive styles */
@media (max-width: 767px) {
  .networks-page {
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
