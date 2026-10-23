<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  Bell, Box, Clock, Connection, Cpu, Files, FolderOpened, Menu, Monitor,
  Moon, Picture, Setting, Sunny, SwitchButton, User, OfficeBuilding
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useNodesStore } from '@/stores/nodes'
import { useContainerStore } from '@/stores/containers'
import { useNetworkStore } from '@/stores/networks'
import { useVolumeStore } from '@/stores/volumes'
import { useComposeStore } from '@/stores/compose'
import { useRegistryStore } from '@/stores/registry'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from '@/composables/useI18n'
import SystemMonitor from '@/components/common/SystemMonitor.vue'
import ChangePasswordDialog from '@/components/common/ChangePasswordDialog.vue'
import SettingsDialog from '@/components/common/SettingsDialog.vue'
import UpdateDialog from '@/components/common/UpdateDialog.vue'

// ========== 版本号（每次递增） ==========
const APP_VERSION = 'v1.0.0'

// ========== 路由 & Store ==========
const router = useRouter()
const authStore = useAuthStore()
const nodesStore = useNodesStore()
const containerStore = useContainerStore()
const networkStore = useNetworkStore()
const volumeStore = useVolumeStore()
const composeStore = useComposeStore()
const registryStore = useRegistryStore()
const { theme, toggleTheme } = useTheme()
const { t } = useI18n()

// ========== 响应式 & 对话框 ==========
const windowWidth = ref(window.innerWidth)
const isMobile = computed(() => windowWidth.value < 768)
const drawerVisible = ref(false)
const showPasswordDialog = ref(false)
const showSettingsDialog = ref(false)
const showUpdateDialog = ref(false)

const themeIcon = computed(() => theme.value === 'dark' ? Sunny : Moon)
const themeTooltip = computed(() => theme.value === 'dark' ? t('header.lightMode') : t('header.darkMode'))

// ========== 菜单（8 项，路由 path 保持不变） ==========
const MENU_GROUPS = [
  {
    title: '概览',
    items: [
      { index: '/', icon: Monitor, title: '概览' },
    ],
  },
  {
    title: '容器',
    items: [
      { index: '/containers', icon: Box, title: '容器管理' },
      { index: '/compose', icon: Files, title: '任务管理' },
      { index: '/images', icon: Picture, title: '镜像管理' },
    ],
  },
  {
    title: '管理',
    items: [
      { index: '/files', icon: FolderOpened, title: '文件管理' },
      { index: '/cron', icon: Clock, title: '计划任务' },
      { index: '/terminal', icon: Cpu, title: '终端' },
    ],
  },
  {
    title: '其它',
    items: [
      { index: '/networks', icon: Connection, title: '网络管理' },
      { index: '/volumes', icon: FolderOpened, title: '存储卷管理' },
      { index: '/registry', icon: OfficeBuilding, title: '仓库管理' },
      { index: '/docker-config', icon: Setting, title: 'Docker 配置' },
    ],
  },
]

// ========== 事件 ==========
function handleResize() {
  windowWidth.value = window.innerWidth
  if (!isMobile.value) drawerVisible.value = false
}

function handleMenuSelect(index: string) {
  router.push(index)
  if (isMobile.value) drawerVisible.value = false
}

async function handleLogout() {
  try {
    await ElMessageBox.confirm(`${t('auth.logout')}?`, t('common.confirm'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning',
    })
    await authStore.logout()
    ElMessage.success(t('auth.logoutSuccess'))
    router.push('/login')
  } catch { /* ignore */ }
}

function handleChangePassword() {
  showPasswordDialog.value = true
  if (isMobile.value) drawerVisible.value = false
}

function handleOpenSettings() {
  showSettingsDialog.value = true
  if (isMobile.value) drawerVisible.value = false
}

function toggleDrawer() { drawerVisible.value = !drawerVisible.value }

