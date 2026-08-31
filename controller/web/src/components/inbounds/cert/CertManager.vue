<template>
  <div class="flex flex-col gap-4">
    <!-- 全局 certificate 配置卡片 -->
    <div class="border-base-content/10 bg-base-100 rounded-lg border p-4 shadow-xs">
      <div class="text-base-content/90 mb-3.5 flex items-center font-semibold text-sm">
        certificate（全局证书配置）
      </div>

      <div
        class="grid max-w-[760px] gap-x-4 gap-y-4 text-sm"
        style="grid-template-columns: 180px minmax(0, 1fr)"
      >
        <label class="text-base-content/70 pt-2">store</label>
        <div class="flex items-center gap-2">
          <SelectField
            v-model="certForm.store"
            class="w-full"
            :options="['system', 'mozilla', 'chrome', 'none']"
          />
          <span class="text-base-content/50 text-xs shrink-0">system 为默认（省略时）</span>
        </div>

        <template
          v-for="key in (['certificate', 'certificate_path', 'certificate_directory_path'] as const)"
          :key="key"
        >
          <label class="text-base-content/70 pt-2">{{ key }}</label>
          <div class="flex flex-col gap-1.5">
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="t in certForm[key]"
                :key="t"
                class="badge badge-primary badge-outline gap-1"
              >
                {{ t }}
                <button
                  type="button"
                  class="cursor-pointer text-xs opacity-70 hover:opacity-100"
                  @click="removeChip(certForm[key], t)"
                >
                  ×
                </button>
              </span>
            </div>
            <input
              :value="certChip[key]"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="`${key}（回车或失焦添加，可多值）`"
              @input="certChip[key] = ($event.target as HTMLInputElement).value"
              @keydown.enter.prevent="addChip(certForm[key], certChip[key] ? { value: certChip[key] } : { value: '' })"
              @blur="certChip[key] && addChip(certForm[key], { value: certChip[key] })"
            />
          </div>
        </template>

        <span />
        <button
          type="button"
          class="btn btn-primary btn-sm w-fit"
          :disabled="savingCert"
          @click="handleSaveCertificate"
        >
          保存 certificate 配置
        </button>
      </div>
    </div>

    <!-- certificate_providers 列表卡片 -->
    <div class="border-base-content/10 bg-base-100 rounded-lg border p-4 shadow-xs">
      <div class="mb-3.5 flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span class="text-base-content/90 font-semibold text-sm">certificate_providers</span>
          <span class="text-base-content/50 text-xs">
            （多态 ACME Provider，被入站 tls.certificate_provider 引用）
          </span>
        </div>
        <button
          type="button"
          class="btn btn-primary btn-sm gap-1.5"
          @click="openCreateProvider"
        >
          <PlusIcon class="h-4 w-4" />
          新建 Provider
        </button>
      </div>

      <div class="border-base-content/10 overflow-hidden rounded-lg border">
        <div
          v-if="loading && !providers.length"
          class="text-base-content/50 p-10 text-center text-sm"
        >
          加载证书 Provider 中…
        </div>

        <table
          v-else
          class="table table-sm w-full"
        >
          <thead>
            <tr class="border-base-content/10">
              <th class="w-[140px]">tag</th>
              <th class="w-24">type</th>
              <th>域名 (domain)</th>
              <th class="w-[200px]">email</th>
              <th class="w-[130px] text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in providers"
              :key="row.id"
              class="hover:bg-base-200/40 border-base-content/5 transition-colors"
            >
              <td>
                <span class="font-medium text-xs">{{ row.provider.tag || '—' }}</span>
              </td>
              <td>
                <span class="badge badge-sm badge-neutral badge-outline font-mono text-[11px]">
                  {{ row.provider.type || 'acme' }}
                </span>
              </td>
              <td>
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="d in toList(row.provider.domain)"
                    :key="d"
                    class="badge badge-sm badge-primary badge-outline"
                  >
                    {{ d }}
                  </span>
                  <span
                    v-if="!toList(row.provider.domain).length"
                    class="text-base-content/40 text-xs"
                  >
                    —
                  </span>
                </div>
              </td>
              <td>
                <span class="text-base-content/80 text-xs">{{ row.provider.email || '—' }}</span>
              </td>
              <td class="text-right">
                <button
                  class="btn btn-ghost btn-xs text-primary"
                  @click="openEditProvider(row)"
                >
                  编辑
                </button>
                <button
                  class="btn btn-ghost btn-xs text-error"
                  @click="handleRemoveProvider(row)"
                >
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div
          v-if="!loading && !providers.length"
          class="text-base-content/60 py-8 text-center text-sm"
        >
          暂无证书 Provider。点击右上角「新建 Provider」添加（如 Let's Encrypt / Cloudflare 等）。
        </div>
      </div>
    </div>

    <!-- Provider 编辑弹窗 -->
    <CertProviderDialog
      v-model="dialogVisible"
      :is-edit="isEdit"
      :provider-data="selectedProvider"
      @saved="load"
    />
  </div>
