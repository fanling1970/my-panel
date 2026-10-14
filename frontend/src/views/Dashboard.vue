<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSystemStore } from '@/stores/system'
import { useContainerStore } from '@/stores/containers'
import { useComposeStore } from '@/stores/compose'
import { useImageStore } from '@/stores/images'
import { useNetworkStore } from '@/stores/networks'
import { useVolumeStore } from '@/stores/volumes'
import { containerApi } from '@/api/containers'
import type { ContainerStats } from '@/types'
import UpdateBanner from '@/components/common/UpdateBanner.vue'
import UpdateDialog from '@/components/common/UpdateDialog.vue'

const systemStore = useSystemStore()
const containerStore = useContainerStore()
const composeStore = useComposeStore()
const imagesStore = useImageStore()
const networkStore = useNetworkStore()
const volumeStore = useVolumeStore()

const showUpdateDialog = ref(false)

// 容器更新检测
const checking = ref(false)
const checkDialogVisible = ref(false)
const checkTotal = ref(0)
const checkCurrent = ref(0)
const checkContainerName = ref('')
const checkImageName = ref('')
const updateResults = ref<Array<{ container_id: string; name: string; image: string; has_update: boolean }>>([])
const updateAvailableCount = computed(() => updateResults.value.filter(r => r.has_update).length)
const pendingCheckCount = computed(() => updateResults.value.length === 0 ? containerStore.containers.length : 0)

async function handleCheckUpdates() {
  try {
    await ElMessageBox.confirm(
      '系统将依次查询容器镜像的远端仓库，并与本地镜像摘要进行比对。检测期间可以离开当前页面；请保持当前标签页打开，不要刷新，否则剩余容器将不会继续检测。',
      '提示',
      { confirmButtonText: '是', cancelButtonText: '否', type: 'warning' },
    )
  } catch {
    return
  }
  checking.value = true
  updateResults.value = []
  checkCurrent.value = 0
  checkTotal.value = 0
  checkContainerName.value = ''
  checkImageName.value = ''
  checkDialogVisible.value = true

  const { getToken } = await import('@/utils/request')
  const token = getToken()
  const es = new EventSource(`/api/containers/check-updates?token=${encodeURIComponent(token || '')}`)
  es.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data)
      if (data.type === 'start') {
        checkTotal.value = data.total
      } else if (data.type === 'progress') {
        checkCurrent.value = data.current
        checkContainerName.value = data.container_name
        checkImageName.value = data.image
      } else if (data.type === 'result') {
        updateResults.value.push(data)
        containerStore.updateMap[data.container_id] = data.has_update
      } else if (data.type === 'done') {
        checking.value = false
        es.close()
        ElMessage.warning(`检测完成：${data.update_count} 个容器有可用更新`)
      } else if (data.type === 'error') {
        checking.value = false
        es.close()
        ElMessage.error('检测失败：' + data.message)
      }
    } catch { /* ignore */ }
  }
  es.onerror = () => {
    checking.value = false
    es.close()
  }
}

// 容器资源统计缓存
const containerStats = reactive<Record<string, ContainerStats | null>>({})

function formatSize(kb: number): string {
  if (!kb || kb === 0) return '0 B'
  const gb = kb / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(2)} GB`
  const mb = kb / 1024
  if (mb >= 1) return `${mb.toFixed(1)} MB`
  return `${kb.toFixed(0)} KB`
}

function formatUptime(seconds: number): string {
  if (!seconds || seconds <= 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}天 ${h}小时`
  if (h > 0) return `${h}小时 ${m}分`
  return `${m}分`
}

// CPU / 内存 / 磁盘 百分比
const cpuPercent = computed(() => Number(systemStore.stats.cpu.toFixed(2)))
const memPercent = computed(() => Number(systemStore.stats.memory.toFixed(2)))
const diskPercent = computed(() => Number(systemStore.stats.disk.toFixed(2)))

const memUsedText = computed(() => formatSize(systemStore.stats.memoryUsed))
const memTotalText = computed(() => formatSize(systemStore.stats.memoryTotal))
const diskUsedText = computed(() => formatSize(systemStore.stats.diskUsed))
const diskTotalText = computed(() => formatSize(systemStore.stats.diskTotal))
const cpuCores = computed(() => systemStore.stats.cpuCores || 0)

// 容器状态统计
const runningCount = computed(() => containerStore.containers.filter(c => c.state === 'running').length)
const exitedCount = computed(() => containerStore.containers.filter(c => c.state === 'exited').length)
const pausedCount = computed(() => containerStore.containers.filter(c => c.state === 'paused').length)

