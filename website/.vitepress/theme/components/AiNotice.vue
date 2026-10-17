<script setup lang="ts">
import { computed } from 'vue'
import { useData } from 'vitepress'

// 非简体中文页面的译文由 AI 生成，在页面顶部给出提示并指向 GitHub 改进
const { theme, localeIndex } = useData()
const text = computed(() => (localeIndex.value === 'root' ? '' : theme.value.aiNotice))
</script>

<template>
  <a v-if="text" class="ai-notice" href="https://github.com/wangwanjie/gately/issues" target="_blank" rel="noreferrer">{{ text }}</a>
</template>

<style scoped>
.ai-notice {
  display: block;
  margin: 0 0 16px;
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  text-decoration: none;
}
.ai-notice:hover {
  color: var(--vp-c-brand-1);
  border-color: var(--vp-c-brand-1);
}
</style>
