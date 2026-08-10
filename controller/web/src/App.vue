<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useStatusStore } from './stores/status'

const route = useRoute()
const statusStore = useStatusStore()

onMounted(() => {
  statusStore.refresh()
})
</script>

<template>
  <el-container class="app-root">
    <el-aside width="210px" class="app-aside">
      <div class="logo">sing-box <span class="logo-sub">WebUI</span></div>
      <el-menu :default-active="route.path" router class="app-menu">
        <el-menu-item index="/inbounds">Inbounds</el-menu-item>
        <el-menu-item index="/outbounds">Outbounds</el-menu-item>
        <el-menu-item index="/routes">Routes</el-menu-item>
        <el-menu-item index="/dns">DNS</el-menu-item>
        <el-menu-item index="/diagnostics">诊断</el-menu-item>
        <el-menu-item index="/config">Config</el-menu-item>
        <el-menu-item index="/settings">Settings</el-menu-item>
      </el-menu>
    </el-aside>

    <el-container class="app-body">
      <el-header class="app-header">
        <div class="header-left">
          <span class="dot" />
          <span>sing-box-controller</span>
          <el-divider direction="vertical" />
          <span>Inbounds: {{ statusStore.status?.inbounds ?? '—' }}</span>
          <el-divider direction="vertical" />
          <span>Outbounds: {{ statusStore.status?.outbounds ?? '—' }}</span>
          <el-divider direction="vertical" />
          <span>Rules: {{ statusStore.status?.rules ?? '—' }}</span>
          <el-divider direction="vertical" />
          <span>min_port: {{ statusStore.status?.min_port ?? '—' }}</span>
        </div>
        <div v-if="statusStore.status?.config_path" class="header-right" :title="statusStore.status.config_path">
          主配置: {{ statusStore.status.config_path }}
        </div>
      </el-header>

      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.app-root {
  height: 100vh;
}
.app-aside {
  background: #001529;
  display: flex;
  flex-direction: column;
}
.logo {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  padding: 20px 16px 14px;
  letter-spacing: 1px;
}
.logo-sub {
  color: #1890ff;
}
.app-menu {
  border-right: none;
  background: transparent;
  flex: 1;
}
.app-menu :deep(.el-menu-item) {
  color: #a6b0bf;
}
.app-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.app-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background: #1890ff;
}
.app-body {
  min-width: 0;
}
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #606266;
}
.header-right {
  font-size: 12px;
  color: #909399;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #1890ff;
  display: inline-block;
}
.app-main {
  background: #f5f7fa;
  padding: 16px;
}
</style>
