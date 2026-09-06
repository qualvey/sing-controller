<template>
  <DialogWrapper
    v-model="isDnsServerModalOpen"
    :title="isServerEditMode ? '编辑 DNS Server' : '新建 DNS Server'"
    box-class="max-w-[680px]"
    :body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"
    @update:model-value="(val: boolean) => (!val ? closeDnsServerModal() : null)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList class="self-start shrink-0">
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div ref="formContentRef">
          <div class="mb-4">
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              @click="fillFromJson"
            >
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON 填充（粘贴解析）
            </button>
            <span class="text-base-content/50 ml-2 text-xs">
              粘贴完整 DNS Server JSON，自动解析并填充下方字段
            </span>
          </div>

          <div
            class="grid gap-x-4 gap-y-5"
            style="grid-template-columns: 130px minmax(0, 1fr)"
          >
            <!-- 基础信息 -->
            <label class="text-base-content/70 pt-2 text-sm">
              类型 <span class="text-destructive">*</span>
            </label>
            <SelectField
              v-model="serverForm.type"
              :options="DNS_TYPES"
            />

            <label class="text-base-content/70 pt-2 text-sm">
              tag <span class="text-destructive">*</span>
            </label>
            <input
              v-model="serverForm.tag"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="唯一标识，如 local-dns / cf-doh"
            />

            <!-- 按类型动态展示字段 -->
            <template v-if="serverFieldKeys(serverForm.type).includes('server')">
              <label class="text-base-content/70 pt-2 text-sm">
                server <span class="text-destructive">*</span>
              </label>
              <input
                v-model="serverForm.server"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="IP 或域名（如 1.1.1.1 或 dns.google）"
              />

              <label class="text-base-content/70 pt-2 text-sm">server_port</label>
              <input
                v-model.number="serverForm.server_port"
                type="number"
                min="1"
                max="65535"
                class="input input-bordered input-sm w-40"
              />
            </template>

            <template v-if="serverFieldKeys(serverForm.type).includes('tls_server_name')">
              <label class="text-base-content/70 pt-2 text-sm">TLS SNI</label>
              <input
                v-model="serverForm.tls_server_name"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="如 dns.google（启用 TLS 并校验证书）"
              />
            </template>

            <template v-if="serverFieldKeys(serverForm.type).includes('path')">
              <label class="text-base-content/70 pt-2 text-sm">path</label>
              <input
                v-model="serverForm.path"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="如 /dns-query"
              />
            </template>

            <template v-if="serverFieldKeys(serverForm.type).includes('inet4_range')">
              <label class="text-base-content/70 pt-2 text-sm">inet4_range</label>
              <input
                v-model="serverForm.inet4_range"
                type="text"
                class="input input-bordered input-sm font-mono w-full"
                placeholder="如 198.18.0.0/15"
              />
            </template>

            <template v-if="serverFieldKeys(serverForm.type).includes('inet6_range')">
              <label class="text-base-content/70 pt-2 text-sm">inet6_range</label>
              <input
                v-model="serverForm.inet6_range"
                type="text"
                class="input input-bordered input-sm font-mono w-full"
                placeholder="如 fc00::/18"
              />
            </template>

            <template v-if="serverFieldKeys(serverForm.type).includes('interface')">
              <label class="text-base-content/70 pt-2 text-sm">interface</label>
              <input
                v-model="serverForm.interface"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="网卡名（如 eth0 / en0）"
              />
            </template>

            <label class="text-base-content/70 pt-2 text-sm">detour</label>
            <SelectField
              v-model="serverForm.detour"
              :options="outboundTags"
              placeholder="经出站代理（可选）"
            />

            <!-- 拨号选项（DialerOptions） -->
            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              拨号选项（DialerOptions）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">bind_interface</label>
            <input
              v-model="serverForm.bind_interface"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="绑定网卡名（可选）"
            />

            <label class="text-base-content/70 pt-2 text-sm">connect_timeout</label>
            <input
              v-model="serverForm.connect_timeout"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 5s（可选）"
            />

            <label class="text-base-content/70 pt-2 text-sm">routing_mark</label>
            <input
              v-model.number="serverForm.routing_mark"
              type="number"
              min="0"
              class="input input-bordered input-sm w-40"
            />

            <label class="text-base-content/70 pt-2 text-sm">reuse_addr</label>
            <Switch
              v-model="serverForm.reuse_addr"
              class="self-center"
            />

            <label class="text-base-content/70 pt-2 text-sm">udp_fragment</label>
            <Switch
              v-model="serverForm.udp_fragment"
              class="self-center"
            />

            <label class="text-base-content/70 pt-2 text-sm">network_strategy</label>
            <SelectField
              v-model="serverForm.network_strategy"
              :options="STRATEGY_OPTIONS"
              placeholder="不指定"
            />

            <label class="text-base-content/70 pt-2 text-sm">network_type</label>
            <ChipInput
              v-model="serverForm.network_type"
              :suggestions="['wifi', 'cellular', 'ethernet', 'other']"
              placeholder="wifi/cellular/ethernet/other"
            />

            <!-- 域名解析器（domain_resolver） -->
            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              域名解析器（domain_resolver）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">resolver server</label>
            <div class="flex flex-col gap-1">
              <SelectField
                v-model="serverForm.dr_server"
                :options="dnsTags"
                placeholder="选择 DNS server（如 local/ali）"
              />
              <p class="text-base-content/50 text-xs">
                server 为域名时，用该 resolver 解析其 IP（避免自举）
              </p>
            </div>

            <template v-if="serverForm.dr_server">
              <label class="text-base-content/70 pt-2 text-sm">resolver timeout</label>
              <input
                v-model="serverForm.dr_timeout"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="如 5s"
              />

              <label class="text-base-content/70 pt-2 text-sm">resolver strategy</label>
              <SelectField
                v-model="serverForm.dr_strategy"
                :options="STRATEGY_OPTIONS"
                placeholder="不指定"
              />

              <label class="text-base-content/70 pt-2 text-sm">禁用缓存</label>
              <Switch
                v-model="serverForm.dr_disable_cache"
                class="self-center"
              />

              <label class="text-base-content/70 pt-2 text-sm">禁用乐观缓存</label>
              <Switch
                v-model="serverForm.dr_disable_optimistic_cache"
                class="self-center"
              />

              <label class="text-base-content/70 pt-2 text-sm">rewrite_ttl</label>
              <input
                v-model.number="serverForm.dr_rewrite_ttl"
                type="number"
                min="0"
                class="input input-bordered input-sm w-40"
              />

              <label class="text-base-content/70 pt-2 text-sm">client_subnet</label>
              <input
                v-model="serverForm.dr_client_subnet"
                type="text"
                class="input input-bordered input-sm font-mono w-full"
                placeholder="如 1.2.3.4/24"
              />
            </template>

            <!-- Hosts predefined -->
            <template v-if="serverForm.type === 'hosts'">
              <div
                class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
                style="grid-column: 1 / -1"
              >
                <span class="border-base-content/10 h-px flex-1 border-t" />
                预定义映射（Predefined Hosts）
                <span class="border-base-content/10 h-px flex-1 border-t" />
              </div>

              <label class="text-base-content/70 pt-2 text-sm">predefined (JSON)</label>
              <textarea
                v-model="serverForm.predefined_json"
                rows="4"
                class="textarea textarea-bordered font-mono text-xs w-full"
                placeholder='{"example.com": "1.2.3.4"}'
              />
            </template>
          </div>

          <p
            v-if="validateMsg"
            class="text-destructive mt-3 text-[0.8rem] font-medium"
          >
            {{ validateMsg }}
          </p>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
          :height="sourceEditorHeight"
        />
      </TabsContent>
    </TabsRoot>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <button
            v-if="isServerEditMode && editingServer?.tag"
            type="button"
            class="btn btn-ghost btn-sm text-error hover:bg-error/10"
            :disabled="saving"
            @click="handleDelete"
          >
            删除
          </button>
        </div>
        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            @click="closeDnsServerModal"
          >
            取消
          </button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="saving"
            @click="save"
          >
            保存
          </button>
        </div>
      </div>
    </template>
  </DialogWrapper>

  <!-- 从 JSON 填充弹窗 -->
  <DialogWrapper
    v-model="jsonDialog"
    title="从 JSON 填充"
    box-class="max-w-[560px]"
  >
    <textarea
      v-model="jsonText"
      rows="10"
      class="textarea textarea-bordered w-full font-mono text-xs"
      placeholder='{"type":"udp","tag":"remote-dns","server":"8.8.8.8","server_port":53}'
    />
    <p class="text-base-content/50 mt-1 text-xs">
      解析成功后填充表单字段（请检查解析结果）
    </p>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          @click="jsonDialog = false"
        >
          取消
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          @click="doFillFromJson"
        >
          解析并填充
        </button>
      </div>
    </template>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api } from '@/api'
