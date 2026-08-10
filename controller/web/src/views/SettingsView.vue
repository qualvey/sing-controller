<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { ControllerSettings, Outbound } from '../api'

const selectorTags = ref<string[]>([])

const form = ref<ControllerSettings>({
  config: './sing-box-config.json',
  listen: '127.0.0.1:8080',
  log: { level: 'info' },
  min_port: 8000,
  reload: { mode: 'none', after_save: false },
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
    const s = await api.settings()
    // 兼容旧配置：attach_to_selector 缺失时按默认开启
    if (s.defaults.attach_to_selector === undefined) {
      s.defaults.attach_to_selector = true
    }
    if (!s.reload) {
      s.reload = { mode: 'none', after_save: false }
    }
    form.value = s
  } catch (e) {
    ElMessage.error((e as Error).message || '加载设置失败')
  } finally {
    loading.value = false
  }
}

// 目标组 tag 候选：现有 type=selector 的 outbound
const loadSelectors = async () => {
  try {
    const obs: Outbound[] = await api.outbounds()
    selectorTags.value = obs.filter((o) => o.type === 'selector').map((o) => String(o.tag)).filter(Boolean)
  } catch {
    // 忽略：候选为空时仍可手动输入
  }
}

onMounted(() => {
  load()
  loadSelectors()
})

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
          <div class="field-hint">默认开启：新建 outbound 时自动追加到指定 selector 的成员列表。</div>
        </el-form-item>
        <el-form-item label="目标组 tag">
          <el-select
            v-model="form.defaults.proxy_selector"
            filterable
            allow-create
            :disabled="!form.defaults.attach_to_selector"
            placeholder="选择或输入 selector tag"
            style="width: 100%"
          >
            <el-option v-for="t in selectorTags" :key="t" :label="t" :value="t" />
          </el-select>
          <div class="field-hint">下拉自动列出现有 type=selector 的 outbound；须先存在同名 selector，否则不生效。</div>
        </el-form-item>

        <el-divider content-position="left">sing-box 重载</el-divider>
        <el-form-item label="重载方式 (mode)">
          <el-select v-model="form.reload!.mode" style="width: 100%">
            <el-option label="不启用 (none)" value="none" />
            <el-option label="systemd（systemctl reload，推荐）" value="systemd" />
            <el-option label="pidfile（读取 PID 后 kill -HUP）" value="pidfile" />
            <el-option label="hook（自定义命令）" value="hook" />
          </el-select>
          <div class="field-hint">sing-box 官方重载机制只有 SIGHUP（收到后重载配置）。保存/手动触发时按此方式执行。</div>
        </el-form-item>
        <el-form-item v-if="form.reload!.mode === 'systemd'" label="systemd 服务名">
          <el-input v-model="form.reload!.service" placeholder="sing-box" />
          <div class="field-hint">执行 systemctl reload &lt;服务名&gt;；需 sing-controller 用户有对应权限（通常需 root 或 polkit 授权）。</div>
        </el-form-item>
        <el-form-item v-if="form.reload!.mode === 'pidfile'" label="PID 文件路径">
          <el-input v-model="form.reload!.pid_file" placeholder="/run/sing-box.pid" />
        </el-form-item>
        <el-form-item v-if="form.reload!.mode === 'hook'" label="hook 命令">
          <el-input v-model="form.reload!.hook" placeholder="sudo systemctl reload sing-box" />
          <div class="field-hint">以 sing-controller 用户执行（sh -c）；如需 sudo 请配置 NOPASSWD。</div>
        </el-form-item>
        <el-form-item label="保存后自动重载">
          <el-switch v-model="form.reload!.after_save" :disabled="form.reload!.mode === 'none'" />
          <div class="field-hint">开启后所有配置写操作保存成功即自动触发重载；失败不影响保存，仅提示。</div>
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
