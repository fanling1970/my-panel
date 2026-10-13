<template>
  <el-drawer
    v-model="dialogVisible"
    :title="'编辑 - ' + containerName"
    direction="rtl"
    :size="drawerSize"
    :close-on-click-modal="false"
    destroy-on-close
    class="container-config-dialog"
    @open="handleOpen"
  >
    <div v-loading="loading" class="container-config">
      <el-form label-position="top" class="config-form">
        <!-- 名称 -->
        <el-form-item label="* 名称">
          <el-input v-model="form.name" placeholder="容器名称" />
        </el-form-item>

        <!-- 镜像 -->
        <el-form-item label="* 镜像">
          <el-checkbox v-model="manualImage">手动输入镜像</el-checkbox>
          <el-checkbox v-model="form.pullLatest" style="margin-left: 16px">
            强制拉取镜像（忽略本地镜像，重新拉取）
          </el-checkbox>
          <el-input v-if="manualImage" v-model="form.image" placeholder="例如 nginx:latest" style="margin-top: 10px" />
          <el-select v-else v-model="form.image" placeholder="选择镜像" style="width: 100%; margin-top: 10px">
            <el-option
              v-for="img in imageList"
              :key="img.id"
              :value="img.name"
              :label="img.name"
            >
              <span :style="{ fontWeight: img.name === config?.image ? 'bold' : 'normal' }">{{ img.name }}</span>
            </el-option>
          </el-select>
        </el-form-item>

        <!-- 端口 -->
        <el-form-item label="端口">
          <el-table :data="form.ports" size="small" border>
            <el-table-column label="服务器端口" min-width="120">
              <template #default="{ row }">
                <el-input v-model="row.host" placeholder="如 8080" />
              </template>
            </el-table-column>
            <el-table-column label="容器端口" min-width="120">
              <template #default="{ row }">
                <el-input v-model="row.container" placeholder="如 80" />
              </template>
            </el-table-column>
            <el-table-column label="协议" width="80">
              <template #default>tcp</template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="form.ports.splice($index, 1)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 8px; width: 100%">
            <el-button size="small" @click="form.ports.push({ host: '', container: '' })">添加</el-button>
          </div>
        </el-form-item>

        <!-- 网络 -->
        <el-form-item label="网络">
          <el-select v-model="form.network" placeholder="选择网络" style="width: 100%">
            <el-option label="bridge" value="bridge" />
            <el-option label="host" value="host" />
            <el-option label="none" value="none" />
            <el-option v-for="n in networkList" :key="n.name" :value="n.name" :label="n.name" />
          </el-select>
        </el-form-item>

        <!-- 挂载 -->
        <el-form-item label="挂载">
          <el-table :data="form.volumes" size="small" border>
            <el-table-column label="本机目录" min-width="160">
              <template #default="{ row }">
                <el-input v-model="row.host" placeholder="/host/path" />
              </template>
            </el-table-column>
            <el-table-column label="容器目录" min-width="160">
              <template #default="{ row }">
                <el-input v-model="row.container" placeholder="/container/path" />
              </template>
            </el-table-column>
            <el-table-column label="权限" width="80">
              <template #default="{ row }">
                <el-select v-model="row.mode">
                  <el-option label="rw" value="rw" />
                  <el-option label="ro" value="ro" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="form.volumes.splice($index, 1)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 8px; width: 100%">
            <el-button size="small" @click="form.volumes.push({ host: '', container: '', mode: 'rw' })">添加</el-button>
          </div>
        </el-form-item>

        <!-- Command / Entrypoint -->
        <el-form-item label="Command">
          <el-input v-model="form.cmd" type="textarea" :rows="2" placeholder="-" />
        </el-form-item>
        <el-form-item label="Entrypoint">
          <el-input v-model="form.entrypoint" placeholder="-" />
        </el-form-item>

        <!-- 重启规则 -->
        <el-form-item label="重启规则">
          <el-radio-group v-model="form.restart">
            <el-radio value="no">不重启</el-radio>
            <el-radio value="always">一直重启</el-radio>
            <el-radio value="on-failure">失败后重启</el-radio>
            <el-radio value="unless-stopped">未手动停止则重启</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- CPU / 内存 -->
        <el-form-item label="CPU 限制（核，0 为不限制）">
          <el-input-number v-model="form.cpus" :min="0" :step="0.1" :precision="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="内存限制（MB，0 为不限制）">
          <el-input-number v-model="form.memory" :min="0" :step="64" controls-position="right" />
        </el-form-item>

        <!-- 特权 -->
        <el-form-item label="特权模式">
          <el-tag :type="config?.privileged ? 'danger' : 'success'">
            {{ config?.privileged ? '已开启（重建后保留）' : '未开启' }}
          </el-tag>
        </el-form-item>

        <!-- 环境变量 -->
        <el-form-item label="环境变量">
          <el-table :data="form.env" size="small" border>
            <el-table-column label="键" min-width="140">
              <template #default="{ row }">
                <el-input v-model="row.key" placeholder="KEY" />
              </template>
            </el-table-column>
            <el-table-column label="值" min-width="160">
              <template #default="{ row }">
                <el-input v-model="row.value" placeholder="value" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="form.env.splice($index, 1)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 8px; width: 100%">
            <el-button size="small" @click="form.env.push({ key: '', value: '' })">添加</el-button>
          </div>
        </el-form-item>

        <!-- 标签（只读展示） -->
        <el-form-item label="标签">
          <el-input
            :model-value="Object.entries(config?.labels || {}).map(([k, v]) => `${k}=${v}`).join('\n')"
            type="textarea"
            :rows="3"
            disabled
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleConfirm">确认</el-button>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { containerApi } from '@/api/containers'
import { imageApi } from '@/api/images'
import { networkApi } from '@/api/networks'
import type { ContainerConfig, ImageInfo, NetworkInfo } from '@/types'

