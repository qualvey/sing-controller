<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { ControllerSettings } from '../api'

const form = ref<ControllerSettings>({
  config: './sing-box-config.json',
  listen: '127.0.0.1:8080',
  log: { level: 'info' },
  min_port: 8000,
  defaults: {
    inbound_type: 'mixed',
    outbound_type: 'vless',
    listen: '127.0.0.1',
    listen_port: 2080,
    attach_to_selector: true,
    proxy_selector: 'Proxy'
  }
})
const loading = ref(false)
const saving = ref(false)

const load = async () => {
  loading.value = true
  try {
    form.value = await api.settings()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载设置失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    const res = (await api.updateSettings(form.value)) as { load_error?: string; message?: string; warning?: string }
    if (res.load_error) {
      ElMessage.warning(res.message || `设置已保存，但新主配置路径加载失败：${res.load_error}`)
    } else if (res.warning) {
      ElMessage.warning(res.warning)
    } else {
      ElMessage.success('设置已保存')
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>sing-box-controller 设置</span>
          <div>
            <el-button :loading="loading" @click="load">刷新</el-button>
            <el-button type="primary" :loading="saving" @click="save">保存</el-button>
          </div>
        </div>
      </template>

      <el-form :model="form" label-width="180px" class="settings-form">
        <el-divider content-position="left">主配置</el-divider>
        <el-form-item label="sing-box 主配置路径">
          <el-input v-model="form.config" placeholder="./sing-box-config.json" />
          <div class="field-hint">controller 只读/写这个文件。路径变更后立即生效，新路径不存在则自动生成骨架。</div>
        </el-form-item>
        <el-form-item label="HTTP 监听地址">
          <el-input v-model="form.listen" placeholder="127.0.0.1:8080" />
          <div class="field-hint">webui 访问地址，格式 host:port。修改后需重启 sing-controller 服务才生效（deb 部署：systemctl restart sing-controller）。</div>
        </el-form-item>
        <el-form-item label="日志级别">
          <el-select v-model="form.log!.level" style="width: 200px">
            <el-option v-for="l in ['trace', 'debug', 'info', 'warn', 'error', 'fatal']" :key="l" :label="l" :value="l" />
          </el-select>
          <div class="field-hint">级别枚举与 sing-box 一致（journald 查看：journalctl -u sing-controller）。</div>
        </el-form-item>
        <el-form-item label="端口分配起点 (min_port)">
          <el-input-number v-model="form.min_port" :min="1024" :max="65535" />
          <div class="field-hint">自动分配端口时从该值开始向上查找第一个空闲端口，默认 8000。</div>
        </el-form-item>

        <el-divider content-position="left">新建默认值</el-divider>
        <el-form-item label="默认 inbound type">
          <el-input v-model="form.defaults.inbound_type" placeholder="mixed" />
        </el-form-item>
        <el-form-item label="默认 outbound type">
          <el-input v-model="form.defaults.outbound_type" placeholder="vless" />
        </el-form-item>
        <el-form-item label="默认 listen">
          <el-input v-model="form.defaults.listen" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="默认 listen_port">
          <el-input-number v-model="form.defaults.listen_port" :min="1" :max="65535" />
          <div class="field-hint">新建 inbound 时预填；若该端口被占用可在表单里点"自动分配端口"。</div>
        </el-form-item>

        <el-divider content-position="left">新建 outbound 自动并入组</el-divider>
        <el-form-item label="并入 Proxy(selector)">
          <el-switch v-model="form.defaults.attach_to_selector" />
          <div class="field-hint">默认开启：新建 outbound 时自动追加到指定 selector/urltest 的成员列表。</div>
        </el-form-item>
        <el-form-item label="目标组 tag">
          <el-input v-model="form.defaults.proxy_selector" placeholder="Proxy" :disabled="!form.defaults.attach_to_selector" />
          <div class="field-hint">须先存在同名 selector（或 urltest）outbound，否则不生效。</div>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.settings-form {
  max-width: 720px;
}
.field-hint {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 2px;
}
</style>
