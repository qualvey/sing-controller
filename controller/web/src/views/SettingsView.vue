<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { api } from '../api'
import type { ControllerSettings, Outbound } from '../api'

const selectorTags = ref<string[]>([])

const form = ref<ControllerSettings>({
  config: './sing-box-config.json',
  listen: '127.0.0.1:8080',
  log: { level: 'info' },
  min_port: 8000,
  reload: { mode: 'auto', after_save: false },
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
      s.reload = { mode: 'auto', after_save: false }
    }
    form.value = s
  } catch (e) {
    showToast((e as Error).message || '加载设置失败', 'error')
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
      showToast(res.message || `设置已保存，但新主配置路径加载失败：${res.load_error}`, 'warning')
    } else if (res.warning) {
      showToast(res.warning, 'warning')
    } else {
      showToast('设置已保存', 'success')
    }
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
      <!-- 卡片头 -->
      <div class="flex items-center justify-between border-b border-[#e4e7ed] px-5 py-3.5 dark:border-[#303030]">
        <span class="text-[15px] font-semibold text-[#303133] dark:text-[#e5eaf3]">sing-box-controller 设置</span>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="load">刷新</button>
          <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
        </div>
      </div>

      <!-- 表单：label 左对齐固定 180px（el-form label-width 同款布局） -->
      <div class="max-w-[720px] p-5">
        <div class="mb-3 flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]">
          <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />主配置<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
        </div>
        <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 180px minmax(0, 1fr)">
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">sing-box 主配置路径</label>
          <div>
            <input v-model="form.config" type="text" class="input input-bordered input-sm w-full" placeholder="./sing-box-config.json" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">controller 只读/写这个文件。路径变更后立即生效，新路径不存在则自动生成骨架。</p>
          </div>

          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">HTTP 监听地址</label>
          <div>
            <input v-model="form.listen" type="text" class="input input-bordered input-sm w-full" placeholder="127.0.0.1:8080" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">webui 访问地址，格式 host:port。修改后需重启 sing-controller 服务才生效（deb 部署：systemctl restart sing-controller）。</p>
          </div>

          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">日志级别</label>
          <div>
            <select v-model="form.log!.level" class="select select-bordered select-sm w-52">
              <option v-for="l in ['trace', 'debug', 'info', 'warn', 'error', 'fatal']" :key="l" :value="l">{{ l }}</option>
            </select>
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">级别枚举与 sing-box 一致（journald 查看：journalctl -u sing-controller）。</p>
          </div>

          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">端口分配起点 (min_port)</label>
          <div>
            <input v-model.number="form.min_port" type="number" min="1024" max="65535" class="input input-bordered input-sm w-40" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">自动分配端口时从该值开始向上查找第一个空闲端口，默认 8000。</p>
          </div>
        </div>

        <div class="mt-6 mb-3 flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]">
          <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />新建默认值<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
        </div>
        <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 180px minmax(0, 1fr)">
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">默认 inbound type</label>
          <input v-model="form.defaults.inbound_type" type="text" class="input input-bordered input-sm w-full" placeholder="mixed" />
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">默认 outbound type</label>
          <input v-model="form.defaults.outbound_type" type="text" class="input input-bordered input-sm w-full" placeholder="vless" />
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">默认 listen</label>
          <input v-model="form.defaults.listen" type="text" class="input input-bordered input-sm w-full" placeholder="127.0.0.1" />
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">默认 listen_port</label>
          <div>
            <input v-model.number="form.defaults.listen_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-40" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">新建 inbound 时预填；若该端口被占用可在表单里点"自动分配端口"。</p>
          </div>
        </div>

        <div class="mt-6 mb-3 flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]">
          <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />新建 outbound 自动并入组<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
        </div>
        <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 180px minmax(0, 1fr)">
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">并入 Proxy(selector)</label>
          <div>
            <input v-model="form.defaults.attach_to_selector" type="checkbox" class="toggle toggle-primary toggle-sm" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">默认开启：新建 outbound 时自动追加到指定 selector 的成员列表。</p>
          </div>
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">目标组 tag</label>
          <div>
            <input
              v-model="form.defaults.proxy_selector"
              list="selector-tags"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="选择或输入 selector tag"
              :disabled="!form.defaults.attach_to_selector"
            />
            <datalist id="selector-tags">
              <option v-for="t in selectorTags" :key="t" :value="t" />
            </datalist>
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">下拉自动列出现有 type=selector 的 outbound；须先存在同名 selector，否则不生效。</p>
          </div>
        </div>

        <div class="mt-6 mb-3 flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]">
          <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />sing-box 重载<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
        </div>
        <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 180px minmax(0, 1fr)">
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">重载方式 (mode)</label>
          <div>
            <select v-model="form.reload!.mode" class="select select-bordered select-sm w-full">
              <option value="auto">自动适配（systemd → rc-service → OpenWrt service，推荐）</option>
              <option value="none">不启用 (none)</option>
              <option value="systemd">systemd（systemctl reload）</option>
              <option value="pidfile">pidfile（读取 PID 后 kill -HUP）</option>
              <option value="hook">hook（自定义命令）</option>
            </select>
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">
              sing-box 官方重载机制只有 SIGHUP（收到后重载配置）。auto 按环境自动选择：
              systemd（Debian/Ubuntu 等）→ openrc rc-service（Alpine）→ OpenWrt service/procd → SysV service；
              均不可用时返回错误，可改 pidfile/hook。
            </p>
          </div>
          <template v-if="form.reload!.mode === 'systemd' || form.reload!.mode === 'auto'">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">服务名</label>
            <div>
              <input v-model="form.reload!.service" type="text" class="input input-bordered input-sm w-full" placeholder="sing-box" />
              <p class="mt-1 text-xs leading-relaxed text-[#909399]">
                对应 systemctl/rc-service/service 的服务名（Alpine/OpenWrt 上即 init 脚本名）；
                需 sing-controller 用户有对应权限（systemd 通常需 root 或 polkit 授权）。
              </p>
            </div>
          </template>
          <template v-if="form.reload!.mode === 'pidfile'">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">PID 文件路径</label>
            <input v-model="form.reload!.pid_file" type="text" class="input input-bordered input-sm w-full" placeholder="/run/sing-box.pid" />
          </template>
          <template v-if="form.reload!.mode === 'hook'">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">hook 命令</label>
            <div>
              <input v-model="form.reload!.hook" type="text" class="input input-bordered input-sm w-full" placeholder="sudo systemctl reload sing-box" />
              <p class="mt-1 text-xs leading-relaxed text-[#909399]">以 sing-controller 用户执行（sh -c）；如需 sudo 请配置 NOPASSWD。</p>
            </div>
          </template>
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">保存后自动重载</label>
          <div>
            <input v-model="form.reload!.after_save" type="checkbox" class="toggle toggle-primary toggle-sm" :disabled="form.reload!.mode === 'none'" />
            <p class="mt-1 text-xs leading-relaxed text-[#909399]">开启后所有配置写操作保存成功即自动触发重载；失败不影响保存，仅提示。</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
