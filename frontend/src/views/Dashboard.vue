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
import request from '@/utils/request'
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

// Docker 信息（版本、容器数、镜像数等）
const dockerInfo = ref<any>({})

// 容器更新检测
const checking = computed(() => containerStore.updateChecking)
const checkDialogVisible = ref(false)
const checkTotal = ref(0)
const checkCurrent = ref(0)
const checkContainerName = ref('')
const checkImageName = ref('')
const updateAvailableCount = computed(() => containerStore.updateResults.filter(r => r.has_update).length)

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
  containerStore.updateChecking = true
  containerStore.updateResults = []
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
        containerStore.updateResults.push(data)
        containerStore.updateMap[data.container_id] = data.has_update
      } else if (data.type === 'done') {
        containerStore.updateChecking = false
        es.close()
        ElMessage.warning(`检测完成：${data.update_count} 个容器有可用更新`)
      } else if (data.type === 'error') {
        containerStore.updateChecking = false
        es.close()
        ElMessage.error('检测失败：' + data.message)
      }
    } catch { /* ignore */ }
  }
  es.onerror = () => {
    containerStore.updateChecking = false
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

async function fetchDockerInfo() {
  try {
    const res = await request.get('/docker/info', { suppressErrorMessage: true })
    dockerInfo.value = res.data || {}
  } catch {
    dockerInfo.value = {}
  }
}

let timer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  systemStore.fetchStats()
  fetchDockerInfo()
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
  }, 5000)
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

        <!-- 资源统计卡片：两行三列 -->
        <div class="stat-cards">
          <div class="stat-card">
            <div class="stat-label">容器</div>
            <div class="stat-num">{{ dockerInfo.containers ?? containerStore.containers.length }}</div>
            <div class="stat-sub">
              <span>运行中 <b class="ok">{{ dockerInfo.containersRunning ?? runningCount }}</b></span>
              <span>已停止 <b class="stop">{{ dockerInfo.containersStopped ?? exitedCount }}</b></span>
              <span>已暂停 <b class="pause">{{ dockerInfo.containersPaused ?? pausedCount }}</b></span>
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
            <div class="stat-num">{{ dockerInfo.images ?? imageTotal }}</div>
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
          <div class="stat-card">
            <div class="stat-label">容器更新</div>
            <div class="stat-num" style="color:#e6a23c">{{ updateAvailableCount }}</div>
            <div class="stat-sub">
              <span>待更新 <b class="warn">{{ updateAvailableCount }}</b></span>
            </div>
            <div style="margin-top:8px">
              <el-button link type="primary" :loading="checking" @click="handleCheckUpdates">
                {{ checking ? '检测中...' : '检测更新' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧栏 -->
      <div class="side-col">
        <div class="info-card">
          <div class="card-title">Docker 信息</div>
          <div class="info-row"><span>Docker 版本</span><b>{{ dockerInfo.serverVersion || '-' }}</b></div>
          <div class="info-row"><span>API 版本</span><b>{{ dockerInfo.apiVersion || '-' }}</b></div>
          <div class="info-row"><span>操作系统</span><b>{{ dockerInfo.operatingSystem || '-' }}</b></div>
          <div class="info-row"><span>内核版本</span><b>{{ dockerInfo.kernelVersion || '-' }}</b></div>
          <div class="info-row"><span>存储驱动</span><b>{{ dockerInfo.driver || '-' }}</b></div>
          <div class="info-row"><span>容器总数</span><b>{{ dockerInfo.containers ?? '-' }}</b></div>
          <div class="info-row"><span>镜像总数</span><b>{{ dockerInfo.images ?? '-' }}</b></div>
        </div>

        <div class="info-card">
          <div class="card-title">宿主机信息</div>
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
.dashboard-page { padding: 0; }
.dash-grid {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 16px;
}

/* ========== 三个资源圆圈 ========== */
.rings-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.ring-card {
  background: #fff;
  border-radius: var(--mp-radius-lg);
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.ring-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--mp-text);
}
.ring-body {
  position: relative;
  width: 140px;
  height: 140px;
  margin: 4px auto 8px;
}
.ring-svg {
  width: 140px;
  height: 140px;
  transform: rotate(-90deg);
  overflow: visible;
}
.ring-bg { fill: none; stroke: #f0f2f5; stroke-width: 10; }
.ring-fg {
  fill: none;
  stroke-width: 10;
  stroke-linecap: round;
  transition: stroke-dasharray 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}
.ring-cpu { stroke: #1677ff; }
.ring-mem { stroke: #52c41a; }
.ring-disk { stroke: #faad14; }
.ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.ring-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--mp-text);
  line-height: 1.2;
}
.ring-label { font-size: 12px; color: var(--mp-text-tertiary); }
.ring-legend { margin-top: 4px; }
.legend-row {
  display: flex;
  align-items: center;
  font-size: 13px;
  padding: 2px 0;
  color: var(--mp-text-secondary);
}
.legend-val { margin-left: auto; color: var(--mp-text); font-weight: 500; font-size: 12px; }
.dot {
  width: 8px; height: 8px; border-radius: 50%;
  display: inline-block; margin-right: 8px; flex-shrink: 0;
}
.dot-blue { background: #1677ff; }
.dot-green { background: #52c41a; }
.dot-orange { background: #faad14; }
.dot-gray { background: #d1d5db; }

/* ========== 六卡片两行三列 ========== */
.stat-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-top: 16px;
}
.stat-card {
  background: #fff;
  border-radius: var(--mp-radius-lg);
  padding: 18px 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.stat-label { font-size: 13px; color: var(--mp-text-tertiary); }
.stat-num { font-size: 28px; font-weight: 700; margin: 4px 0 6px; color: var(--mp-text); }
.stat-sub {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--mp-text-tertiary);
}
.stat-sub b { font-weight: 600; color: var(--mp-text-secondary); }
.stat-sub .ok { color: #52c41a; }
.stat-sub .stop { color: #ff4d4f; }
.stat-sub .pause { color: #faad14; }

/* ========== 右侧栏 ========== */
.side-col { display: flex; flex-direction: column; gap: 16px; }
.info-card {
  background: #fff;
  border-radius: var(--mp-radius-lg);
  padding: 18px 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.card-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
  padding-left: 10px;
  border-left: 3px solid #1677ff;
  color: var(--mp-text);
}
.info-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  padding: 5px 0;
  color: var(--mp-text-secondary);
  border-bottom: 1px dashed var(--mp-border-light);
}
.info-row:last-child { border-bottom: none; }
.info-row b { color: var(--mp-text); font-weight: 500; max-width: 180px; text-align: right; word-break: break-all; }
.rank-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  padding: 5px 0;
  border-bottom: 1px dashed var(--mp-border-light);
}
.rank-item:last-child { border-bottom: none; }
.rank-name { color: var(--mp-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 160px; }
.rank-val { color: #1677ff; font-weight: 600; flex-shrink: 0; }
.empty { font-size: 13px; color: var(--mp-text-tertiary); text-align: center; padding: 20px 0; }

@media (max-width: 1100px) {
  .dash-grid { grid-template-columns: 1fr; }
  .rings-row { grid-template-columns: 1fr; }
  .stat-cards { grid-template-columns: repeat(2, 1fr); }
}
</style>
