<template>
  <span
    class="inline-block text-[11px] font-semibold px-2 py-0.5 rounded-[5px] bg-[#e53535] text-white shrink-0 tracking-[0.01em] font-mono leading-tight select-none"
  >
    {{ port }}
  </span>
</template>

<script setup lang="ts">
defineProps<{
  port: number | string
}>()
</script>
