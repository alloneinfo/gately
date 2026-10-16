<template>
  <div class="tabs" role="tablist">
    <div class="tab-scroll" ref="scroller">
      <div
        v-for="t in tabs"
        :key="t.id"
        class="tab"
        :class="{ active: t.id === active, insplit: splitMode && splitIds.includes(t.id) }"
        role="tab"
        :title="tabTip(t)"
        @click="$emit('select', t.id)"
        @dblclick="!isMobile && $emit('command', t.id, 'rename')"
      >
        <span class="dot" :class="t.status" />
        <span class="tab-title">{{ t.title }}</span>
        <el-dropdown trigger="click" @command="(c: string) => $emit('command', t.id, c)">
          <button class="tab-btn" aria-label="标签菜单" @click.stop><ChevronDown :size="14" /></button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="rename" :icon="Pencil">重命名标签</el-dropdown-item>
              <el-dropdown-item command="remark" :icon="StickyNote">会话备注</el-dropdown-item>
              <el-dropdown-item command="duplicate" :icon="Copy">复制为新会话</el-dropdown-item>
              <el-dropdown-item v-if="!isMobile && !splitIds.includes(t.id) && splitMode" command="join" :icon="Columns2">加入分屏</el-dropdown-item>
              <el-dropdown-item v-if="!isMobile && splitMode && splitIds.includes(t.id) && splitIds.length > 1" command="leave" :icon="Columns2">移出分屏</el-dropdown-item>
              <el-dropdown-item command="reconnect" :icon="RefreshCw">重新连接</el-dropdown-item>
              <el-dropdown-item command="kill" :icon="Power" divided>结束会话并关闭</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <button class="tab-btn close" aria-label="关闭标签" @click.stop="$emit('close', t.id)"><X :size="14" /></button>
      </div>
    </div>
    <el-dropdown v-if="!isMobile" trigger="click" @command="(k: string) => $emit('add', k)">
      <button class="add-btn" aria-label="新建会话" title="新建会话"><Plus :size="16" /></button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-for="c in conns" :key="c.key" :command="c.key" :disabled="!c.enable">{{ c.name }}</el-dropdown-item>
          <el-dropdown-item v-if="!conns.length" disabled>暂无连接</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <button v-else class="add-btn" aria-label="新建会话" @click="$emit('add-list')"><Plus :size="18" /></button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { ChevronDown, Columns2, Copy, Pencil, Plus, Power, RefreshCw, StickyNote, X } from 'lucide-vue-next'
import type { Conn, TermTab } from './types'

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

function tabTip(t: TermTab): string {
  return t.remark ? `${t.title}\n备注：${t.remark}` : t.title
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