</template>

<script setup lang="ts">
import { api } from '@/api'
import CertProviderDialog from '@/components/inbounds/cert/CertProviderDialog.vue'
import { SelectField } from '@/components/ui/select'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { PlusIcon } from '@heroicons/vue/24/outline'
import { onMounted, reactive, ref } from 'vue'

const statusStore = useStatusStore()

const loading = ref(false)
const savingCert = ref(false)
const providers = ref<Array<{ id: string; provider: Record<string, any> }>>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const selectedProvider = ref<{ id: string; provider: Record<string, any> } | null>(null)

const certForm = reactive({
  store: 'system',
  certificate: [] as string[],
  certificate_path: [] as string[],
  certificate_directory_path: [] as string[],
})

const certChip = reactive({
  certificate: '',
  certificate_path: '',
  certificate_directory_path: '',
})

const toList = (v: unknown): string[] => {
  if (v == null) return []
  return Array.isArray(v) ? v.map(String) : [String(v)]
}

const addChip = (list: string[], input: { value: string }) => {
  const parts = input.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!list.includes(p)) list.push(p)
  }
  input.value = ''
}

const removeChip = (list: string[], t: string) => {
  const i = list.indexOf(t)
  if (i >= 0) list.splice(i, 1)
}

const load = async () => {
  loading.value = true
  try {
    const data = await api.certificate()
    const cert = data.certificate as Record<string, any> | null
    if (cert && typeof cert === 'object') {
      certForm.store = String(cert.store || 'system')
      certForm.certificate = toList(cert.certificate)
      certForm.certificate_path = toList(cert.certificate_path)
      certForm.certificate_directory_path = toList(cert.certificate_directory_path)
    } else {
      certForm.store = 'system'
      certForm.certificate = []
      certForm.certificate_path = []
      certForm.certificate_directory_path = []
    }
    providers.value = data.providers || []
  } catch (e) {
    showToast((e as Error).message || '加载证书配置失败', 'error')
  } finally {
    loading.value = false
  }
}

const handleSaveCertificate = async () => {
  savingCert.value = true
  try {
    const cert: Record<string, any> = {}
    if (certForm.store && certForm.store !== 'system') cert.store = certForm.store
    if (certForm.certificate.length) cert.certificate = [...certForm.certificate]
    if (certForm.certificate_path.length) cert.certificate_path = [...certForm.certificate_path]
    if (certForm.certificate_directory_path.length)
      cert.certificate_directory_path = [...certForm.certificate_directory_path]
    await api.saveCertificate(Object.keys(cert).length ? cert : null)
    showToast('certificate 配置已保存', 'success')
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    savingCert.value = false
  }
}

const openCreateProvider = () => {
  isEdit.value = false
  selectedProvider.value = null
  dialogVisible.value = true
}

const openEditProvider = (row: { id: string; provider: Record<string, any> }) => {
  isEdit.value = true
  selectedProvider.value = { ...row }
  dialogVisible.value = true
}

const handleRemoveProvider = async (row: { id: string; provider: Record<string, any> }) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 certificate provider「${row.provider.tag || row.id}」？`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!first.confirmed) return
  try {
    await api.deleteCertProvider(row.id)
    showToast('删除成功', 'success')
    await load()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该 provider 被引用：${err.references.join('、')}\n删除后将自动清除引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error',
      })
      if (!second.confirmed) return
      try {
        await api.deleteCertProvider(row.id, true)
        showToast('删除成功（已清除引用）', 'success')
        await load()
        await statusStore.refresh()
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

onMounted(load)
</script>