import {
  deleteDnsServer,
  DNS_TYPES,
  dnsTags,
  outboundTags,
  saveDnsServer,
  saving,
  serverFieldKeys,
  STRATEGY_OPTIONS,
} from '@/assembly/dns'
import ChipInput from '@/components/common/ChipInput.vue'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import {
  closeDnsServerModal,
  editingServer,
  isDnsServerModalOpen,
  isServerEditMode,
} from '@/composables/dnsManagement'
import { showToast } from '@/helper/toast'
import { ClipboardPasteIcon } from '@lucide/vue'
import { useResizeObserver } from '@vueuse/core'
import { computed, reactive, ref, watch } from 'vue'

const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const validateMsg = ref('')
const formContentRef = ref<HTMLElement>()
const sharedTabHeight = ref<number>(460)

const jsonDialog = ref(false)
const jsonText = ref('')

const sourceEditorHeight = computed(() => {
  if (typeof window === 'undefined') return 440
  const maxAvailable = Math.max(300, Math.floor(window.innerHeight * 0.85 - 170))
  return Math.min(sharedTabHeight.value, maxAvailable)
})

useResizeObserver(formContentRef, (entries) => {
  const entry = entries[0]
  if (entry && activeTab.value === 'form' && entry.contentRect.height > 60) {
    sharedTabHeight.value = Math.max(300, Math.round(entry.contentRect.height))
  }
})

