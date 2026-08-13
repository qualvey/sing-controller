// 复用自 zashboard（MIT）简化版：弹窗打开计数 + DialogWrapper 状态管理
// 原版还耦合 useViewportHeight（移动端软键盘避让），本项目暂无该场景，已剥离。
import { onUnmounted, ref, watch, type Ref } from 'vue'

/** 当前可见弹窗数量（嵌套弹窗也正确计数） */
export const openDialogCount = ref(0)

export const useDialogOpenState = (isOpen: Ref<boolean | undefined>) => {
  let held = false

  const acquire = () => {
    if (held) return
    held = true
    openDialogCount.value++
  }

  const release = () => {
    if (!held) return
    held = false
    openDialogCount.value--
  }

  watch(isOpen, (val) => (val ? acquire() : release()), { immediate: true })
  // 弹窗可能被 v-if 卸载时仍处于打开态；归还计数，避免泄漏
  onUnmounted(release)
}
