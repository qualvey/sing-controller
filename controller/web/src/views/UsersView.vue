<script setup lang="ts">
// Users 页：shadcn-vue 表单样板（vee-validate + zod v4 校验）
import { onMounted, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { KeyIcon, PlusIcon, ArrowPathIcon, UserIcon } from '@heroicons/vue/24/outline'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { useForm } from '@/components/ui/form'
import { toTypedSchema, z } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { Checkbox } from '@/components/ui/checkbox'
import { SelectField } from '@/components/ui/select'
import { api, type Inbound, type UserMeta } from '../api'

const loading = ref(false)
const users = ref<UserMeta[]>([])
const inbounds = ref<Inbound[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const generating = ref(false)
const isEdit = ref(false)
const editingName = ref('')

// 支持 users 的入站类型（与后端一致）
const USER_TYPES = ['vless', 'vmess', 'trojan', 'tuic', 'hysteria2', 'hysteria', 'shadowsocks', 'anytls', 'shadowtls']
const bindableInbounds = ref<Inbound[]>([])

// UUID v4 格式校验
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

// zod schema：name 必填；uuid 填则必须 v4；uuid/password 至少一个
const schema = toTypedSchema(
  z
    .object({
      name: z.string().min(1, 'name 必填'),
      uuid: z.string().optional().refine((v) => !v || UUID_RE.test(v), { message: 'uuid 格式无效（应为标准 UUID v4）' }),
      password: z.string().optional(),
      flow: z.string().optional(),
      bound_inbounds: z.array(z.string()).default([])
    })
    .superRefine((data, ctx) => {
      if (!data.uuid && !data.password) {
        ctx.addIssue({ code: 'custom', path: ['uuid'], message: 'uuid 和 password 至少填一个' })
      }
    })
)

const { handleSubmit, errors, defineField, setValues, resetForm } = useForm({
  validationSchema: schema,
  initialValues: { name: '', uuid: '', password: '', flow: 'none', bound_inbounds: [] as string[] }
})

const [name, nameAttrs] = defineField('name')
const [uuid, uuidAttrs] = defineField('uuid')
const [password, passwordAttrs] = defineField('password')
const [flow, flowAttrs] = defineField('flow')
const [boundInbounds] = defineField('bound_inbounds')

const load = async () => {
  loading.value = true
  try {
    const [u, i] = await Promise.all([api.users(), api.inbounds()])
    users.value = u
    inbounds.value = i
    bindableInbounds.value = i.filter((x) => USER_TYPES.includes(x.type))
  } catch (e) {
    showToast((e as Error).message || '加载失败', 'error')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  editingName.value = ''
  resetForm()
  dialogVisible.value = true
  // 新建默认自动生成密码（与 shadowsocks 入站同格式：16 字节 → base64）
  void genPassword()
}

const openEdit = (u: UserMeta) => {
  isEdit.value = true
  editingName.value = u.name
  setValues({
    name: u.name,
    uuid: u.uuid || '',
    password: u.password || '',
    flow: u.flow || 'none',
    bound_inbounds: u.bound_inbounds || []
  })
  dialogVisible.value = true
}

// crypto.randomUUID 需要安全上下文（https/localhost）——局域网 http 下不可用，
// 用 getRandomValues（无此限制）实现 UUID v4
const genUuid = () => {
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
  uuid.value = hex.slice(0, 8) + '-' + hex.slice(8, 12) + '-' + hex.slice(12, 16) + '-' + hex.slice(16, 20) + '-' + hex.slice(20)
}

// 生成随机密码：优先后端（POST /api/tools/password），失败浏览器端兜底（格式一致）
const genPassword = async (notify = false) => {
  generating.value = true
  try {
    password.value = await api.genPassword()
    if (notify) showToast('已生成随机密码', 'success')
  } catch {
    const bytes = crypto.getRandomValues(new Uint8Array(16))
    password.value = btoa(String.fromCharCode(...bytes))
    if (notify) showToast('后端生成失败，已用浏览器随机数生成', 'warning')
  } finally {
    generating.value = false
  }
}

const toggleInbound = (tag: string) => {
  const cur = boundInbounds.value || []
  boundInbounds.value = cur.includes(tag) ? cur.filter((t) => t !== tag) : [...cur, tag]
}

// vee-validate 校验通过后回调
const onSubmit = handleSubmit(async (values) => {
  saving.value = true
  try {
    const payload: UserMeta = {
      name: values.name.trim(),
      uuid: values.uuid?.trim() || '',
      password: values.password?.trim() || '',
      flow: values.flow === 'none' ? '' : values.flow?.trim() || '',
      bound_inbounds: values.bound_inbounds
    }
    if (isEdit.value) {
      await api.updateUser(editingName.value, payload)
    } else {
      await api.createUser(payload)
    }
    showToast('已保存（绑定入站的 users 已同步）', 'success')
    dialogVisible.value = false
    await load()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
})

const remove = async (u: UserMeta) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除用户',
    message: `删除用户 ${u.name}？将从其绑定的所有入站移除`,
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return
  try {
    await api.deleteUser(u.name)
    showToast('已删除', 'success')
    await load()
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[#e4e7ed] bg-white px-4 py-3 dark:border-[#303030] dark:bg-[#1d1e1f]">
      <button class="btn btn-primary btn-sm" @click="openCreate">
        <PlusIcon class="h-4 w-4" />
        新建用户
      </button>
      <button class="btn btn-ghost btn-sm" :disabled="loading" @click="load">
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        刷新
      </button>
      <span class="ml-auto text-xs text-[#606266] dark:text-[#a6b0bf]">
        {{ users.length }} 个用户 · 绑定入站：{{ bindableInbounds.length }} 个（vless/vmess/trojan/tuic/hysteria2 等）
      </span>
    </div>

    <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
      <div v-if="loading && !users.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
      <table v-else class="table table-sm w-full">
        <thead>
          <tr>
            <th class="w-[140px]">name</th>
            <th class="w-[280px]">uuid</th>
            <th class="w-[120px]">password</th>
            <th class="w-[100px]">flow</th>
            <th>绑定入站</th>
            <th class="w-[130px] text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.name">
            <td>
              <span class="flex items-center gap-1.5 font-medium">
                <UserIcon class="h-3.5 w-3.5 text-[#909399]" />{{ u.name }}
              </span>
            </td>
            <td><span class="font-mono text-xs">{{ u.uuid || '—' }}</span></td>
            <td><span class="font-mono text-xs">{{ u.password ? '••••••' : '—' }}</span></td>
            <td><span class="text-xs">{{ u.flow || '—' }}</span></td>
            <td>
              <div class="flex flex-wrap gap-1">
                <span v-for="t in u.bound_inbounds || []" :key="t" class="badge badge-sm badge-primary badge-outline">{{ t }}</span>
                <span v-if="!(u.bound_inbounds || []).length" class="text-xs text-[#909399]">未绑定</span>
              </div>
            </td>
            <td class="text-right">
              <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(u)">编辑</button>
              <button class="btn btn-ghost btn-xs text-error" @click="remove(u)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!loading && !users.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">
        暂无用户。新建用户后可绑定到多个入站（vless/tuic 等），自动同步到各入站的 users[]
      </div>
    </div>

    <!-- 新建/编辑弹窗（DialogWrapper + vee-validate 表单） -->
    <DialogWrapper v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新建用户'" box-class="max-w-[560px]"> 
      <form class="grid gap-x-4 gap-y-5" style="grid-template-columns: 120px minmax(0, 1fr)" @submit.prevent="onSubmit">
        <FormItem class="contents">
          <FormLabel class="pt-2">name <span class="text-destructive">*</span></FormLabel>
          <FormControl>
            <Input v-model="name" v-bind="nameAttrs" :disabled="isEdit" placeholder="用户标识，唯一" :aria-invalid="!!errors.name" />
            <FormMessage>{{ errors.name }}</FormMessage>
          </FormControl>
        </FormItem>

        <FormItem class="contents">
          <FormLabel class="pt-2">uuid</FormLabel>
          <FormControl>
            <div class="flex w-full gap-2">
              <Input v-model="uuid" v-bind="uuidAttrs" class="font-mono" placeholder="UUID（vless/vmess/tuic 用）" :aria-invalid="!!errors.uuid" />
              <button type="button" class="btn btn-ghost btn-sm shrink-0" @click="genUuid">
                <KeyIcon class="h-4 w-4" />
                生成
              </button>
            </div>
            <FormMessage>{{ errors.uuid }}</FormMessage>
          </FormControl>
        </FormItem>

        <FormItem class="contents">
          <FormLabel class="pt-2">password</FormLabel>
          <FormControl>
            <div class="flex w-full gap-2">
              <Input v-model="password" v-bind="passwordAttrs" type="password" placeholder="密码（trojan/tuic/hysteria2 用）" :aria-invalid="!!errors.password" />
              <button type="button" class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genPassword(true)">
                <KeyIcon class="h-4 w-4" />
                生成
              </button>
            </div>
            <FormMessage>{{ errors.password }}</FormMessage>
          </FormControl>
        </FormItem>

        <FormItem class="contents">
          <FormLabel class="pt-2">flow</FormLabel>
          <FormControl>
            <SelectField v-model="flow" v-bind="flowAttrs" :options="[{ value: 'none', label: '（无）' }, { value: 'xtls-rprx-vision', label: 'xtls-rprx-vision' }, { value: 'xtls-rprx-vision-udp443', label: 'xtls-rprx-vision-udp443' }]" placeholder="vless flow（xtls-rprx-vision 等）" />
          </FormControl>
        </FormItem>

        <FormItem class="contents">
          <FormLabel class="pt-2">绑定入站</FormLabel>
          <FormControl>
            <div class="flex flex-col gap-1.5">
              <label v-for="i in bindableInbounds" :key="i.tag" class="flex cursor-pointer items-center gap-2 text-sm">
                <Checkbox :checked="(boundInbounds || []).includes(i.tag)" @update:checked="toggleInbound(i.tag)" />
                {{ i.tag }} <span class="text-xs text-[#909399]">({{ i.type }})</span>
              </label>
              <p v-if="!bindableInbounds.length" class="text-xs text-[#909399]">暂无支持 users 的入站</p>
              <p class="text-xs leading-relaxed text-[#606266] dark:text-[#a6b0bf]">
                保存后自动把该用户注入所选入站的 users[]（按入站类型取用 uuid/password/flow 字段）
              </p>
            </div>
          </FormControl>
        </FormItem>

        <div class="flex justify-end gap-2" style="grid-column: 2">
          <button type="button" class="btn btn-ghost btn-sm" @click="dialogVisible = false">取消</button>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">保存</button>
        </div>
      </form>
    </DialogWrapper>
  </div>
</template>