// ========== Store 联动 ==========
watch(() => nodesStore.currentNodeId, (id, prev) => {
  if (id === prev) return
  containerStore.prepareNodeSwitch()
  networkStore.prepareNodeSwitch()
  volumeStore.prepareNodeSwitch()
  composeStore.prepareNodeSwitch()
  registryStore.prepareNodeSwitch()
  containerStore.fetchContainers()
  networkStore.fetchNetworks()
  volumeStore.fetchVolumes()
  composeStore.fetchProjects()
  registryStore.fetchRegistries()
})

watch(() => nodesStore.fellBackToMaster, (fellBack) => {
  if (!fellBack) return
  ElMessage.warning(t('node.nodeOfflineSwitched'))
  nodesStore.acknowledgeFallback()
})

onMounted(() => {
  window.addEventListener('resize', handleResize)
  nodesStore.startPolling()
})
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  nodesStore.stopPolling()
})
</script>

<template>
  <el-container class="mp-layout">
    <!-- ========== 侧边栏（桌面） ========== -->
    <el-aside v-if="!isMobile" width="220px" class="mp-sidebar">
      <!-- Logo + 版本号 -->
      <div class="mp-sidebar-logo">
        <span class="mp-logo-title">My Panel</span>
        <span class="mp-logo-version">{{ APP_VERSION }}</span>
      </div>

      <!-- 菜单（按组分隔） -->
      <el-menu
        :default-active="$route.path"
        class="mp-sidebar-menu"
        @select="handleMenuSelect"
      >
        <template v-for="group in MENU_GROUPS" :key="group.title">
          <div class="mp-menu-group-title">{{ group.title }}</div>
          <el-menu-item
            v-for="item in group.items"
            :key="item.index"
            :index="item.index"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <!-- ========== 侧边栏（移动端 Drawer） ========== -->
    <el-drawer v-model="drawerVisible" direction="ltr" :size="280" :with-header="false" class="mp-mobile-drawer">
      <div class="mp-drawer-logo">
        <span class="mp-logo-title">My Panel</span>
        <span class="mp-logo-version">{{ APP_VERSION }}</span>
      </div>
      <el-menu :default-active="$route.path" class="mp-drawer-menu" @select="handleMenuSelect">
        <template v-for="group in MENU_GROUPS" :key="group.title">
          <div class="mp-menu-group-title">{{ group.title }}</div>
          <el-menu-item v-for="item in group.items" :key="item.index" :index="item.index">
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.title }}</span>
          </el-menu-item>
        </template>
      </el-menu>
      <div class="mp-drawer-footer">
        <div class="mp-drawer-user">
          <el-avatar :size="40" :icon="User" />
          <span>{{ authStore.username }}</span>
        </div>
        <div class="mp-drawer-actions">
          <el-button text @click="showUpdateDialog = true">
            <el-icon><Bell /></el-icon>{{ t('update.title') }}
          </el-button>
          <el-button text @click="handleChangePassword">
            <el-icon><SwitchButton /></el-icon>{{ t('header.changePassword') }}
          </el-button>
          <el-button text type="danger" @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>{{ t('nav.logout') }}
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- ========== 右侧区域 ========== -->
    <el-container>
      <!-- 顶部栏：资源监控 + 用户菜单 -->
      <el-header class="mp-header">
        <div class="mp-header-left">
          <el-button v-if="isMobile" :icon="Menu" circle @click="toggleDrawer" />
          <SystemMonitor />
        </div>
        <div class="mp-header-right">
          <el-tooltip :content="themeTooltip" placement="bottom">
            <el-button :icon="themeIcon" circle @click="toggleTheme" />
          </el-tooltip>
          <el-dropdown v-if="!isMobile">
            <div class="mp-user-chip">
              <el-avatar :size="32" :icon="User" />
              <span>{{ authStore.username }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="showUpdateDialog = true">
                  <el-icon><Bell /></el-icon>{{ t('update.title') }}
                </el-dropdown-item>
                <el-dropdown-item @click="handleOpenSettings">
                  <el-icon><Setting /></el-icon>{{ t('settings.pageSettings') }}
                </el-dropdown-item>
                <el-dropdown-item @click="handleChangePassword">
                  <el-icon><SwitchButton /></el-icon>{{ t('header.changePassword') }}
                </el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>{{ t('nav.logout') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="mp-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>

  <ChangePasswordDialog v-model="showPasswordDialog" @success="() => { showPasswordDialog = false; ElMessage.success(t('auth.passwordChanged')) }" />
  <SettingsDialog v-model="showSettingsDialog" />
  <UpdateDialog v-model="showUpdateDialog" />
</template>

<style scoped>
/* ========== 整体 ========== */
.mp-layout { height: 100vh; overflow: hidden; }

/* ========== 侧边栏（深色 dpanel 风） ========== */
.mp-sidebar {
  background: #001529;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: none;
}

/* Logo + 版本号 */
.mp-sidebar-logo {
  padding: 18px 20px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  text-align: center;
}
.mp-logo-title {
  display: block;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}
.mp-logo-version {
  display: inline-block;
  margin-top: 4px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.06);
  padding: 1px 8px;
  border-radius: 10px;
  letter-spacing: 0.5px;
}

/* 菜单分组标题 */
.mp-menu-group-title {
  padding: 16px 20px 6px;
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.35);
  text-transform: uppercase;
  letter-spacing: 1px;
}

/* 菜单主体 */
.mp-sidebar-menu {
  flex: 1;
  border-right: none;
  overflow-y: auto;
  background: #001529 !important;
  --el-menu-bg-color: #001529;
  --el-menu-text-color: rgba(255, 255, 255, 0.75);
  --el-menu-hover-bg-color: rgba(255, 255, 255, 0.08);
  --el-menu-hover-text-color: #fff;
  --el-menu-active-color: #fff;
  --el-menu-item-height: 42px;
  --el-menu-sub-item-height: 42px;
}
.mp-sidebar-menu :deep(.el-menu-item) {
  margin: 2px 10px;
  border-radius: 6px;
  padding-left: 14px !important;
}
.mp-sidebar-menu :deep(.el-menu-item.is-active) {
  background: #1677ff !important;
  color: #fff !important;
}
.mp-sidebar-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08) !important;
}
.mp-sidebar-menu :deep(.el-icon) { font-size: 17px; }

