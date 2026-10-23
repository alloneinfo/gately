<template>
  <!-- 容器内终端：xterm + exec WebSocket，移动端提供特殊键工具栏 -->
  <div class="ct-term">
    <div class="opts">
      <el-select v-model="shell" allow-create filterable default-first-option class="o-shell" :disabled="connected">
        <el-option label="/bin/sh" value="/bin/sh" />
        <el-option label="/bin/bash" value="/bin/bash" />
        <el-option label="/bin/ash" value="/bin/ash" />
      </el-select>
      <el-input v-model="user" :placeholder="t('docker.container.term.userPlaceholder')" class="o-user" :disabled="connected" />
      <el-button v-if="!connected" type="primary" :icon="Plug" @click="connect">{{ t('docker.container.term.connect') }}</el-button>
      <el-button v-else :icon="Unplug" @click="disconnect">{{ t('docker.container.term.disconnect') }}</el-button>
    </div>
    <div class="term-box" :style="{ background: bg }">
      <div ref="el" class="term-pane"></div>
    </div>
    <div v-if="isMobile" class="keys">
      <button v-for="k in keys" :key="k.label" type="button" class="key" :class="{ on: k.sticky && ctrl }" @pointerdown.prevent @click="press(k)">{{ k.label }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plug, Unplug } from 'lucide-vue-next'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useIsMobile } from '../../composables/useIsMobile'
import { useTermPrefs } from '../../components/terminal/prefs'
import { findTheme } from '../../components/terminal/themes'

const { t } = useI18n()

const props = defineProps<{ containerId: string }>()
const isMobile = useIsMobile()
// 配色沿用终端页的偏好（默认跟随界面主题）
const { themeName } = useTermPrefs()
const termTheme = computed(() => findTheme(themeName.value).theme)
const bg = computed(() => termTheme.value.background ?? '#1d1f21')

const el = ref<HTMLElement>()
const shell = ref('/bin/sh')
const user = ref('')
const connected = ref(false)
const ctrl = ref(false)
let term: Terminal | undefined
let ws: WebSocket | undefined
let observer: ResizeObserver | undefined
let fitTimer: number | undefined
const fit = new FitAddon()

interface Key {
  label: string
  seq?: string
  sticky?: boolean
}

// 移动端软键盘缺少的常用按键
const keys: Key[] = [
  { label: 'Esc', seq: '\x1b' },
  { label: 'Tab', seq: '\t' },
  { label: 'Ctrl', sticky: true },
  { label: '↑', seq: '\x1b[A' },
  { label: '↓', seq: '\x1b[B' },
  { label: '←', seq: '\x1b[D' },
  { label: '→', seq: '\x1b[C' },
  { label: '|', seq: '|' },
  { label: '/', seq: '/' },
  { label: '-', seq: '-' },
  { label: '~', seq: '~' },
  { label: '.', seq: '.' },
  { label: 'Home', seq: '\x1b[H' },
  { label: 'End', seq: '\x1b[F' },
  { label: '^C', seq: '\x03' },
  { label: '^D', seq: '\x04' },
]

function send(msg: object) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg))
  }
}

function press(k: Key) {
  if (k.sticky) {
    ctrl.value = !ctrl.value
    term?.focus()
    return
  }
  if (k.seq) {
    sendInput(k.seq)
  }
  term?.focus()
}

// Ctrl 粘滞：下一次输入的单个字符转换为控制字符
function sendInput(d: string) {
  if (ctrl.value && d.length === 1) {
    const c = d.toUpperCase().charCodeAt(0)
    ctrl.value = false
    if (c >= 64 && c <= 95) {
      send({ t: 'i', d: String.fromCharCode(c - 64) })
      return
    }
  }
  send({ t: 'i', d })
}

function doFit() {
  window.clearTimeout(fitTimer)
  fitTimer = window.setTimeout(() => {
    try {
      fit.fit()
    } catch {
      // 容器尚未可见时忽略
    }
  }, 60)
}

function disconnect() {
  ws?.close()
}

function connect() {
  if (!term || connected.value) {
    return
  }
  term.reset()
  doFit()
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const q = new URLSearchParams({ cmd: shell.value || '/bin/sh', cols: String(term.cols), rows: String(term.rows) })
  if (user.value.trim()) {
    q.set('user', user.value.trim())
  }
  const sock = new WebSocket(`${proto}://${location.host}/api/docker/containers/${props.containerId}/exec?${q.toString()}`)
  ws = sock
  sock.binaryType = 'arraybuffer'
  sock.onopen = () => {
    connected.value = true
    send({ t: 'r', c: term!.cols, r: term!.rows })
    term?.focus()
  }
  sock.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      try {
        const m = JSON.parse(ev.data)
        term?.write(`\r\n\x1b[33m[Gately] ${m.d}\x1b[0m\r\n`)
      } catch {
        term?.write(ev.data)
      }
      return
    }
    term?.write(new Uint8Array(ev.data))
  }
  sock.onclose = () => {
    if (ws === sock) {
      connected.value = false
      term?.write(`\r\n\x1b[90m${t('docker.container.term.closed')}\x1b[0m\r\n`)
    }
  }
  sock.onerror = () => {
    term?.write(`\r\n\x1b[31m${t('docker.container.term.failed')}\x1b[0m\r\n`)
  }
}

onMounted(() => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: isMobile.value ? 12 : 13,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
    theme: termTheme.value,
  })
  term.loadAddon(fit)
  term.open(el.value!)
  fit.fit()
  term.onData((d) => sendInput(d))
  term.onResize(({ cols, rows }) => send({ t: 'r', c: cols, r: rows }))
  observer = new ResizeObserver(() => doFit())
  observer.observe(el.value!)
  window.addEventListener('orientationchange', doFit)
  connect()
})

watch(termTheme, (th) => {
  if (term) {
    term.options.theme = th
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('orientationchange', doFit)
  window.clearTimeout(fitTimer)
  observer?.disconnect()
  const s = ws
  ws = undefined
  s?.close()
  term?.dispose()
})
</script>

<style scoped>
.ct-term {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.opts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.opts .el-button + .el-button {
  margin-left: 0;
}
.o-shell {
  width: 150px;
}
.o-user {
  width: 150px;
}
.term-box {
  height: 56vh;
  min-height: 260px;
  border-radius: var(--g-radius-sm);
  overflow: hidden;
}
.term-pane {
  height: 100%;
  width: 100%;
  padding: 6px;
  box-sizing: border-box;
}
.keys {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 6px;
  padding-bottom: env(safe-area-inset-bottom);
}
.key {
  min-height: 40px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-family: var(--g-mono);
  padding: 0;
  touch-action: manipulation;
}
.key:active {
  background: var(--el-fill-color);
}
.key.on {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #fff;
}
@media (max-width: 767.98px) {
  .term-box {
    height: calc(100dvh - 330px);
    min-height: 200px;
  }
  .o-shell,
  .o-user {
    flex: 1 1 120px;
    width: auto;
  }
  .opts :deep(input) {
    font-size: 16px;
  }
}
</style>
