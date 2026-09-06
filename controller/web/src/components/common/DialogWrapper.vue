<template>
  <Teleport to="body">
    <Transition
      name="modal"
      :duration="350"
    >
      <!-- 注意：Teleport 内禁用 v-show（vue 3.5.41 下 Teleport+v-show 切换会崩 emitsOptions/nextSibling），改用 v-if（Transition 标准用法） -->

      <div
        v-if="isOpen"
        ref="backdropRef"
        class="modal modal-open"
        :class="wrapperClass"
        :style="wrapperStyle"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title ? 'dialog-title' : undefined"
        @keydown.escape="handleEscape"
      >
        <!-- 遮罩，点击关闭 -->
        <div
          class="modal-backdrop w-screen"
          aria-hidden="true"
          @click="handleBackdropClick"
        />

        <!-- 弹窗主体：内容区自行滚动，标题/按钮区固定 -->
        <div
          ref="modalBoxRef"
          class="modal-box bg-base-100 relative flex flex-col overflow-hidden p-0 outline-none max-md:max-h-[85dvh] max-md:min-h-[calc(var(--dialog-viewport-height,100dvh)*0.4)] md:max-h-[85vh]"
          :class="[
            boxClass,
            size === 'large' ? 'max-w-4xl w-11/12' : size === 'small' ? 'max-w-sm' : ''
          ]"
          tabindex="-1"
          @keydown.enter.self="enter"
        >
          <div
            v-if="title && isOpen"
            id="dialog-title"
            class="border-base-content/10 relative shrink-0 border-b px-4 py-2.5 text-base font-bold flex items-center justify-between"
          >
            <span>{{ title }}</span>
            <div class="flex items-center gap-2">
              <slot name="title-right" />
              <button
                v-if="closable !== false"
                type="button"
                class="btn btn-circle btn-ghost btn-xs"
                aria-label="close"
                @click="close"
              >
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
          </div>
          <div
            v-if="isOpen"
            class="min-h-0 flex-1"
            :class="[
              bodyClass || 'overflow-y-auto',
              noPadding
                ? 'p-0 max-md:pb-[env(safe-area-inset-bottom)]'
                : 'p-4 max-md:pb-[calc(1rem+env(safe-area-inset-bottom))]'
            ]"
          >
            <slot />
          </div>
          <div
            v-if="$slots.footer && isOpen"
            class="border-base-content/10 bg-base-100/90 relative shrink-0 border-t px-4 py-3"
          >
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
// 复用自 zashboard（MIT）：daisyUI modal 弹窗包装（替代 el-dialog）
// 注意：#app-content 由 App.vue 提供；blurIntensity（zashboard 毛玻璃设置）已剥离
import { useDialogOpenState } from '@/composables/dialog'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { ref, watch } from 'vue'

const isOpen = defineModel<boolean>({ default: false })
const props = withDefaults(
  defineProps<{
    noPadding?: boolean
    boxClass?: string
    bodyClass?: string
    wrapperClass?: string
    wrapperStyle?: string | Record<string, any>
    title?: string
    closable?: boolean
    size?: 'small' | 'medium' | 'large'
  }>(),
  {
    closable: true,
  },
)
const emits = defineEmits<{
  (e: 'enter'): void
}>()

const modalBoxRef = ref<HTMLDivElement | undefined>(undefined)

// 弹窗计数（嵌套弹窗/页面手势判断用）
useDialogOpenState(isOpen)

watch(isOpen, (val) => {
  if (val) {
    requestAnimationFrame(() => {
      modalBoxRef.value?.focus()
    })
  }
})

function close() {
  if (props.closable === false) return
  isOpen.value = false
}

function handleEscape() {
  if (props.closable !== false) {
    close()
  }
}

function handleBackdropClick() {
  if (props.closable !== false) {
    close()
  }
}

function enter() {
  emits('enter')
}
</script>

<style scoped>
/* 遮罩淡入淡出 */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.25s ease-out;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

/* 桌面：弹窗缩放进入 */
.modal-enter-active .modal-box,
.modal-leave-active .modal-box {
  transition: transform 0.35s cubic-bezier(0.32, 0.72, 0, 1);
}
.modal-enter-from .modal-box,
.modal-leave-to .modal-box {
  transform: scale(0.95);
}

/* 移动端：底部滑入 */
@media (width < 48rem) {
  .modal-enter-from .modal-box,
  .modal-leave-to .modal-box {
    transform: translateY(100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .modal-enter-active .modal-box,
  .modal-leave-active .modal-box {
    transition: none;
  }
  .modal-enter-from .modal-box,
  .modal-leave-to .modal-box {
    transform: none;
  }
}
</style>