const serverForm = reactive<Record<string, any>>({
  type: 'udp',
  tag: '',
  server: '',
  server_port: 53,
  detour: '',
  tls_server_name: '',
  path: '',
  inet4_range: '',
  inet6_range: '',
  interface: '',
  predefined_json: '',
  bind_interface: '',
  connect_timeout: '',
  routing_mark: 0,
  reuse_addr: false,
  udp_fragment: false,
  network_strategy: '',
  network_type: [] as string[],
  dr_server: '',
  dr_timeout: '',
  dr_strategy: '',
  dr_disable_cache: false,
  dr_disable_optimistic_cache: false,
  dr_rewrite_ttl: 0,
  dr_client_subnet: '',
})

const resetServerForm = () => {
  validateMsg.value = ''
  serverForm.type = 'udp'
  serverForm.tag = ''
  serverForm.server = ''
  serverForm.server_port = 53
  serverForm.detour = ''
  serverForm.tls_server_name = ''
  serverForm.path = ''
  serverForm.inet4_range = ''
  serverForm.inet6_range = ''
  serverForm.interface = ''
  serverForm.predefined_json = ''
  serverForm.bind_interface = ''
  serverForm.connect_timeout = ''
  serverForm.routing_mark = 0
  serverForm.reuse_addr = false
  serverForm.udp_fragment = false
  serverForm.network_strategy = ''
  serverForm.network_type = []
  serverForm.dr_server = ''
  serverForm.dr_timeout = ''
  serverForm.dr_strategy = ''
  serverForm.dr_disable_cache = false
  serverForm.dr_disable_optimistic_cache = false
  serverForm.dr_rewrite_ttl = 0
  serverForm.dr_client_subnet = ''
}

