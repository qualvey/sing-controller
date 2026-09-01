<template>
  <DialogWrapper
    v-model="isUserModalOpen"
    :title="isEditMode ? '编辑用户' : '新建用户'"
    box-class="max-w-[560px]"
    @update:model-value="(val) => (!val ? closeUserModal() : null)"
  >
    <form
      class="grid gap-x-4 gap-y-5"
      style="grid-template-columns: 120px minmax(0, 1fr)"
      @submit.prevent="onSubmit"
    >
      <FormItem class="contents">
        <FormLabel class="pt-2">
          name <span class="text-destructive">*</span>
        </FormLabel>
        <FormControl>
          <Input
            v-model="name"
            v-bind="nameAttrs"
            :disabled="isEditMode"
            placeholder="用户标识，唯一"
            :aria-invalid="!!errors.name"
          />
          <FormMessage>{{ errors.name }}</FormMessage>
        </FormControl>
      </FormItem>

      <FormItem class="contents">
        <FormLabel class="pt-2">uuid</FormLabel>
        <FormControl>
          <div class="flex w-full gap-2">
            <Input
              v-model="uuid"
              v-bind="uuidAttrs"
              class="font-mono text-xs"
              placeholder="UUID（vless/vmess/tuic 用）"
              :aria-invalid="!!errors.uuid"
            />
            <button
              type="button"
              class="btn btn-ghost btn-sm shrink-0"
              @click="handleGenUuid"
            >
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
            <Input
              v-model="password"
              v-bind="passwordAttrs"
              :type="showPassword ? 'text' : 'password'"
              class="font-mono text-xs"
              placeholder="密码（trojan/tuic/hysteria2 用）"
              :aria-invalid="!!errors.password"
            />
            <button
              type="button"
              class="btn btn-ghost btn-sm shrink-0"
              :title="showPassword ? '隐藏密码' : '显示密码'"
              @click="showPassword = !showPassword"
            >
              <EyeSlashIcon
                v-if="showPassword"
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
              @click="handleGenPassword(true)"
            >
              <KeyIcon
                class="h-4 w-4"
                :class="{ 'animate-spin': generating }"
              />
              生成
            </button>
          </div>
          <FormMessage>{{ errors.password }}</FormMessage>
        </FormControl>
      </FormItem>

      <FormItem class="contents">
        <FormLabel class="pt-2">flow</FormLabel>
        <FormControl>
          <SelectField
            v-model="flow"
            v-bind="flowAttrs"
            :options="[
              { value: 'xtls-rprx-vision', label: 'xtls-rprx-vision' },
              { value: 'xtls-rprx-vision-udp443', label: 'xtls-rprx-vision-udp443' },
              { value: 'none', label: '（无）' },
            ]"
            placeholder="vless flow（xtls-rprx-vision 等）"
          />
        </FormControl>
      </FormItem>

      <FormItem class="contents">
        <FormLabel class="pt-2">绑定入站</FormLabel>
        <FormControl>
          <div class="flex flex-col gap-1.5">
            <label
              v-for="i in bindableInbounds"
              :key="i.tag"
              class="hover:bg-base-200/50 flex cursor-pointer items-center gap-2 rounded px-1.5 py-1 text-sm transition-colors"
            >
              <Checkbox
                :checked="(boundInbounds || []).includes(i.tag)"
                @update:checked="toggleInbound(i.tag)"
              />
              <span class="font-medium">{{ i.tag }}</span>
              <span class="text-base-content/50 text-xs">({{ i.type }})</span>
            </label>
            <p
              v-if="!bindableInbounds.length"
              class="text-base-content/50 text-xs"
            >
              暂无支持 users 的入站
            </p>
            <p class="text-base-content/60 text-xs leading-relaxed">
              保存后自动把该用户注入所选入站的 users[]（按入站类型取用 uuid/password/flow 字段）
            </p>
          </div>
        </FormControl>
      </FormItem>

      <div
        class="mt-2 flex justify-end gap-2"
        style="grid-column: 2"
      >
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          @click="closeUserModal"
        >
          取消
        </button>
        <button
          type="submit"
          class="btn btn-primary btn-sm"
          :disabled="saving"
        >
          保存
        </button>
      </div>
    </form>
  </DialogWrapper>