const props = defineProps<{
  visible: boolean
  containerId: string
  containerName: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  updated: []
}>()

const drawerSize = computed(() => (window.innerWidth < 768 ? '100%' : '50%'))

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

const loading = ref(false)
const saving = ref(false)
const config = ref<ContainerConfig | null>(null)
const imageList = ref<ImageInfo[]>([])
const networkList = ref<NetworkInfo[]>([])
const manualImage = ref(false)

const form = reactive({
  name: '',
  image: '',
  pullLatest: false,
  network: 'bridge',
  restart: 'no',
  cpus: 0,
  memory: 0,
  cmd: '',
  entrypoint: '',
  ports: [] as Array<{ host: string; container: string }>,
  volumes: [] as Array<{ host: string; container: string; mode: string }>,
  env: [] as Array<{ key: string; value: string }>,
})

async function loadConfig() {
  if (!props.containerId) return
  loading.value = true
  try {
    const [cfg, imgs, nets] = await Promise.all([
      containerApi.inspect(props.containerId),
      imageApi.list(),
      networkApi.list(),
    ])
    config.value = cfg
    imageList.value = imgs
    networkList.value = nets || []

    form.name = cfg.name
    form.image = cfg.image
    form.network = cfg.networkMode || 'bridge'
    form.restart = cfg.restart || 'no'
    form.cpus = cfg.cpus || 0
    form.memory = cfg.memory || 0
    form.cmd = (cfg.cmd || []).join(' ')
    form.entrypoint = (cfg.entrypoint || []).join(' ')
    form.ports = (cfg.ports || []).map((p) => {
      // container 形如 "5244/tcp"，只留数字
      const portNum = String(p.container || '').split('/')[0] || ''
      return { host: p.host || '', container: portNum }
    })
    form.volumes = (cfg.volumes || []).map((v) => ({
      host: v.host || '',
      container: v.container || '',
      mode: v.mode || 'rw',
    }))
    form.env = (cfg.env || []).map((e) => ({ key: e.key, value: e.value }))
  } catch {
    ElMessage.error('加载容器配置失败')
    dialogVisible.value = false
  } finally {
    loading.value = false
  }
}

async function handleConfirm() {
  if (!form.name.trim() || !form.image.trim()) {
    ElMessage.warning('名称和镜像不能为空')
    return
  }
  try {
    await ElMessageBox.confirm(
      '编辑容器需要重建，任何未持久化的数据将丢失，是否继续？',
      '编辑',
      { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }

  saving.value = true
  try {
    await containerApi.recreate({
      container_id: props.containerId,
      name: form.name.trim(),
      image: form.image.trim(),
      restart: form.restart,
      network: form.network,
      cpus: form.cpus,
      memory: form.memory,
      pull_latest: form.pullLatest,
      ports: form.ports.filter((p) => p.host && p.container),
      volumes: form.volumes.filter((v) => v.host && v.container),
      env: form.env.filter((e) => e.key),
    })
    ElMessage.success('容器已重建')
    emit('updated')
    dialogVisible.value = false
  } catch {
    // 拦截器已提示
  } finally {
    saving.value = false
  }
}

function handleOpen() {
  loadConfig()
}
</script>

<style scoped>
.config-form :deep(.el-form-item) {
  margin-bottom: 18px;
}
</style>
