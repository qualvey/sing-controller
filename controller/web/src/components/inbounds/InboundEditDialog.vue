<template>
  <DialogWrapper
    v-model="isInboundModalOpen"
    :title="isEditMode ? '编辑 Inbound' : '新建 Inbound'"
    box-class="max-w-[680px]"
    @update:model-value="(val) => (!val ? closeInboundModal() : null)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
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
            粘贴完整 inbound JSON，自动解析并填充下方字段
          </span>
        </div>

        <div
          class="grid gap-x-4 gap-y-5"
          style="grid-template-columns: 130px minmax(0, 1fr)"
        >
          <label class="text-base-content/70 pt-2 text-sm">类型</label>
          <template v-if="inboundTypes.length">
            <SelectField
              v-model="form.type"
              :disabled="isEditMode"
              :options="inboundTypes"
            />
          </template>
          <div
            v-else
            class="alert alert-warning py-2 text-xs"
          >
            当前没有后端连接，无法使用该功能
          </div>

          <label class="text-base-content/70 pt-2 text-sm">
            tag <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.tag"
            type="text"
            class="input input-bordered input-sm w-full"
            placeholder="唯一标识，如 mixed-in"
          />

          <!-- mixed -->
          <template v-if="isMixed">
            <label class="text-base-content/70 pt-2 text-sm">
              listen <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.listen"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="监听地址，如 127.0.0.1"
            />

            <label class="text-base-content/70 pt-2 text-sm">
              listen_port <span class="text-destructive">*</span>
            </label>
            <div class="flex gap-2">
              <input
                v-model.number="form.listen_port"
                type="number"
                min="1"
                max="65535"
                class="input input-bordered input-sm w-full"
              />
              <button
                type="button"
                class="btn btn-ghost btn-sm shrink-0"
                :disabled="allocating"
                @click="() => handleAllocatePort()"
              >
                <WandIcon
                  class="h-4 w-4"
                  :class="{ 'animate-spin': allocating }"
                />
                自动分配
              </button>
            </div>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              用户（可选）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <template
              v-for="(u, i) in form.users"
              :key="i"
            >
              <label class="text-base-content/70 pt-2 text-sm">用户 {{ i + 1 }}</label>
              <div class="flex items-center gap-2">
                <input
                  v-model="u.username"
                  type="text"
                  class="input input-bordered input-sm flex-1"
                  placeholder="用户名"
                />
                <div class="flex flex-1 items-center gap-1">
                  <input
                    v-model="u.password"
                    :type="showUserPasswords[i] ? 'text' : 'password'"
                    class="input input-bordered input-sm flex-1"
                    placeholder="密码"
                  />
                  <button
                    type="button"
                    class="btn btn-ghost btn-xs shrink-0"
                    :title="showUserPasswords[i] ? '隐藏密码' : '显示明文'"
                    @click="showUserPasswords[i] = !showUserPasswords[i]"
                  >
                    <EyeSlashIcon
                      v-if="showUserPasswords[i]"
                      class="h-3.5 w-3.5"
                    />
                    <EyeIcon
                      v-else
                      class="h-3.5 w-3.5"
                    />
                  </button>
                </div>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs text-error shrink-0"
                  title="删除用户"
                  @click="form.users.splice(i, 1)"
                >
                  <TrashIcon class="h-3.5 w-3.5" />
                </button>
              </div>
            </template>
            <span style="grid-column: 2">
              <button
                type="button"
                class="btn btn-ghost btn-xs gap-1"
                @click="form.users.push({ username: '', password: '' })"
              >
                <UserPlusIcon class="h-3.5 w-3.5" />
                添加用户
              </button>
            </span>
          </template>

          <!-- shadowsocks -->
          <template v-else-if="isShadowsocks">
            <label class="text-base-content/70 pt-2 text-sm">
              listen <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.listen"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="监听地址，默认 ::"
            />

            <label class="text-base-content/70 pt-2 text-sm">
              listen_port <span class="text-destructive">*</span>
            </label>
            <div class="flex gap-2">
              <input
                v-model.number="form.listen_port"
                type="number"
                min="1"
                max="65535"
                class="input input-bordered input-sm w-full"
              />
              <button
                type="button"
                class="btn btn-ghost btn-sm shrink-0"
                :disabled="allocating"
                @click="() => handleAllocatePort()"
              >
                <WandIcon
                  class="h-4 w-4"
                  :class="{ 'animate-spin': allocating }"
                />
                自动分配
              </button>
            </div>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              协议
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">
              method <span class="text-destructive">*</span>
            </label>
            <div class="flex flex-col gap-1">
              <SelectField
                v-model="form.ssMethod"
                :options="SS_METHODS"
              />
              <p class="text-base-content/60 text-xs">
                默认 chacha20-ietf-poly1305；none 为无加密（仅调试用），2022-blake3-* 为 SIP022 系
              </p>
            </div>

            <template v-if="!isSsNoEncryption">
              <label class="text-base-content/70 pt-2 text-sm">
                password <span class="text-destructive">*</span>
              </label>
              <div class="flex flex-col gap-1">
                <div class="flex gap-2">
                  <input
                    v-model="form.ssPassword"
                    :type="showSsPassword ? 'text' : 'password'"
                    class="input input-bordered input-sm font-mono text-xs w-full"
                    placeholder="新建时自动生成"
                  />
                  <button
                    type="button"
                    class="btn btn-ghost btn-sm shrink-0"
                    :title="showSsPassword ? '隐藏密码' : '显示明文'"
                    @click="showSsPassword = !showSsPassword"
                  >
                    <EyeSlashIcon
                      v-if="showSsPassword"
                      class="h-4 w-4"
                    />
                    <EyeIcon
                      v-else
                      class="h-4 w-4"
                    />
                  </button>
                  <button
                    type="button"
                    class="btn btn-ghost btn-sm shrink-0"
                    :disabled="generating"
                    @click="handleGenerateSsPassword()"
                  >
                    <KeyRoundIcon
                      class="h-4 w-4"
                      :class="{ 'animate-spin': generating }"
                    />
                    重新生成
                  </button>
                </div>
                <p class="text-base-content/60 text-xs">
                  默认自动生成 16 字节随机密码（base64，如 8JCsPssfgS8tiRwiMlhARg==）
                </p>
              </div>
            </template>

            <p
              v-else
              class="text-base-content/60 text-xs"
              style="grid-column: 2"
            >
              method 为 none（无加密）时不需要密码
            </p>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              用户（Users 页统一管理，可选）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">绑定用户</label>
            <div class="flex flex-col gap-1">
              <ChipInput
                v-model="selectedPoolUsers"
                :suggestions="poolUsers.map((u) => u.name)"
                placeholder="从用户池选择（多选）"
              />
              <p class="text-base-content/60 text-xs">
                绑定后注入 users[]（name+password）；非 2022 method 以用户密码为准，2022 method 以顶部 password 为主密钥
              </p>
            </div>
          </template>

          <!-- tuic -->
          <template v-else-if="isTuic">
            <label class="text-base-content/70 pt-2 text-sm">
              listen <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.listen"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="监听地址，如 0.0.0.0"
            />

            <label class="text-base-content/70 pt-2 text-sm">
              listen_port <span class="text-destructive">*</span>
            </label>
            <div class="flex gap-2">
              <input
                v-model.number="form.listen_port"
                type="number"
                min="1"
                max="65535"
                class="input input-bordered input-sm w-full"
              />
              <button
                type="button"
                class="btn btn-ghost btn-sm shrink-0"
                :disabled="allocating"
                @click="() => handleAllocatePort()"
              >
                <WandIcon
                  class="h-4 w-4"
                  :class="{ 'animate-spin': allocating }"
                />
                自动分配
              </button>
            </div>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              用户（Users 页统一管理）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">绑定用户</label>
            <div class="flex flex-col gap-1">
              <ChipInput
                v-model="selectedPoolUsers"
                :suggestions="poolUsers.map((u) => u.name)"
                placeholder="从用户池选择（多选）"
              />
              <p class="text-base-content/60 text-xs">
                用户统一在 Users 页创建/编辑；此处仅选择要绑定到本入站的用户，保存后自动注入 users[]（按类型取用字段）
              </p>
            </div>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              协议
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">拥塞控制</label>
            <SelectField
              v-model="form.congestionControl"
              :options="['cubic', 'new_reno', 'bbr']"
            />

            <label class="text-base-content/70 pt-2 text-sm">认证超时</label>
            <input
              v-model="form.authTimeout"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 3s"
            />

            <label class="text-base-content/70 pt-2 text-sm">心跳间隔</label>
            <input
              v-model="form.heartbeat"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 10s"
            />

            <label class="text-base-content/70 pt-2 text-sm">0-RTT 握手</label>
            <Switch
              v-model="form.zeroRTT"
              class="self-center"
            />

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              TLS（tuic 强制启用）
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">证书来源</label>
            <div class="flex flex-col gap-1">
              <SelectField
                v-model="form.certificateProvider"
                :disabled="!!form.certificatePath.trim() || !!form.keyPath.trim()"
                :options="certProviders"
                placeholder="选择 Certificate Provider（推荐）"
              />
              <p class="text-base-content/60 text-xs">
                证书 Provider 在「证书」页管理（引用其 tag）；与下方手动证书路径二选一
              </p>
            </div>

            <label class="text-base-content/70 pt-2 text-sm">server_name</label>
            <input
              v-model="form.serverName"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="证书域名"
            />

            <label class="text-base-content/70 pt-2 text-sm">证书路径</label>
            <input
              v-model="form.certificatePath"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="/etc/sing-box/cert.pem"
              :disabled="!!form.certificateProvider"
            />

            <label class="text-base-content/70 pt-2 text-sm">私钥路径</label>
            <input
              v-model="form.keyPath"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="/etc/sing-box/key.pem"
              :disabled="!!form.certificateProvider"
            />

            <label class="text-base-content/70 pt-2 text-sm">ALPN</label>
            <span class="badge badge-info mt-1 w-fit">h3</span>

            <label class="text-base-content/70 pt-2 text-sm">TLS 版本</label>
            <div class="flex gap-2">
              <SelectField
                v-model="form.minVersion"
                class="flex-1"
                :options="TLS_VERSIONS"
                placeholder="最小"
              />
              <SelectField
                v-model="form.maxVersion"
                class="flex-1"
                :options="TLS_VERSIONS"
                placeholder="最大"
              />
            </div>

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              QUIC
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2 text-sm">空闲超时</label>
            <input
              v-model="form.idleTimeout"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 30s"
            />

            <label class="text-base-content/70 pt-2 text-sm">保活周期</label>
            <input
              v-model="form.keepAlivePeriod"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 15s"
            />

            <label class="text-base-content/70 pt-2 text-sm">初始包大小</label>
            <input
              v-model.number="form.initialPacketSize"
              type="number"
              min="0"
              max="10000"
              class="input input-bordered input-sm w-40"
            />

            <label class="text-base-content/70 pt-2 text-sm">禁用 MTU 发现</label>
            <Switch
              v-model="form.disablePathMTU"
              class="self-center"
            />
          </template>

          <!-- 其他类型 -->
          <template v-else>
            <label class="text-base-content/70 pt-2 text-sm">listen</label>
            <input
              v-model="form.listen"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="监听地址（可选）"
            />

            <label class="text-base-content/70 pt-2 text-sm">listen_port</label>
            <div class="flex gap-2">
              <input
                v-model.number="form.listen_port"
                type="number"
                min="1"
                max="65535"
                class="input input-bordered input-sm w-full"
              />
              <button
                type="button"
                class="btn btn-ghost btn-sm shrink-0"
                :disabled="allocating"
                @click="() => handleAllocatePort()"
              >
                <WandIcon
                  class="h-4 w-4"
                  :class="{ 'animate-spin': allocating }"
                />
                自动分配
              </button>
            </div>
          </template>
        </div>

        <p
          v-if="validateMsg"
          class="text-destructive mt-3 text-[0.8rem] font-medium"
        >
          {{ validateMsg }}
        </p>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
        />
      </TabsContent>
    </TabsRoot>

    <div class="mt-5 flex justify-end gap-2">
      <button
        type="button"
        class="btn btn-ghost btn-sm"
        @click="closeInboundModal"
      >
        取消
      </button>
      <button
        type="button"
        class="btn btn-primary btn-sm"
        :disabled="saving"
        @click="handleSave"
      >
        保存
      </button>
    </div>
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
      placeholder='{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":2080}'
    />
    <p class="text-base-content/50 mt-1 text-xs">
      解析成功后填充表单字段（type 变化会重置表单，请最后检查）
    </p>
    <div class="mt-4 flex justify-end gap-2">
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
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api, type Inbound } from '@/api'
import {
  allocateAvailablePort,
  certProviders,
  createInbound,
  fetchCertProviders,
  fetchDefaults,
  fetchPoolUsers,
  generateSsPassword,
  inboundTypes,
  poolUsers,
  SS_DEFAULT_LISTEN,
  SS_DEFAULT_METHOD,
  SS_DEFAULT_PORT,
  SS_METHODS,
  syncBoundUsers,
  TLS_VERSIONS,
  updateInbound,
} from '@/assembly/inbounds'
import ChipInput from '@/components/common/ChipInput.vue'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import {
  closeInboundModal,
  editingInboundTag,
  isEditMode,
  isInboundModalOpen,
} from '@/composables/inboundManagement'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import {
  ClipboardPasteIcon,
  KeyRoundIcon,
  TrashIcon,
  UserPlusIcon,
  WandIcon,
} from 'lucide-vue-next'
import { computed, reactive, ref, watch } from 'vue'


