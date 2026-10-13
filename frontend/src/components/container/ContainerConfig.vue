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
          <el-input v-model="newName" placeholder="容器名称" />
          <el-button type="primary" link @click="handleRename">保存名称</el-button>
        </el-form-item>

        <!-- 镜像 -->
        <el-form-item label="* 镜像">
          <el-input :model-value="config?.image" disabled />
        </el-form-item>

        <!-- 端口 -->
        <el-form-item label="端口">
          <el-table :data="config?.ports || []" size="small" stripe>
            <el-table-column prop="host" label="服务器" />
            <el-table-column prop="container" label="容器" />
            <el-table-column label="协议" width="90">
              <template #default>tcp</template>
            </el-table-column>
          </el-table>
          <div v-if="!config?.ports?.length" class="hint">未映射端口</div>
        </el-form-item>

        <!-- 网络 -->
        <el-form-item label="网络">
          <el-input :model-value="config?.networkMode" disabled />
        </el-form-item>

        <!-- 挂载 -->
        <el-form-item label="挂载">
          <el-table :data="config?.volumes || []" size="small" stripe>
            <el-table-column prop="host" label="本机目录" />
            <el-table-column prop="container" label="容器目录" />
            <el-table-column prop="mode" label="权限" width="90" />
          </el-table>
          <div v-if="!config?.volumes?.length" class="hint">无挂载</div>
        </el-form-item>

        <!-- Command / Entrypoint -->
        <el-form-item label="Command">
          <el-input :model-value="config?.cmd?.join(' ') || '-'" type="textarea" :rows="2" disabled />
        </el-form-item>
        <el-form-item label="Entrypoint">
          <el-input :model-value="config?.entrypoint?.join(' ') || '-'" disabled />
        </el-form-item>

        <!-- 重启规则 -->
        <el-form-item label="重启规则">
          <el-radio-group v-model="resourceForm.restart">
            <el-radio value="no">不重启</el-radio>
            <el-radio value="always">一直重启</el-radio>
            <el-radio value="on-failure">失败后重启</el-radio>
            <el-radio value="unless-stopped">未手动停止则重启</el-radio>
          </el-radio-group>
          <el-button type="primary" link @click="updateRestart">保存重启规则</el-button>
        </el-form-item>

        <!-- CPU 限制 -->
        <el-form-item label="CPU 限制（核，0 为不限制）">
          <el-input-number v-model="resourceForm.cpus" :min="0" :step="0.1" :precision="1" controls-position="right" />
          <el-button type="primary" link @click="updateCpus">保存</el-button>
        </el-form-item>

        <!-- 内存限制 -->
        <el-form-item label="内存限制（MB，0 为不限制）">
          <el-input-number v-model="resourceForm.memory" :min="0" :step="64" controls-position="right" />
          <el-button type="primary" link @click="updateMemory">保存</el-button>
        </el-form-item>

        <!-- 特权 -->
        <el-form-item label="特权模式">
          <el-tag :type="config?.privileged ? 'danger' : 'success'">
            {{ config?.privileged ? '已开启' : '未开启' }}
          </el-tag>
        </el-form-item>

        <!-- 标签 -->
        <el-form-item label="标签">
          <el-input
            :model-value="Object.entries(config?.labels || {}).map(([k, v]) => `${k}=${v}`).join('\n')"
            type="textarea"
            :rows="4"
            disabled
          />
        </el-form-item>

        <!-- 环境变量 -->
        <el-form-item label="环境变量">
          <el-input
            :model-value="(config?.env || []).map(e => `${e.key}=${e.value}`).join('\n')"
            type="textarea"
            :rows="4"
            disabled
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button type="warning" @click="showRecreateDialog = true">
        重建容器（修改端口/挂载/镜像等）
      </el-button>
      <el-button @click="dialogVisible = false">{{ t('common.close') }}</el-button>
    </template>

    <!-- Recreate Container Dialog -->
    <RecreateContainerDialog
      v-model:visible="showRecreateDialog"
      :container-id="containerId"
      :container-config="config"
      @recreated="handleRecreated"
    />
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from '@/composables/useI18n'
import { containerApi } from '@/api/containers'
import type { ContainerConfig } from '@/types'
import RecreateContainerDialog from './RecreateContainerDialog.vue'

const props = defineProps<{
  visible: boolean
  containerId: string
  containerName: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  updated: []
}>()

const { t } = useI18n()

const drawerSize = computed(() => (window.innerWidth < 768 ? '100%' : '640px'))

// Dialog visibility
const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

// State
const loading = ref(false)
const config = ref<ContainerConfig | null>(null)
const newName = ref('')
const showRecreateDialog = ref(false)

// Resource form
const resourceForm = reactive({
  memory: 0,
  cpus: 0,
  restart: 'no',
})

// Load container config
async function loadConfig() {
  if (!props.containerId) return

  loading.value = true
  try {
    config.value = await containerApi.inspect(props.containerId)
    newName.value = config.value?.name || ''
    resourceForm.memory = config.value.memory || 0
    resourceForm.cpus = config.value.cpus || 0
    resourceForm.restart = config.value.restart || 'no'
  } catch {
    ElMessage.error(t('config.loadError'))
    dialogVisible.value = false
  } finally {
    loading.value = false
  }
}

async function handleRename() {
  if (!newName.value.trim()) {
    ElMessage.warning(t('config.nameRequired'))
    return
  }

  try {
    await containerApi.rename(props.containerId, newName.value)
    ElMessage.success(t('config.renameSuccess'))
    if (config.value) {
      config.value.name = newName.value
    }
    emit('updated')
  } catch {
    // Error handled by interceptor
  }
}

// Resource updates
async function updateMemory() {
  try {
    await containerApi.update(props.containerId, { memory: resourceForm.memory })
    ElMessage.success(t('config.updateSuccess'))
    emit('updated')
  } catch {
    // Error handled by interceptor
  }
}

async function updateCpus() {
  try {
    await containerApi.update(props.containerId, { cpus: resourceForm.cpus })
    ElMessage.success(t('config.updateSuccess'))
    emit('updated')
  } catch {
    // Error handled by interceptor
  }
}

async function updateRestart() {
  try {
    await containerApi.update(props.containerId, { restart: resourceForm.restart })
    ElMessage.success(t('config.updateSuccess'))
    emit('updated')
  } catch {
    // Error handled by interceptor
  }
}

// Handle recreated
function handleRecreated() {
  emit('updated')
  dialogVisible.value = false
}

// Handle dialog open
function handleOpen() {
  loadConfig()
}
</script>

<style scoped>
.container-config {
  min-height: 400px;
}

.editable-field {
  display: flex;
  align-items: center;
  gap: 8px;
}

.truncate {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
}

.resource-field {
  display: flex;
  align-items: center;
  gap: 10px;
}

.unit {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 15px;
}

h4 {
  margin: 20px 0 10px;
  font-size: 14px;
  font-weight: 600;
}

code {
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}
</style>
