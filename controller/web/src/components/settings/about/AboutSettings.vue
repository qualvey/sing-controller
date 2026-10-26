<template>
  <div class="text-sm">
    <div class="settings-section-label">{{ $t('aboutVersions') }}</div>
    <div class="settings-grid">
      <div class="setting-item">
        <div class="setting-item-label">{{ $t('aboutFrontendVersion') }}</div>
        <code class="text-xs break-all">{{ frontendVersion }}</code>
      </div>
      <div class="setting-item">
        <div class="setting-item-label">{{ $t('aboutControllerVersion') }}</div>
        <code class="text-xs break-all">{{ controllerVersion }}</code>
      </div>
      <div class="setting-item">
        <div class="setting-item-label">{{ $t('aboutCoreVersion') }}</div>
        <code class="text-xs break-all">{{ coreVersion || '—' }}</code>
      </div>
      <div class="setting-item">
        <div class="setting-item-label">{{ $t('aboutBuildTime') }}</div>
        <span class="text-xs break-all">{{ buildInfo.buildTime || '—' }}</span>
      </div>
    </div>

    <div class="settings-section-label">{{ $t('aboutProject') }}</div>
    <div class="settings-grid">
      <div class="setting-item">
        <div class="setting-item-label">{{ $t('aboutRepository') }}</div>
        <a
          class="link link-primary text-xs break-all"
          href="https://github.com/qualvey/sing-controller"
          target="_blank"
          rel="noopener noreferrer"
        >
          github.com/qualvey/sing-controller
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { version as coreVersion } from '@/assembly/version'
import { useStatusStore } from '@/store/status'
import { computed, onMounted } from 'vue'
import { buildInfo } from 'virtual:recent-changes'

const statusStore = useStatusStore()
const commitId = __COMMIT_ID__ || buildInfo.commit || 'dev'
const frontendVersion = computed(() => `${__APP_VERSION__ || 'dev'}-${commitId}`)
const controllerVersion = computed(() => statusStore.status?.controller_version || '—')

onMounted(() => {
  void statusStore.refresh()
})
</script>