const fillForm = (row: Record<string, any>) => {
  resetServerForm()
  serverForm.type = String(row.type || 'udp')
  serverForm.tag = String(row.tag || '')
  if (row.server != null) serverForm.server = String(row.server)
  if (row.server_port != null) serverForm.server_port = Number(row.server_port)
  if (row.detour != null) serverForm.detour = String(row.detour)
  if (row.tls && typeof row.tls === 'object') {
    serverForm.tls_server_name = String((row.tls as any).server_name || '')
  }
  if (row.path != null) serverForm.path = String(row.path)
  if (row.inet4_range != null) serverForm.inet4_range = String(row.inet4_range)
  if (row.inet6_range != null) serverForm.inet6_range = String(row.inet6_range)
  if (row.interface != null) serverForm.interface = String(row.interface)
  if (row.predefined != null) serverForm.predefined_json = JSON.stringify(row.predefined, null, 2)
  if (row.bind_interface != null) serverForm.bind_interface = String(row.bind_interface)
  if (row.connect_timeout != null) serverForm.connect_timeout = String(row.connect_timeout)
  if (row.routing_mark != null) serverForm.routing_mark = Number(row.routing_mark)
  serverForm.reuse_addr = row.reuse_addr === true
  serverForm.udp_fragment = row.udp_fragment === true
  if (row.network_strategy != null) serverForm.network_strategy = String(row.network_strategy)
  serverForm.network_type = Array.isArray(row.network_type)
    ? row.network_type.map(String)
    : row.network_type
      ? [String(row.network_type)]
      : []
  const dr = row.domain_resolver
  if (dr && typeof dr === 'object') {
    serverForm.dr_server = typeof dr.server === 'string' ? dr.server : ''
    if (dr.timeout != null) serverForm.dr_timeout = String(dr.timeout)
    if (dr.strategy != null) serverForm.dr_strategy = String(dr.strategy)
    serverForm.dr_disable_cache = dr.disable_cache === true
    serverForm.dr_disable_optimistic_cache = dr.disable_optimistic_cache === true
    if (dr.rewrite_ttl != null) serverForm.dr_rewrite_ttl = Number(dr.rewrite_ttl)
    if (dr.client_subnet != null) serverForm.dr_client_subnet = String(dr.client_subnet)
  }
}

