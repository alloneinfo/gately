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

function onCommand(code: string) {
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
.active {
  color: var(--el-color-primary);
}
</style>
