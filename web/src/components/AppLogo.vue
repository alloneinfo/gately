<template>
  <!-- 应用标志：渐变圆角方块 + 由弧线与横杠构成的 G（Gate 入口），右上角为信号节点；配色随主题变化 -->
  <svg :width="size" :height="size" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" aria-label="Gately" class="app-logo">
    <defs>
      <linearGradient :id="gid" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
        <stop offset="0" style="stop-color: var(--g-logo-from, #18a888)" />
        <stop offset="1" style="stop-color: var(--g-logo-to, #0b5f4f)" />
      </linearGradient>
    </defs>
    <rect class="logo-bg" width="32" height="32" rx="9" :fill="`url(#${gid})`" />
    <path d="M21.7 10.3A8 8 0 1 0 24 16h-7" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
    <circle cx="23.6" cy="8.6" r="2.2" style="fill: var(--g-logo-dot, #9ff3dc)" />
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
/* 朋克主题使用锐角 */
:global(html[data-theme='punk']) .logo-bg {
  rx: 4px;
}
</style>