// Compose 项目统计
const composeTotal = computed(() => composeStore.projects.length)
const composeRunning = computed(() => composeStore.projects.filter(p => p.status === 'running').length)
const composeStopped = computed(() => composeStore.projects.filter(p => p.status === 'stopped').length)
const composePartial = computed(() => composeStore.projects.filter(p => p.status === 'partial').length)

// 镜像统计
const imageTotal = computed(() => imagesStore.images.length)
const imageUnused = computed(() => imagesStore.images.filter(i => !i.in_use).length)

// 网络统计
const networkTotal = computed(() => networkStore.networks.length)
const networkUnused = computed(() => networkStore.networks.filter(n => !n.in_use).length)

// 存储卷统计
const volumeTotal = computed(() => volumeStore.volumes.length)
const volumeUnused = computed(() => volumeStore.volumes.filter(v => !v.in_use).length)

// Docker 容器资源占用合计（所有运行中容器 stats 求和）
const dockerMemUsage = computed(() => {
  let sum = 0
  for (const id in containerStats) {
    const s = containerStats[id]
    if (s && typeof s.memory_usage === 'number') sum += s.memory_usage
  }
  return sum // bytes
})
const dockerCpuUsage = computed(() => {
  let sum = 0
  for (const id in containerStats) {
    const s = containerStats[id]
    if (s && typeof s.cpu_percent === 'number') sum += s.cpu_percent
  }
  return Number(sum.toFixed(2))
})

function formatBytesToReadable(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 MB'
  const mb = bytes / 1024 / 1024
  if (mb >= 1024) return `${(mb / 1024).toFixed(2)} GB`
  return `${mb.toFixed(1)} MB`
}

// 容器资源排行（按 CPU 排序，取前 8）
const topContainers = computed(() => {
  return containerStore.containers
    .filter(c => c.state === 'running')
    .map(c => ({
      id: c.id,
      name: c.name,
      cpu: containerStats[c.id]?.cpu_percent ?? 0,
      mem: containerStats[c.id]?.memory_usage ?? 0,
    }))
    .sort((a, b) => b.cpu - a.cpu)
    .slice(0, 8)
})

async function fetchAllStats() {
  const running = containerStore.containers.filter(c => c.state === 'running')
  for (const c of running) {
    try {
      containerStats[c.id] = await containerApi.stats(c.id)
    } catch {
      containerStats[c.id] = null
    }
  }
}

