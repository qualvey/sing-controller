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
        <el-menu-item index="/rule-sets">规则集</el-menu-item>
        <el-menu-item index="/dns">DNS</el-menu-item>
        <el-menu-item index="/certificate">证书</el-menu-item>
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

/* 左下角悬浮重载按钮 */
.reload-fab {
  position: fixed;
  left: 18px;
  bottom: 18px;
  z-index: 2000;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: #409eff;
  color: #fff;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.4);
  transition: background 0.2s, transform 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.reload-fab:hover {
  background: #66b1ff;
}
.reload-fab:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
.reload-fab.spinning {
  animation: fab-spin 1s linear infinite;
}
@keyframes fab-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
