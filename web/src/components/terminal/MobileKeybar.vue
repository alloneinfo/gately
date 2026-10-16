<template>
  <!-- 移动端虚拟按键栏：软键盘上没有的功能键与常用符号。
       按下时阻止默认行为，避免终端输入框失焦导致软键盘收起。 -->
  <div class="keybar" @contextmenu.prevent>
    <button
      v-for="k in row1"
      :key="k.label"
      class="kb"
      :class="{ on: k.sticky && (k.sticky === 'ctrl' ? ctrl : alt), repeat: k.repeat }"
      v-bind="bind(k)"
    >
      <component :is="k.icon" v-if="k.icon" :size="16" />
      <template v-else>{{ k.label }}</template>
    </button>
    <button v-for="k in row2" :key="k.label" class="kb" v-bind="bind(k)">
      <component :is="k.icon" v-if="k.icon" :size="16" />
      <template v-else>{{ k.label }}</template>
    </button>
    <button class="kb wide" @pointerdown.prevent="$emit('shortcuts')" @mousedown.prevent>
      <Zap :size="14" />快捷命令
    </button>
    <button class="kb" aria-label="呼出键盘" @pointerdown.prevent="$emit('focus')" @mousedown.prevent>
      <Keyboard :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount } from 'vue'
import type { Component } from 'vue'
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, Keyboard, Zap } from 'lucide-vue-next'
import type { KeyName } from './terminalKeys'

defineProps<{ ctrl: boolean; alt: boolean }>()
const emit = defineEmits<{
  key: [name: KeyName]
  text: [s: string]
  'toggle-ctrl': []
  'toggle-alt': []
  shortcuts: []
  focus: []
}>()

interface Key {
  label: string
  icon?: Component
  key?: KeyName
  text?: string
  sticky?: 'ctrl' | 'alt'
  repeat?: boolean
}

const row1: Key[] = [
  { label: 'Esc', key: 'esc' },
  { label: 'Tab', key: 'tab' },
  { label: 'Ctrl', sticky: 'ctrl' },
  { label: 'Alt', sticky: 'alt' },
  { label: '←', icon: ArrowLeft, key: 'left', repeat: true },
  { label: '↓', icon: ArrowDown, key: 'down', repeat: true },
  { label: '↑', icon: ArrowUp, key: 'up', repeat: true },
  { label: '→', icon: ArrowRight, key: 'right', repeat: true },
]
const row2: Key[] = [
  { label: 'Home', key: 'home' },
  { label: 'End', key: 'end' },
  { label: 'PgUp', key: 'pgup', repeat: true },
  { label: 'PgDn', key: 'pgdn', repeat: true },
  { label: '|', text: '|' },
  { label: '~', text: '~' },
  { label: '/', text: '/' },
  { label: '-', text: '-' },
  { label: '_', text: '_' },
]

let timer: ReturnType<typeof setTimeout> | undefined
let interval: ReturnType<typeof setInterval> | undefined

function fire(k: Key) {
  if (k.sticky === 'ctrl') {
    emit('toggle-ctrl')
  } else if (k.sticky === 'alt') {
    emit('toggle-alt')
  } else if (k.key) {
    emit('key', k.key)
  } else if (k.text) {
    emit('text', k.text)
  }
}

function stop() {
  if (timer) {
    clearTimeout(timer)
    timer = undefined
  }
  if (interval) {
    clearInterval(interval)
    interval = undefined
  }
}

// 按下立即触发一次；方向键等长按 350ms 后以 70ms 间隔连发
function down(k: Key) {
  stop()
  fire(k)
  if (k.repeat) {
    timer = setTimeout(() => {
      interval = setInterval(() => fire(k), 70)
    }, 350)
  }
}

function bind(k: Key) {
  return {
    onPointerdown: (e: PointerEvent) => {
      e.preventDefault()
      down(k)
    },
    onPointerup: stop,
    onPointercancel: stop,
    onPointerleave: stop,
    onMousedown: (e: MouseEvent) => e.preventDefault(),
    // 阻止触摸默认行为，避免按键抢走终端输入框焦点导致软键盘收起
    onTouchstart: (e: TouchEvent) => e.preventDefault(),
    'aria-label': k.label,
  }
}

onBeforeUnmount(stop)
</script>

<style scoped>
.keybar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 5px 6px calc(5px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--g-border);
  background: var(--el-bg-color);
  user-select: none;
  -webkit-user-select: none;
  touch-action: manipulation;
}
.kb {
  flex: 1 1 36px;
  min-width: 36px;
  height: 40px;
  padding: 0 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-family: var(--g-mono);
  font-size: 13px;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}
.kb:active {
  background: var(--el-color-primary-light-8);
}
.kb.on {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #fff;
}
.kb.wide {
  flex: 2 1 90px;
  font-family: var(--g-font);
}
</style>