</template>

<script setup lang="ts">
import type { UserMeta } from '@/api'
import {
  bindableInbounds,
  generatePassword,
  generateUuid,
  saveUser,
} from '@/assembly/users'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { Checkbox } from '@/components/ui/checkbox'
import {
  FormControl,
  FormItem,
  FormLabel,
  FormMessage,
  toTypedSchema,
  useForm,
  z,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { SelectField } from '@/components/ui/select'
import {
  closeUserModal,
  editingUser,
  isEditMode,
  isUserModalOpen,
  onUserSavedCallback,
} from '@/composables/userManagement'
import { EyeIcon, EyeSlashIcon, KeyIcon } from '@heroicons/vue/24/outline'
import { ref, watch } from 'vue'

const saving = ref(false)
const generating = ref(false)
const showPassword = ref(false)

// UUID v4 格式校验
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

const schema = toTypedSchema(
  z
    .object({
      name: z.string().min(1, 'name 必填'),
      uuid: z
        .string()
        .optional()
        .refine((v) => !v || UUID_RE.test(v), {
          message: 'uuid 格式无效（应为标准 UUID v4）',
        }),
      password: z.string().optional(),
      flow: z.string().optional(),
      bound_inbounds: z.array(z.string()).default([]),
    })
    .superRefine((data, ctx) => {
      if (!data.uuid && !data.password) {
        ctx.addIssue({
          code: 'custom',
          path: ['uuid'],
          message: 'uuid 和 password 至少填一个',
        })
      }
    }),
)

const { handleSubmit, errors, defineField, setValues, resetForm } = useForm({
  validationSchema: schema,
  initialValues: {
    name: '',
    uuid: '',
    password: '',
    flow: 'xtls-rprx-vision',
    bound_inbounds: [] as string[],
  },
})

const [name, nameAttrs] = defineField('name')
const [uuid, uuidAttrs] = defineField('uuid')
const [password, passwordAttrs] = defineField('password')
const [flow, flowAttrs] = defineField('flow')
const [boundInbounds] = defineField('bound_inbounds')

watch(isUserModalOpen, async (isOpen) => {
  showPassword.value = false
  if (!isOpen) return

  if (isEditMode.value && editingUser.value) {
    const u = editingUser.value
    setValues({
      name: u.name,
      uuid: u.uuid || '',
      password: u.password || '',
      flow: u.flow || 'none',
      bound_inbounds: u.bound_inbounds || [],
    })
  } else {
    resetForm()
    // 新建默认自动生成密码
    await handleGenPassword(false)
    await handleGenUuid()
  }
})

const handleGenUuid = () => {
  uuid.value = generateUuid()
}

const handleGenPassword = async (notify = false) => {
  generating.value = true
  try {
    password.value = await generatePassword(notify)
  } finally {
    generating.value = false
  }
}

const toggleInbound = (tag: string) => {
  const cur = boundInbounds.value || []
  boundInbounds.value = cur.includes(tag) ? cur.filter((t) => t !== tag) : [...cur, tag]
}

const onSubmit = handleSubmit(async (values) => {
  saving.value = true
  try {
    const payload: UserMeta = {
      name: values.name.trim(),
      uuid: values.uuid?.trim() || '',
      password: values.password?.trim() || '',
      flow: values.flow === 'none' ? '' : values.flow?.trim() || '',
      bound_inbounds: values.bound_inbounds,
    }
    const success = await saveUser(payload, isEditMode.value, editingUser.value?.name)
    if (success) {
      if (onUserSavedCallback.value) {
        onUserSavedCallback.value(payload)
      }
      closeUserModal()
    }
  } finally {
    saving.value = false
  }
})
</script>

