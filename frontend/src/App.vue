<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUpdateStore } from '@/stores/update'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from '@/composables/useI18n'

const route = useRoute()
const authStore = useAuthStore()
const updateStore = useUpdateStore()
const { initTheme } = useTheme()
const { initLanguage } = useI18n()

watch(
  () => route.name,
  (name) => {
    if (name !== 'login' && authStore.isAuthenticated && !updateStore.info && !updateStore.loading) {
      updateStore.fetchUpdateInfo()
    }
  }
)

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, wasAuthenticated) => {
    if (isAuthenticated && !wasAuthenticated) {
      updateStore.fetchUpdateInfo()
    }
    if (!isAuthenticated && wasAuthenticated) {
      updateStore.resetAllState()
    }
  }
)

onMounted(() => {
  authStore.initialize()
  initTheme()
  initLanguage()
  if (authStore.isAuthenticated) {
    updateStore.fetchUpdateInfo()
  }
})
</script>

<template>
  <router-view />
</template>

<style>
/* Global app styles handled in styles/index.css */
</style>
