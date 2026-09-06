<template>
  <span
    class="inline-flex items-center text-xs font-mono font-medium px-2.5 py-0.5 rounded-md bg-base-200/90 text-base-content/90 border border-base-content/15 shrink-0 tracking-tight leading-none select-none shadow-2xs"
  >
    {{ port }}
  </span>
</template>

<script setup lang="ts">
defineProps<{
  port: number | string
}>()
</script>