let timer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  systemStore.fetchStats()
  await Promise.all([
    containerStore.fetchContainers(),
    composeStore.fetchProjects(),
    imagesStore.fetchImages(),
    networkStore.fetchNetworks(),
    volumeStore.fetchVolumes(),
  ])
  await fetchAllStats()
  timer = setInterval(() => {
    systemStore.fetchStats()
    fetchAllStats()
  }, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="dashboard-page">
    <UpdateBanner @click="showUpdateDialog = true" />
    <UpdateDialog v-model="showUpdateDialog" />

    <!-- 检测更新进度弹窗 -->
    <el-dialog v-model="checkDialogVisible" title="检测更新" width="560px" :close-on-click-modal="false" :close-on-press-escape="false">
      <div style="margin-bottom: 8px; color: #606262;">
        当前检测 <b>{{ checkCurrent }}</b> / {{ checkTotal }}
      </div>
      <div style="background: #f5f7fa; border-radius: 6px; padding: 14px; margin-bottom: 16px;">
        <div style="margin-bottom: 6px;">容器名称：<b>{{ checkContainerName || '-' }}</b></div>
        <div>镜像名称：<span style="color:#606262">{{ checkImageName || '-' }}</span></div>
      </div>
      <div style="margin-bottom: 6px;">检测更新 <b>{{ checkTotal ? Math.round(checkCurrent / checkTotal * 100) : 0 }}%</b></div>
      <el-progress :percentage="checkTotal ? Math.round(checkCurrent / checkTotal * 100) : 0" />
    </el-dialog>

    <div class="dash-grid">
      <!-- 左侧主区 -->
      <div class="main-col">
        <!-- 三个环形图 -->
        <div class="rings-row">
          <div class="ring-card">
            <div class="ring-title">CPU 使用</div>
            <div class="ring-body">
              <svg class="ring-svg" viewBox="0 0 120 120">
                <circle cx="60" cy="60" r="50" class="ring-bg" />
                <circle cx="60" cy="60" r="50" class="ring-fg ring-cpu" :stroke-dasharray="`${(cpuPercent / 100) * 314} 314`" />
              </svg>
              <div class="ring-center">
                <div class="ring-value">{{ cpuPercent.toFixed(2) }}%</div>
                <div class="ring-label">主机</div>
              </div>
            </div>
            <div class="ring-legend">
              <div class="legend-row"><span class="dot dot-blue"></span><span>主机</span><span class="legend-val">{{ cpuCores }} 核</span></div>
              <div class="legend-row"><span class="dot dot-orange"></span><span>Docker</span><span class="legend-val">{{ dockerCpuUsage.toFixed(2) }} 核</span></div>
              <div class="legend-row"><span class="dot dot-gray"></span><span>总量</span><span class="legend-val">{{ cpuCores }} 核</span></div>
            </div>
          </div>

          <div class="ring-card">
            <div class="ring-title">内存使用</div>
            <div class="ring-body">
              <svg class="ring-svg" viewBox="0 0 120 120">
                <circle cx="60" cy="60" r="50" class="ring-bg" />
                <circle cx="60" cy="60" r="50" class="ring-fg ring-mem" :stroke-dasharray="`${(memPercent / 100) * 314} 314`" />
              </svg>
              <div class="ring-center">
                <div class="ring-value">{{ memPercent.toFixed(2) }}%</div>
                <div class="ring-label">主机</div>
              </div>
            </div>
            <div class="ring-legend">
              <div class="legend-row"><span class="dot dot-blue"></span><span>主机</span><span class="legend-val">{{ memUsedText }}</span></div>
              <div class="legend-row"><span class="dot dot-orange"></span><span>Docker</span><span class="legend-val">{{ formatBytesToReadable(dockerMemUsage) }}</span></div>
              <div class="legend-row"><span class="dot dot-gray"></span><span>总量</span><span class="legend-val">{{ memTotalText }}</span></div>
            </div>
          </div>

          <div class="ring-card">
            <div class="ring-title">磁盘使用</div>
            <div class="ring-body">
              <svg class="ring-svg" viewBox="0 0 120 120">
                <circle cx="60" cy="60" r="50" class="ring-bg" />
                <circle cx="60" cy="60" r="50" class="ring-fg ring-disk" :stroke-dasharray="`${(diskPercent / 100) * 314} 314`" />
              </svg>
              <div class="ring-center">
                <div class="ring-value">{{ diskPercent.toFixed(2) }}%</div>
                <div class="ring-label">主机</div>
              </div>
            </div>
            <div class="ring-legend">
              <div class="legend-row"><span class="dot dot-blue"></span><span>主机</span><span class="legend-val">{{ diskUsedText }}</span></div>
              <div class="legend-row"><span class="dot dot-orange"></span><span>Docker</span><span class="legend-val">{{ formatSize(0) }}</span></div>
              <div class="legend-row"><span class="dot dot-gray"></span><span>总量</span><span class="legend-val">{{ diskTotalText }}</span></div>
            </div>
          </div>
        </div>

        <!-- 资源统计卡片 -->
        <div class="stat-cards">
          <div class="stat-card">
            <div class="stat-label">容器</div>
            <div class="stat-num">{{ containerStore.containers.length }}</div>
            <div class="stat-sub">
              <span>运行中 <b class="ok">{{ runningCount }}</b></span>
              <span>已停止 <b class="stop">{{ exitedCount }}</b></span>
              <span>已暂停 <b class="pause">{{ pausedCount }}</b></span>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-label">容器更新</div>
            <div class="stat-num" style="color:#e6a23c">{{ updateAvailableCount }}</div>
            <div class="stat-sub">
              <span>待更新 <b class="warn">{{ updateAvailableCount }}</b></span>
              <span>待检查 <b class="muted">{{ pendingCheckCount }}</b></span>
            </div>
            <div style="margin-top:8px">
              <el-button link type="primary" :loading="checking" @click="handleCheckUpdates">
                {{ checking ? '检测中...' : '检测更新' }}
              </el-button>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-label">Compose</div>
            <div class="stat-num">{{ composeTotal }}</div>
            <div class="stat-sub">
              <span>运行中 <b class="ok">{{ composeRunning }}</b></span>
              <span>已停止 <b class="stop">{{ composeStopped }}</b></span>
              <span>部分 <b class="pause">{{ composePartial }}</b></span>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-label">镜像</div>
            <div class="stat-num">{{ imageTotal }}</div>
            <div class="stat-sub">
              <span>未使用 <b class="muted">{{ imageUnused }}</b></span>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-label">网络</div>
            <div class="stat-num">{{ networkTotal }}</div>
            <div class="stat-sub">
              <span>未使用 <b class="muted">{{ networkUnused }}</b></span>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-label">存储卷</div>
            <div class="stat-num">{{ volumeTotal }}</div>
            <div class="stat-sub">
              <span>未使用 <b class="muted">{{ volumeUnused }}</b></span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧栏 -->
      <div class="side-col">
        <div class="info-card">
          <div class="card-title">系统信息</div>
          <div class="info-row"><span>主机名称</span><b>{{ systemStore.stats.hostname || '-' }}</b></div>
          <div class="info-row"><span>发行版本</span><b>{{ systemStore.stats.os || '-' }}</b></div>
          <div class="info-row"><span>内核版本</span><b>{{ systemStore.stats.kernel || '-' }}</b></div>
          <div class="info-row"><span>系统类型</span><b>{{ systemStore.stats.arch || '-' }}</b></div>
          <div class="info-row"><span>CPU 核数</span><b>{{ cpuCores }}</b></div>
          <div class="info-row"><span>运行时间</span><b>{{ formatUptime(systemStore.stats.uptime) }}</b></div>
          <div class="info-row"><span>服务器时间</span><b>{{ systemStore.stats.time || '-' }}</b></div>
        </div>

        <div class="info-card">
          <div class="card-title">容器资源排行</div>
          <div class="rank-list" v-if="topContainers.length">
            <div class="rank-item" v-for="c in topContainers" :key="c.id">
              <span class="rank-name">{{ c.name }}</span>
              <span class="rank-val">{{ c.cpu.toFixed(2) }}%</span>
            </div>
          </div>
          <div class="empty" v-else>暂无运行中容器</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-page {
  padding: 20px;
}
.dash-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 16px;
}
.rings-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.ring-card, .info-card, .stat-card {
  background: #fff;
  border-radius: 10px;
  padding: 18px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}