const statusStore = useStatusStore()

const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const saving = ref(false)
const allocating = ref(false)
const generating = ref(false)
const suppressTypeWatch = ref(false)
const validateMsg = ref('')
const showSsPassword = ref(false)
const showUserPasswords = reactive<Record<number, boolean>>({})

// 从 JSON 填充弹窗
const jsonDialog = ref(false)
const jsonText = ref('')

const selectedPoolUsers = ref<string[]>([])

interface InboundForm {
  type: string
  tag: string
  listen: string
  listen_port?: number
  users: { username: string; password: string }[]
  tuicUsers: { name: string; uuid: string; password: string }[]
  congestionControl: string
  authTimeout: string
  heartbeat: string
  zeroRTT: boolean
  tlsEnabled: boolean
  certificateProvider: string
  serverName: string
  insecure: boolean
  certificatePath: string
  keyPath: string
  alpn: string
  minVersion: string
  maxVersion: string
  idleTimeout: string
  keepAlivePeriod: string
  initialPacketSize: number
  disablePathMTU: boolean
  ssMethod: string
  ssPassword: string
}

const form = reactive<InboundForm>({
  type: '',
  tag: '',
  listen: '',
  listen_port: undefined,
  users: [{ username: '', password: '' }],
  tuicUsers: [{ name: '', uuid: '', password: '' }],
  congestionControl: 'bbr',
  authTimeout: '',
  heartbeat: '',
  zeroRTT: false,
  tlsEnabled: true,
  certificateProvider: '',
  serverName: '',
  insecure: false,
  certificatePath: '',
  keyPath: '',
  alpn: '',
  minVersion: '',
  maxVersion: '',
  idleTimeout: '',
  keepAlivePeriod: '',
  initialPacketSize: 1452,
  disablePathMTU: false,
  ssMethod: SS_DEFAULT_METHOD,
  ssPassword: '',
})

