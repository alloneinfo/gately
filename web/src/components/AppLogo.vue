<template>
  <!-- 应用标志：圆角方块 + 拱门（Gate）造型 -->
  <svg :width="size" :height="size" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" aria-label="Gately" class="app-logo">
    <defs>
      <linearGradient :id="gid" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
        <stop offset="0" stop-color="#18a888" />
        <stop offset="1" stop-color="#0b5f4f" />
      </linearGradient>
    </defs>
    <rect width="32" height="32" rx="8" :fill="`url(#${gid})`" />
    <path d="M9 24V15a7 7 0 0 1 14 0v9" stroke="#fff" stroke-width="2.6" stroke-linecap="round" />
    <path d="M13.5 24v-8.5a2.5 2.5 0 0 1 5 0V24" stroke="#fff" stroke-opacity="0.55" stroke-width="2.2" stroke-linecap="round" />
    <circle cx="16" cy="7.2" r="1.6" fill="#9ff3dc" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 32 })
// 每个实例使用独立的渐变 ID，避免同页多个标志冲突
const gid = `gl-${Math.random().toString(36).slice(2, 8)}`
</script>

<style scoped>
.app-logo {
  flex-shrink: 0;
  display: block;
}
</style>