function buildServer(): Record<string, any> {
  const s: Record<string, any> = { type: serverForm.type, tag: serverForm.tag.trim() }
  if (serverForm.detour) s.detour = serverForm.detour
  if (serverForm.bind_interface.trim()) s.bind_interface = serverForm.bind_interface.trim()
  if (serverForm.connect_timeout.trim()) s.connect_timeout = serverForm.connect_timeout.trim()
  if (serverForm.routing_mark) s.routing_mark = Number(serverForm.routing_mark)
  if (serverForm.reuse_addr) s.reuse_addr = true
  if (serverForm.udp_fragment) s.udp_fragment = true
  if (serverForm.network_strategy) s.network_strategy = serverForm.network_strategy
  if (serverForm.network_type.length) s.network_type = [...serverForm.network_type]

  if (serverForm.dr_server) {
    const dr: Record<string, any> = { server: serverForm.dr_server }
    if (serverForm.dr_timeout.trim()) dr.timeout = serverForm.dr_timeout.trim()
    if (serverForm.dr_strategy) dr.strategy = serverForm.dr_strategy
    if (serverForm.dr_disable_cache) dr.disable_cache = true
    if (serverForm.dr_disable_optimistic_cache) dr.disable_optimistic_cache = true
    if (serverForm.dr_rewrite_ttl) dr.rewrite_ttl = Number(serverForm.dr_rewrite_ttl)
    if (serverForm.dr_client_subnet.trim()) dr.client_subnet = serverForm.dr_client_subnet.trim()
    s.domain_resolver = dr
  }

  const keys = serverFieldKeys(serverForm.type)
  if (keys.includes('server') && serverForm.server.trim()) s.server = serverForm.server.trim()
  if (keys.includes('server_port') && serverForm.server_port)
    s.server_port = Number(serverForm.server_port)
  if (keys.includes('tls_server_name') && serverForm.tls_server_name.trim()) {
    s.tls = { enabled: true, server_name: serverForm.tls_server_name.trim() }
  }
  if (keys.includes('path') && serverForm.path.trim()) s.path = serverForm.path.trim()
  if (keys.includes('inet4_range') && serverForm.inet4_range.trim())
    s.inet4_range = serverForm.inet4_range.trim()
  if (keys.includes('inet6_range') && serverForm.inet6_range.trim())
    s.inet6_range = serverForm.inet6_range.trim()
  if (keys.includes('interface') && serverForm.interface.trim())
    s.interface = serverForm.interface.trim()

  if (serverForm.predefined_json.trim()) {
    try {
      s.predefined = JSON.parse(serverForm.predefined_json.trim())
    } catch (e) {
      throw new Error(`predefined JSON 格式错误：${(e as Error).message}`)
    }
  }
  return s
}

const fillFromJson = () => {
  jsonText.value = ''
  jsonDialog.value = true
}

const doFillFromJson = async () => {
  try {
    let parsed: Record<string, any>
    try {
      const res = await api.parseJson(jsonText.value)
      parsed = (res?.data || res) as Record<string, any>
    } catch {
      parsed = JSON.parse(jsonText.value)
    }
    if (!parsed || typeof parsed !== 'object') {
      throw new Error('解析内容必须为 JSON 对象')
    }
    fillForm(parsed)
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

watch(
  () => [isDnsServerModalOpen.value, editingServer.value],
  ([visible]) => {
    if (!visible) return
    activeTab.value = 'form'
    validateMsg.value = ''
    const row = editingServer.value
    if (row && typeof row === 'object') {
      sourceJson.value = JSON.stringify(row, null, 2)
      fillForm(row)
    } else {
      sourceJson.value = '{}'
      resetServerForm()
    }
  },
  { immediate: true },
)

watch(activeTab, (tab) => {
  if (tab === 'source') {
    if (!srcTab.value?.isDirty()) {
      try {
        sourceJson.value = JSON.stringify(buildServer(), null, 2)
      } catch {
        // ignore build errors when previewing
      }
    }
  }
})

const save = async () => {
  validateMsg.value = ''
  if (!serverForm.tag.trim()) {
    validateMsg.value = '请填写 tag'
    showToast('请填写 tag', 'warning')
    return
  }

  try {
    const payload =
      activeTab.value === 'source' || srcTab.value?.isDirty()
        ? JSON.parse(srcTab.value?.getText() || '{}')
        : buildServer()

    const targetTag = isServerEditMode.value
      ? String(editingServer.value?.tag || serverForm.tag)
      : serverForm.tag
    const ok = await saveDnsServer(targetTag, payload, isServerEditMode.value)
    if (ok) closeDnsServerModal()
  } catch (e) {
    validateMsg.value = (e as Error).message || '保存失败'
    showToast((e as Error).message || '保存失败', 'error')
  }
}

const handleDelete = async () => {
  if (!editingServer.value?.tag) return
  const ok = await deleteDnsServer(editingServer.value.tag)
  if (ok) {
    closeDnsServerModal()
  }
}
</script>
