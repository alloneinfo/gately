<template>
  <div class="toolbar">
    <template v-if="!isMobile">
      <el-tooltip content="文件管理（SFTP）" placement="bottom"><el-button class="tb" text :icon="FolderOpen" :disabled="!sftpEnabled" @click="$emit('action', 'sftp')" /></el-tooltip>
      <el-tooltip content="搜索" placement="bottom"><el-button class="tb" text :icon="Search" @click="$emit('action', 'search')" /></el-tooltip>
      <el-tooltip content="复制选中内容" placement="bottom"><el-button class="tb" text :icon="Copy" @click="$emit('action', 'copy')" /></el-tooltip>
      <el-tooltip content="粘贴" placement="bottom"><el-button class="tb" text :icon="ClipboardPaste" @click="$emit('action', 'paste')" /></el-tooltip>
      <span class="sep" />
      <el-tooltip content="减小字号" placement="bottom"><el-button class="tb" text :icon="Minus" @click="$emit('action', 'font-dec')" /></el-tooltip>
      <span class="font-size">{{ fontSize }}</span>
      <el-tooltip content="增大字号" placement="bottom"><el-button class="tb" text :icon="Plus" @click="$emit('action', 'font-inc')" /></el-tooltip>
      <el-dropdown trigger="click" @command="(n: string) => $emit('update:themeName', n)">
        <el-button class="tb" text :icon="Palette" title="终端配色" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-for="t in termThemes" :key="t.name" :command="t.name" :class="{ 'is-current': t.name === themeName }">
              <span class="swatch" :style="{ background: t.theme.background, borderColor: t.theme.foreground }" />{{ t.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <span class="sep" />
      <el-dropdown trigger="click" @command="(s: Shortcut) => $emit('shortcut', s)">
        <el-button class="tb wide" text :icon="Zap">快捷命令</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-for="(s, i) in shortcuts" :key="i" :command="s">
              {{ s.name }}<span class="sc-cmd mono">{{ s.command }}</span>
            </el-dropdown-item>
            <el-dropdown-item v-if="!shortcuts.length" disabled>暂无快捷命令</el-dropdown-item>
            <el-dropdown-item divided :icon="Settings2" @click="$emit('action', 'shortcuts-manage')">管理快捷命令</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown trigger="click" @command="(c: string) => $emit('action', c)">
        <el-button class="tb wide" text :icon="Columns2">{{ splitMode ? '分屏中' : '分屏' }}</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="split-h" :icon="Columns2">左右分屏</el-dropdown-item>
            <el-dropdown-item command="split-v" :icon="Rows2">上下分屏</el-dropdown-item>
            <el-dropdown-item command="split-grid" :icon="LayoutGrid">网格分屏</el-dropdown-item>
            <el-dropdown-item command="split-off" :icon="Square" :disabled="!splitMode" divided>退出分屏</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <span class="grow" />
      <el-tooltip content="清屏" placement="bottom"><el-button class="tb" text :icon="Eraser" @click="$emit('action', 'clear')" /></el-tooltip>
      <el-tooltip content="会话列表" placement="bottom"><el-button class="tb" text :icon="ListTree" @click="$emit('action', 'sessions')" /></el-tooltip>
      <el-tooltip :content="fullscreen ? '退出全屏' : '全屏'" placement="bottom">
        <el-button class="tb" text :icon="fullscreen ? Minimize2 : Maximize2" @click="$emit('action', 'fullscreen')" />
      </el-tooltip>
    </template>

    <template v-else>
      <el-button class="tb" text :icon="FolderOpen" :disabled="!sftpEnabled" aria-label="文件管理" @click="$emit('action', 'sftp')" />
      <el-button class="tb" text :icon="Search" aria-label="搜索" @click="$emit('action', 'search')" />
      <el-dropdown trigger="click" placement="bottom-end" @command="onMore">
        <el-button class="tb" text :icon="Ellipsis" aria-label="更多" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="copy" :icon="Copy">复制（选区或当前屏幕）</el-dropdown-item>
            <el-dropdown-item command="paste" :icon="ClipboardPaste">粘贴</el-dropdown-item>
            <el-dropdown-item command="font-inc" :icon="Plus">增大字号（{{ fontSize }}）</el-dropdown-item>
            <el-dropdown-item command="font-dec" :icon="Minus">减小字号</el-dropdown-item>
            <el-dropdown-item command="clear" :icon="Eraser">清屏</el-dropdown-item>
            <el-dropdown-item command="sessions" :icon="ListTree">会话列表</el-dropdown-item>
            <el-dropdown-item command="shortcuts-manage" :icon="Settings2">管理快捷命令</el-dropdown-item>
            <el-dropdown-item
              v-for="t in termThemes"
              :key="t.name"
              :command="'theme:' + t.name"
              :divided="t.name === termThemes[0].name"
              :class="{ 'is-current': t.name === themeName }"
            >
              <span class="swatch" :style="{ background: t.theme.background, borderColor: t.theme.foreground }" />配色：{{ t.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  ClipboardPaste, Columns2, Copy, Ellipsis, Eraser, FolderOpen, LayoutGrid, ListTree, Maximize2, Minimize2, Minus, Palette, Plus, Rows2, Search, Settings2, Square, Zap,
} from 'lucide-vue-next'
import { termThemes } from './themes'
import type { Shortcut } from './types'

defineProps<{
  isMobile: boolean
  sftpEnabled: boolean
  fontSize: number
  themeName: string
  shortcuts: Shortcut[]
  splitMode: boolean
  fullscreen: boolean
}>()
const emit = defineEmits<{
  action: [name: string]
  'update:themeName': [name: string]
  shortcut: [s: Shortcut]
}>()

function onMore(cmd: string) {
  if (cmd.startsWith('theme:')) {
    emit('update:themeName', cmd.slice(6))
    return
  }
  emit('action', cmd)
}
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.toolbar :deep(.el-button + .el-button) {
  margin-left: 0;
}
.tb {
  padding: 8px;
  min-width: 32px;
}
.tb.wide {
  padding: 8px 10px;
}
.sep {
  width: 1px;
  height: 18px;
  margin: 0 4px;
  background: var(--g-border);
}
.grow {
  flex: 1;
}
.font-size {
  min-width: 22px;
  text-align: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.swatch {
  display: inline-block;
  width: 14px;
  height: 14px;
  margin-right: 8px;
  border: 1px solid;
  border-radius: 4px;
  vertical-align: -2px;
}
.sc-cmd {
  margin-left: 12px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.is-current {
  color: var(--el-color-primary);
}
@media (max-width: 767.98px) {
  .tb {
    min-width: 40px;
    height: 40px;
    padding: 0;
  }
}
</style>
