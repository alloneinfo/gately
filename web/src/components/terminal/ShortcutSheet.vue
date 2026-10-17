<template>
  <!-- 移动端的快捷命令底部面板：点击即发送 -->
  <el-drawer :model-value="modelValue" direction="btt" size="auto" :with-header="false" class="sheet" @update:model-value="(v: boolean) => $emit('update:modelValue', v)">
    <div class="sheet-head">
      <span>{{ t('terminal.sheet.title') }}</span>
      <el-button text :icon="Settings2" @click="$emit('manage')">{{ t('terminal.sheet.manage') }}</el-button>
    </div>
    <div class="list">
      <button v-for="(s, i) in shortcuts" :key="i" class="item" @click="$emit('send', s)">
        <span class="name">{{ s.name }}</span>
        <span class="cmd mono">{{ s.command }}</span>
        <CornerDownLeft v-if="s.sendEnter" :size="14" class="enter" />
      </button>
      <el-empty v-if="!shortcuts.length" :description="t('terminal.sheet.empty')" :image-size="50" />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { CornerDownLeft, Settings2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import type { Shortcut } from './types'

const { t } = useI18n()

defineProps<{ modelValue: boolean; shortcuts: Shortcut[] }>()
defineEmits<{ 'update:modelValue': [v: boolean]; send: [s: Shortcut]; manage: [] }>()
</script>

<style scoped>
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  padding-bottom: 6px;
}
.list {
  max-height: 55vh;
  overflow: auto;
  padding-bottom: env(safe-area-inset-bottom);
}
.item {
  width: 100%;
  min-height: 48px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  border: none;
  border-bottom: 1px solid var(--g-border);
  background: transparent;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
}
.item:active {
  background: var(--el-fill-color-light);
}
.name {
  flex-shrink: 0;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}
.cmd {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.enter {
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
</style>
