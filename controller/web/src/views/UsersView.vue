<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Key, User } from '@element-plus/icons-vue'
import { api, type Inbound, type UserMeta } from '../api'

const loading = ref(false)
const users = ref<UserMeta[]>([])
const inbounds = ref<Inbound[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const isEdit = ref(false)
const editingName = ref('')

// 支持 users 的入站类型（与后端一致）
const USER_TYPES = ['vless', 'vmess', 'trojan', 'tuic', 'hysteria2', 'hysteria', 'shadowsocks', 'anytls', 'shadowtls']
const bindableInbounds = ref<Inbound[]>([])

const form = reactive({
  name: '',
  uuid: '',
  password: '',
  flow: '',
  bound_inbounds: [] as string[]
})

const load = async () => {
  loading.value = true
  try {
    const [u, i] = await Promise.all([api.users(), api.inbounds()])
    users.value = u
    inbounds.value = i
    bindableInbounds.value = i.filter((x) => USER_TYPES.includes(x.type))
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.name = ''
  form.uuid = ''
  form.password = ''
  form.flow = ''
  form.bound_inbounds = []
}

const openCreate = () => {
  isEdit.value = false
  editingName.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (u: UserMeta) => {
  isEdit.value = true
  editingName.value = u.name
  form.name = u.name
  form.uuid = u.uuid || ''
  form.password = u.password || ''
  form.flow = u.flow || ''
  form.bound_inbounds = u.bound_inbounds || []
  dialogVisible.value = true
}

const genUuid = () => {
  form.uuid = crypto.randomUUID()
}

const save = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('name 必填')
    return
  }
  if (!form.uuid.trim() && !form.password.trim()) {
    ElMessage.warning('uuid 和 password 至少填一个')
    return
  }
  saving.value = true
  try {
    const payload: UserMeta = {
      name: form.name.trim(),
      uuid: form.uuid.trim(),
      password: form.password.trim(),
      flow: form.flow.trim(),
      bound_inbounds: form.bound_inbounds
    }
    if (isEdit.value) {
      await api.updateUser(editingName.value, payload)
    } else {
      await api.createUser(payload)
    }
    ElMessage.success('已保存（绑定入站的 users 已同步）')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const remove = async (u: UserMeta) => {
  try {
    await ElMessageBox.confirm(`删除用户 ${u.name}？将从其绑定的所有入站移除`, '删除用户', {
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.deleteUser(u.name)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-button type="primary" :icon="Plus" @click="openCreate">新建用户</el-button>
      <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
      <span class="ml-auto text-xs text-[var(--el-text-color-secondary)]">
        {{ users.length }} 个用户 · 绑定入站：{{ bindableInbounds.length }} 个（vless/vmess/trojan/tuic/hysteria2 等）
      </span>
    </div>

    <div class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)]">
      <el-table :data="users" v-loading="loading" size="small" style="width: 100%">
        <el-table-column label="name" min-width="140">
          <template #default="{ row }">
            <span class="flex items-center gap-1.5 font-medium">
              <el-icon :size="14"><User /></el-icon>{{ row.name }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="uuid" min-width="280">
          <template #default="{ row }">
            <span class="font-mono text-xs">{{ row.uuid || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="password" min-width="120">
          <template #default="{ row }">
            <span class="font-mono text-xs">{{ row.password ? '••••••' : '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="flow" width="100">
          <template #default="{ row }">
            <span class="text-xs">{{ row.flow || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="绑定入站" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-1">
              <el-tag v-for="t in row.bound_inbounds || []" :key="t" size="small" type="primary" effect="plain">{{ t }}</el-tag>
              <span v-if="!(row.bound_inbounds || []).length" class="text-xs text-[var(--el-text-color-placeholder)]">未绑定</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="" width="130" align="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <div class="py-8 text-sm text-[var(--el-text-color-secondary)]">
            暂无用户。新建用户后可绑定到多个入站（vless/tuic 等），自动同步到各入站的 users[]
          </div>
        </template>
      </el-table>
    </div>

    <!-- 新建/编辑 dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新建用户'" width="560px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="name" required>
          <el-input v-model="form.name" :disabled="isEdit" placeholder="用户标识，唯一" />
        </el-form-item>
        <el-form-item label="uuid">
          <div class="flex w-full gap-2">
            <el-input v-model="form.uuid" placeholder="UUID（vless/vmess/tuic 用）" class="font-mono" />
            <el-button :icon="Key" @click="genUuid">生成</el-button>
          </div>
        </el-form-item>
        <el-form-item label="password">
          <el-input v-model="form.password" type="password" show-password placeholder="密码（trojan/tuic/hysteria2 用）" />
        </el-form-item>
        <el-form-item label="flow">
          <el-select v-model="form.flow" clearable style="width: 100%" placeholder="vless flow（xtls-rprx-vision 等）">
            <el-option label="xtls-rprx-vision" value="xtls-rprx-vision" />
            <el-option label="xtls-rprx-vision-udp443" value="xtls-rprx-vision-udp443" />
          </el-select>
        </el-form-item>
        <el-form-item label="绑定入站">
          <el-select v-model="form.bound_inbounds" multiple filterable style="width: 100%" placeholder="选择要绑定到的入站（可多选）">
            <el-option v-for="i in bindableInbounds" :key="i.tag" :label="`${i.tag} (${i.type})`" :value="i.tag" />
          </el-select>
          <div class="mt-1 w-full text-xs text-[var(--el-text-color-secondary)]">
            保存后自动把该用户注入所选入站的 users[]（按入站类型取用 uuid/password/flow 字段）
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