.ring-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 10px;
}
.ring-body {
  position: relative;
  width: 150px;
  margin: 0 auto;
}
.ring-svg {
  width: 150px;
  height: 150px;
  transform: rotate(-90deg);
}
.ring-bg {
  fill: none;
  stroke: #eef1f4;
  stroke-width: 10;
}
.ring-fg {
  fill: none;
  stroke-width: 10;
  stroke-linecap: round;
  transition: stroke-dasharray 0.6s ease;
}
.ring-cpu { stroke: #409eff; }
.ring-mem { stroke: #67c23a; }
.ring-disk { stroke: #e6a23c; }
.ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.ring-value {
  font-size: 24px;
  font-weight: 700;
}
.ring-label {
  font-size: 12px;
  color: #909399;
}
.ring-legend {
  margin-top: 12px;
}
.legend-row {
  display: flex;
  align-items: center;
  font-size: 13px;
  padding: 3px 0;
  color: #606266;
}
.legend-val {
  margin-left: auto;
  color: #303133;
  font-weight: 500;
}
.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 8px;
}
.dot-blue { background: #409eff; }
.dot-orange { background: #e6a23c; }
.dot-gray { background: #909399; }

.stat-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 16px;
  margin-top: 16px;
}
.stat-label { font-size: 13px; color: #909399; }
.stat-num { font-size: 28px; font-weight: 700; margin: 6px 0; }
.stat-sub { font-size: 12px; display: flex; gap: 10px; flex-wrap: wrap; color: #909399; }
.stat-sub b { font-weight: 600; }
.stat-sub .ok { color: #67c23a; }
.stat-sub .stop { color: #f56c6c; }
.stat-sub .pause { color: #e6a23c; }
.stat-sub .muted { color: #909399; }

.card-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
  padding-left: 8px;
  border-left: 3px solid #409eff;
}
.info-card { margin-bottom: 16px; }
.info-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  padding: 6px 0;
  color: #606266;
}
.info-row b { color: #303133; font-weight: 500; }
.rank-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  padding: 5px 0;
}
.rank-name { color: #606266; }
.rank-val { color: #409eff; font-weight: 500; }
.empty { font-size: 13px; color: #909399; text-align: center; padding: 16px 0; }

@media (max-width: 1100px) {
  .dash-grid { grid-template-columns: 1fr; }
  .rings-row { grid-template-columns: 1fr; }
  .stat-cards { grid-template-columns: repeat(2, 1fr); }
}
</style>