const isMixed = computed(() => form.type === 'mixed')
const isTuic = computed(() => form.type === 'tuic')
const isShadowsocks = computed(() => form.type === 'shadowsocks')
const isSsNoEncryption = computed(() => form.ssMethod === 'none')

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

function resetForm(type: string) {
  form.type = type
  form.tag = ''
  form.listen = ''
  form.listen_port = 443
  form.users = [{ username: '', password: '' }]
  form.ssMethod = SS_DEFAULT_METHOD
  form.ssPassword = ''
}

function fillForm(obj: Inbound) {
  form.type = str(obj.type)
  form.tag = str(obj.tag)
  form.listen = str(obj.listen)
  form.listen_port = num(obj.listen_port)
  if (form.type === 'mixed') {
    form.users = Array.isArray(obj.users)
      ? obj.users.map((u) => {
          const rec = (u ?? {}) as Record<string, unknown>
          return { username: str(rec.username), password: str(rec.password) }
        })
      : [{ username: '', password: '' }]
  } else {
    form.users = []
  }
  if (form.type === 'shadowsocks') {
    const s = (obj ?? {}) as Record<string, unknown>
    form.ssMethod = str(s.method) || SS_DEFAULT_METHOD
    form.ssPassword = str(s.password)
  }
  if (form.type === 'tuic') {
    const t = (obj ?? {}) as Record<string, unknown>
    form.tuicUsers = Array.isArray(t.users)
      ? (t.users as Record<string, unknown>[]).map((u) => ({
          name: str(u.name),
          uuid: str(u.uuid),
          password: str(u.password),
        }))
      : [{ name: '', uuid: '', password: '' }]
    form.congestionControl = str(t.congestion_control) || 'bbr'
    form.authTimeout = str(t.auth_timeout)
    form.heartbeat = str(t.heartbeat)
    form.zeroRTT = Boolean(t.zero_rtt_handshake)
    const tls = (t.tls ?? {}) as Record<string, unknown>
    form.tlsEnabled = true
    form.certificateProvider = str(tls.certificate_provider)
    form.serverName = str(tls.server_name)
    form.insecure = Boolean(tls.insecure)
    form.certificatePath = str(tls.certificate_path)
    form.keyPath = str(tls.key_path)
    form.alpn = 'h3'
    form.minVersion = str(tls.min_version)
    form.maxVersion = str(tls.max_version)
    form.idleTimeout = str(t.idle_timeout)
    form.keepAlivePeriod = str(t.keep_alive_period)
    form.initialPacketSize =
      typeof t.initial_packet_size === 'number' ? t.initial_packet_size : 1452
    form.disablePathMTU = Boolean(t.disable_path_mtu_discovery)
  }
}

