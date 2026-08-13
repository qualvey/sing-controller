// 极简全局 toast（重构过渡期替代 ElMessage 的公共设施）
// 用法：showToast('保存成功', 'success')
type ToastType = 'success' | 'error' | 'info' | 'warning'

let container: HTMLDivElement | null = null

export function showToast(message: string, type: ToastType = 'info', duration = 2500): void {
  if (!container) {
    container = document.createElement('div')
    container.className = 'toast-container'
    document.body.appendChild(container)
  }
  const el = document.createElement('div')
  el.className = `toast-item toast-${type}`
  el.textContent = message
  container.appendChild(el)
  // 双 rAF 确保进入动画生效
  requestAnimationFrame(() => {
    requestAnimationFrame(() => el.classList.add('toast-in'))
  })
  setTimeout(() => {
    el.classList.remove('toast-in')
    el.classList.add('toast-out')
    setTimeout(() => el.remove(), 250)
  }, duration)
}
