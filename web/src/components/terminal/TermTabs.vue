<template>
  <div class="tabs" role="tablist">
    <div class="tab-scroll" ref="scroller">
      <div
        v-for="tb in tabs"
        :key="tb.id"
        class="tab"
        :class="{ active: tb.id === active, insplit: splitMode && splitIds.includes(tb.id) }"
        role="tab"
        :title="tabTip(tb)"
        @click="$emit('select', tb.id)"
        @dblclick="!isMobile && $emit('command', tb.id, 'rename')"
      >
        <span class="dot" :class="tb.status" />
        <span class="tab-title">{{ tb.title }}</span>
        <el-dropdown trigger="click" @command="(c: string) => $emit('command', tb.id, c)">
          <button class="tab-btn" :aria-label="t('terminal.tabs.menu')" @click.stop><ChevronDown :size="14" /></button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="rename" :icon="Pencil">{{ t('terminal.tabs.rename') }}</el-dropdown-item>
              <el-dropdown-item command="remark" :icon="StickyNote">{{ t('terminal.tabs.remark') }}</el-dropdown-item>
              <el-dropdown-item command="duplicate" :icon="Copy">{{ t('terminal.tabs.duplicate') }}</el-dropdown-item>
              <el-dropdown-item v-if="!isMobile && !splitIds.includes(tb.id) && splitMode" command="join" :icon="Columns2">{{ t('terminal.tabs.join') }}</el-dropdown-item>
              <el-dropdown-item v-if="!isMobile && splitMode && splitIds.includes(tb.id) && splitIds.length > 1" command="leave" :icon="Columns2">{{ t('terminal.tabs.leave') }}</el-dropdown-item>
              <el-dropdown-item command="reconnect" :icon="RefreshCw">{{ t('terminal.tabs.reconnect') }}</el-dropdown-item>
              <el-dropdown-item command="kill" :icon="Power" divided>{{ t('terminal.tabs.kill') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <button class="tab-btn close" :aria-label="t('terminal.tabs.close')" @click.stop="$emit('close', tb.id)"><X :size="14" /></button>
      </div>
    </div>
    <el-dropdown v-if="!isMobile" trigger="click" @command="(k: string) => $emit('add', k)">
      <button class="add-btn" :aria-label="t('terminal.tabs.newSession')" :title="t('terminal.tabs.newSession')"><Plus :size="16" /></button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-for="c in conns" :key="c.key" :command="c.key" :disabled="!c.enable">{{ c.name }}</el-dropdown-item>
          <el-dropdown-item v-if="!conns.length" disabled>{{ t('terminal.tabs.noConns') }}</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <button v-else class="add-btn" :aria-label="t('terminal.tabs.newSession')" @click="$emit('add-list')"><Plus :size="18" /></button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { ChevronDown, Columns2, Copy, Pencil, Plus, Power, RefreshCw, StickyNote, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import type { Conn, TermTab } from './types'

const { t } = useI18n()

const props = defineProps<{
  tabs: TermTab[]
  active: string
  conns: Conn[]
  splitMode: boolean
  splitIds: string[]
  isMobile: boolean
}>()
defineEmits<{
  select: [id: string]
  close: [id: string]
  command: [id: string, cmd: string]
  add: [connKey: string]
  'add-list': []
}>()

const scroller = ref<HTMLElement>()

function tabTip(tab: TermTab): string {
  return tab.remark ? `${tab.title}\n${t('terminal.tabs.remarkTip', { remark: tab.remark })}` : tab.title
}

// 切换标签时把活动标签滚动到可见区域
watch(
  () => props.active,
  async () => {
    await nextTick()
    scroller.value?.querySelector('.tab.active')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  },
)
</script>

<style scoped>
.tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 1;
}
.tab-scroll {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow-x: auto;
  min-width: 0;
  scrollbar-width: none;
}
.tab-scroll::-webkit-scrollbar {
  display: none;
}
.tab {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  max-width: 200px;
  padding: 0 4px 0 10px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-regular);
  cursor: pointer;
  user-select: none;
  -webkit-user-select: none;
}
.tab.active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
  color: var(--el-color-primary);
}
.tab.insplit:not(.active) {
  border-style: dashed;
  border-color: var(--el-color-primary-light-5);
}
.tab-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--el-color-info);
}
.dot.connected {
  background: var(--el-color-success);
}
.dot.connecting,
.dot.reconnecting {
  background: var(--el-color-warning);
}
.dot.closed,
.dot.ended {
  background: var(--el-color-danger);
}
.tab-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  opacity: 0.75;
}
.tab-btn:hover {
  background: var(--el-fill-color);
  opacity: 1;
}
.add-btn {
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: transparent;
  color: var(--el-text-color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.add-btn:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
}
@media (max-width: 767.98px) {
  .tab {
    height: 40px;
    max-width: 150px;
  }
  .tab-btn {
    width: 32px;
    height: 32px;
  }
  .tab:not(.active) .tab-btn.close {
    display: none;
  }
  .add-btn {
    width: 40px;
    height: 40px;
  }
}
</style>
