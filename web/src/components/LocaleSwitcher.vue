<template>
  <el-dropdown trigger="click" @command="onCommand">
    <button class="picker-btn" :aria-label="t('common.language')" :title="t('common.language')">
      <Languages :size="18" />
    </button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item v-for="l in locales" :key="l.code" :command="l.code" :class="{ active: l.code === locale }">
          <Check v-if="l.code === locale" :size="14" class="check" />
          <span v-else class="check" />
          {{ l.name }}
        </el-dropdown-item>
        <!-- 译文由 AI 生成，非简体中文界面给出提示并链接到 GitHub -->
        <el-dropdown-item v-if="locale !== 'zh-CN'" command="__notice" divided class="notice">
          <a :href="ISSUES_URL" target="_blank" rel="noreferrer" @click.stop>{{ t('common.aiTranslated') }}</a>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { Check, Languages } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { setLocale } from '../i18n'
import { locales } from '../i18n/locales'

const { t, locale } = useI18n()

const ISSUES_URL = 'https://github.com/wangwanjie/gately/issues'

function onCommand(code: string) {
  if (code === '__notice') {
    return
  }
  setLocale(code)
}
</script>

<style scoped>
.picker-btn {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    color 0.15s,
    border-color 0.15s;
}
.picker-btn:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
}
.check {
  display: inline-block;
  width: 14px;
  margin-right: 6px;
}
.notice {
  max-width: 260px;
  white-space: normal;
  line-height: 1.5;
  font-size: 12px;
}
.notice a {
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.notice a:hover {
  color: var(--el-color-primary);
}
.active {
  color: var(--el-color-primary);
}
</style>