watch(
  () => form.type,
  async (t) => {
    if (!suppressTypeWatch.value && isInboundModalOpen.value && !isEditMode.value) {
      resetForm(t)
      if (t === 'shadowsocks') {
        form.listen = SS_DEFAULT_LISTEN
        form.listen_port = SS_DEFAULT_PORT
        form.ssPassword = await generateSsPassword(false)
      } else {
        const defListen = await fetchDefaults()
        if (defListen) form.listen = defListen
      }
    }
  },
)

watch(isInboundModalOpen, async (isOpen) => {
  if (!isOpen) return

  activeTab.value = 'form'
  validateMsg.value = ''
  showSsPassword.value = false

  if (isEditMode.value && editingInboundTag.value) {
    try {
      const data = await api.getInbound(editingInboundTag.value)
      sourceJson.value = JSON.stringify(data, null, 2)
      fillForm(data)
      if (isTuic.value || isShadowsocks.value) {
        void fetchPoolUsers().then(() => {
          selectedPoolUsers.value = poolUsers.value
            .filter((u) => u.bound_inbounds?.includes(data.tag))
            .map((u) => u.name)
        })
        if (isTuic.value) void fetchCertProviders()
      }
    } catch (e) {
      showToast((e as Error).message || '加载失败', 'error')
      closeInboundModal()
    }
  } else {
    sourceJson.value = '{}'
    const def = statusStore.status?.defaults
    const defType = def?.inbound_type
    resetForm(
      defType && inboundTypes.value.includes(defType)
        ? defType
        : inboundTypes.value[0] || 'mixed',
    )
    const defListen = await fetchDefaults()
    if (defListen) form.listen = defListen

    if (form.type === 'shadowsocks') {
      form.listen = SS_DEFAULT_LISTEN
      form.listen_port = SS_DEFAULT_PORT
      form.ssPassword = await generateSsPassword(false)
    } else {
      form.listen_port = 443
    }
    selectedPoolUsers.value = []
    void fetchPoolUsers()
    void fetchCertProviders()
  }
})

