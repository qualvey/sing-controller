<template>
  <span
    class="inline-block text-[10px] font-bold tracking-[0.07em] uppercase px-1.5 py-0.5 rounded-[4px] shrink-0 border select-none leading-none"
    :style="{
      color: tagColor,
      backgroundColor: `${tagColor}18`,
      borderColor: `${tagColor}30`,
    }"
  >
    {{ (protocol || '').toUpperCase() }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  protocol: string
}>()

const PROTOCOL_COLORS: Record<string, string> = {
  MIXED: '#4a9eff',
  TUIC: '#a78bfa',
  VLESS: '#34d399',
  VMESS: '#f59e0b',
  SS: '#fb923c',
  SHADOWSOCKS: '#fb923c',
  HYSTERIA2: '#ec4899',
  HYSTERIA: '#ec4899',
  TROJAN: '#06b6d4',
  HTTP: '#3b82f6',
  SOCKS: '#6366f1',
  DIRECT: '#94a3b8',
  BLOCK: '#ef4444',
  DNS: '#10b981',
  NAIVE: '#8b5cf6',
  WIREGUARD: '#f97316',
  TUN: '#14b8a6',
  REDIRECT: '#eab308',
  TPROXY: '#f43f5e',
}

const tagColor = computed(() => {
  const key = (props.protocol || '').toUpperCase()
  return PROTOCOL_COLORS[key] || '#818cf8'
})
</script>
