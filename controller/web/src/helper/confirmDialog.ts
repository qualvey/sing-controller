// 复用自 zashboard（MIT）：确认对话框队列（替代 ElMessageBox.confirm）
// 挂载见 components/common/ConfirmDialogHost.vue；用法：
//   const { confirmed, checked } = await showConfirmDialog({ message: '确定删除？', checkboxText: '强制删除' })
import { readonly, ref } from 'vue'

export type ConfirmDialogOptions = {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmButtonClass?: string
  /** 勾选框文案；存在时显示「勾选后不再询问」类选项，结果通过 checked 返回 */
  checkboxText?: string
}

export type ConfirmDialogResult = {
  confirmed: boolean
  checked: boolean
}

type ConfirmDialogRequest = ConfirmDialogOptions & {
  resolve: (value: ConfirmDialogResult) => void
}

const activeConfirmDialog = ref<ConfirmDialogRequest>()
const confirmDialogQueue: ConfirmDialogRequest[] = []

const showNextConfirmDialog = () => {
  if (activeConfirmDialog.value || confirmDialogQueue.length === 0) return
  activeConfirmDialog.value = confirmDialogQueue.shift()
}

export const confirmDialogState = readonly(activeConfirmDialog)

export const showConfirmDialog = (options: ConfirmDialogOptions) => {
  return new Promise<ConfirmDialogResult>((resolve) => {
    confirmDialogQueue.push({
      ...options,
      resolve
    })
    showNextConfirmDialog()
  })
}

export const resolveConfirmDialog = (confirmed: boolean, checked = false) => {
  const currentConfirmDialog = activeConfirmDialog.value
  if (!currentConfirmDialog) return

  activeConfirmDialog.value = undefined
  currentConfirmDialog.resolve({ confirmed, checked })
  showNextConfirmDialog()
}