const handleAllocatePort = async () => {
  allocating.value = true
  try {
    const p = await allocateAvailablePort(true)
    if (p != null) form.listen_port = p
  } finally {
    allocating.value = false
  }
}

const handleGenerateSsPassword = async () => {
  generating.value = true
  try {
    form.ssPassword = await generateSsPassword(true)
  } finally {
    generating.value = false
  }
}

const fillFromJson = () => {
  jsonText.value = ''
  jsonDialog.value = true
}

const doFillFromJson = async () => {
  try {
    const res = await api.parseJson(jsonText.value)
    suppressTypeWatch.value = true
    fillForm(res.data as Inbound)
    suppressTypeWatch.value = false
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

function validateForm(): string {
  if (!form.tag.trim()) return 'tag 必填'
  if (isMixed.value || isShadowsocks.value || isTuic.value) {
    if (!form.listen.trim()) return 'listen 必填'
    if (
      typeof form.listen_port !== 'number' ||
      form.listen_port < 1 ||
      form.listen_port > 65535
    ) {
      return 'listen_port 必填（1-65535）'
    }
  }
  if (
    isShadowsocks.value &&
    !isSsNoEncryption.value &&
    !form.ssPassword.trim() &&
    !selectedPoolUsers.value.length
  ) {
    return 'password 必填（method 为 none 或绑定用户时除外）'
  }
  return ''
}

function buildBody(): Inbound {
  const body: Inbound = { type: form.type, tag: form.tag.trim() }
  if (form.listen.trim()) body.listen = form.listen.trim()
  if (typeof form.listen_port === 'number') body.listen_port = form.listen_port

  if (form.type === 'mixed') {
    const users = form.users
      .map((u) => ({ username: u.username.trim(), password: u.password.trim() }))
      .filter((u) => u.username || u.password)
    if (users.some((u) => !u.username || !u.password)) {
      throw new Error('用户名和密码需同时填写')
    }
    if (users.length) body.users = users
  }
  if (form.type === 'tuic') {
    const users = form.tuicUsers
      .map((u) => ({ name: u.name.trim(), uuid: u.uuid.trim(), password: u.password.trim() }))
      .filter((u) => u.uuid || u.password)
    if (users.some((u) => !u.uuid || !u.password)) {
      throw new Error('tuic 用户必须填写 uuid 和 password')
    }
    if (users.length) body.users = users
    if (form.congestionControl) body.congestion_control = form.congestionControl
    if (form.authTimeout.trim()) body.auth_timeout = form.authTimeout.trim()
    if (form.heartbeat.trim()) body.heartbeat = form.heartbeat.trim()
    if (form.zeroRTT) body.zero_rtt_handshake = true
    const tls: Record<string, unknown> = {}
    tls.enabled = true
    if (form.certificateProvider.trim())
      tls.certificate_provider = form.certificateProvider.trim()
    if (form.serverName.trim()) tls.server_name = form.serverName.trim()
    if (form.certificatePath.trim()) tls.certificate_path = form.certificatePath.trim()
    if (form.keyPath.trim()) tls.key_path = form.keyPath.trim()
    if (form.alpn.trim()) tls.alpn = form.alpn.split(',').map((s) => s.trim()).filter(Boolean)
    if (form.minVersion) tls.min_version = form.minVersion
    if (form.maxVersion) tls.max_version = form.maxVersion
    if (Object.keys(tls).length) body.tls = tls
    if (form.idleTimeout.trim()) body.idle_timeout = form.idleTimeout.trim()
    if (form.keepAlivePeriod.trim()) body.keep_alive_period = form.keepAlivePeriod.trim()
    if (form.initialPacketSize > 0) body.initial_packet_size = form.initialPacketSize
    if (form.disablePathMTU) body.disable_path_mtu_discovery = true
  }
  if (form.type === 'shadowsocks') {
    body.method = form.ssMethod || SS_DEFAULT_METHOD
    if (form.ssMethod !== 'none' && form.ssPassword.trim()) body.password = form.ssPassword.trim()
  }
  return body
}

const handleSave = async () => {
  validateMsg.value = validateForm()
  if (validateMsg.value) return

  saving.value = true
  try {
    if (isTuic.value || isShadowsocks.value) {
      await syncBoundUsers(
        isEditMode.value ? editingInboundTag.value : form.tag.trim(),
        selectedPoolUsers.value,
      )
    }
    const body = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildBody()
    const success = isEditMode.value
      ? await updateInbound(editingInboundTag.value, body)
      : await createInbound(body)

    if (success) {
      closeInboundModal()
    }
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>