/* ========== 顶部栏 ========== */
.mp-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: var(--mp-bg-card);
  border-bottom: 1px solid var(--mp-border-light);
}
.mp-header-left { display: flex; align-items: center; gap: 12px; }
.mp-header-right { display: flex; align-items: center; gap: 8px; }

/* 用户 chip */
.mp-user-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 6px;
  transition: background 0.2s;
}
.mp-user-chip:hover { background: var(--mp-bg-hover); }

/* ========== 内容区 ========== */
.mp-content {
  background: var(--mp-bg-page);
  overflow-y: auto;
  padding: 0;
}

/* ========== 移动端 Drawer ========== */
.mp-mobile-drawer :deep(.el-drawer__body) { padding: 0; display: flex; flex-direction: column; }
.mp-drawer-logo {
  padding: 20px;
  border-bottom: 1px solid rgba(0,0,0,0.06);
  text-align: center;
  background: #001529;
}
.mp-drawer-menu { flex: 1; border-right: none; overflow-y: auto; }
.mp-drawer-menu :deep(.el-menu-item.is-active) { background: #1677ff !important; color: #fff !important; }
.mp-drawer-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--mp-border-light);
}
.mp-drawer-user { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; font-weight: 500; }
.mp-drawer-actions { display: flex; flex-direction: column; gap: 4px; }

@media (max-width: 768px) {
  .mp-header { padding: 0 12px; }
}
</style>
