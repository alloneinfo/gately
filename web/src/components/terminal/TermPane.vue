<template>
  <div class="term-pane" :style="{ background: bg }">
    <div ref="host" class="term-host"></div>
    <div v-if="status === 'reconnecting' || status === 'connecting' || status === 'closed' || status === 'ended'" class="term-badge" :class="status">
      <template v-if="status === 'connecting'">{{ t('terminal.pane.connecting') }}</template>
      <template v-else-if="status === 'reconnecting'">{{ t('terminal.pane.reconnecting', { count: attempts }) }}</template>
      <template v-else>
        <span>{{ status === 'ended' ? t('terminal.pane.ended') : t('terminal.pane.closed') }}</span>
        <button v-if="status === 'closed'" class="badge-btn" @click="reconnect">{{ t('terminal.pane.reconnect') }}</button>
        <button class="badge-btn" @click="restart">{{ t('terminal.pane.newSession') }}</button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { SearchAddon } from '@xterm/addon-search'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { ApiError, api } from '../../api'
import { ensureAccess, isCancel } from './security'
import { ctrlChar, keySequence, type KeyName } from './terminalKeys'
import { findTheme } from './themes'
import { FONT_MAX, FONT_MIN } from './prefs'
import type { PaneStatus } from './types'

const { t } = useI18n()

const props = defineProps<{
  connKey: string
  // 初始附着的会话 ID，为空时新建会话
  sessionId?: string
  fontSize: number
  themeName: string
  // 心跳间隔（秒）
  heartbeat: number
  // 虚拟键盘的粘滞修饰键，仅活动窗格会传入 true
  ctrl: boolean
  alt: boolean
}>()

const emit = defineEmits<{
  session: [id: string]
  status: [s: PaneStatus]
  notice: [msg: string]
  'update:ctrl': [v: boolean]
  'update:alt': [v: boolean]
  zoom: [size: number]
}>()

const host = ref<HTMLElement>()
const status = ref<PaneStatus>('connecting')
const attempts = ref(0)
const bg = computed(() => findTheme(props.themeName).theme.background ?? '#1d1f21')

const MAX_RETRY = 30
let term: Terminal | undefined
let fit: FitAddon | undefined
let search: SearchAddon | undefined
let ws: WebSocket | undefined
let observer: ResizeObserver | undefined
let sid = props.sessionId ?? ''
let ended = false
let disposed = false
let generation = 0
let needReset = false
let retryTimer: ReturnType<typeof setTimeout> | undefined
let hbTimer: ReturnType<typeof setInterval> | undefined
let fitFrame = 0

function setStatus(s: PaneStatus) {
  status.value = s
  emit('status', s)
}

function writeNote(text: string, color = '33') {
  term?.write(`\r\n\x1b[${color}m[Gately] ${text}\x1b[0m\r\n`)
}

function sendJSON(msg: object) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg))
  }
}

function sendInput(d: string) {
  sendJSON({ t: 'i', d })
}

function startHeartbeat() {
  stopHeartbeat()
  const sec = Math.max(5, props.heartbeat || 30)
  hbTimer = setInterval(() => sendJSON({ t: 'p' }), sec * 1000)
}

function stopHeartbeat() {
  if (hbTimer) {
    clearInterval(hbTimer)
    hbTimer = undefined
  }
}

// connect 建立 WebSocket：有 sid 时附着已有会话（回放缓冲），否则新建会话。
async function connect() {
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = undefined
  }
  const gen = ++generation
  setStatus(sid && attempts.value > 0 ? 'reconnecting' : 'connecting')
  try {
    await ensureAccess()
  } catch (e) {
    if (gen !== generation || disposed) {
      return
    }
    if (isCancel(e)) {
      writeNote(t('terminal.pane.cancelled'), '90')
      setStatus('closed')
      return
    }
    if (sid) {
      scheduleRetry()
      return
    }
    writeNote((e as Error).message, '31')
    setStatus('closed')
    return
  }
  if (gen !== generation || disposed || !term) {
    return
  }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const q = `cols=${term.cols}&rows=${term.rows}`
  const path = sid
    ? `/api/webterminal/sessions/${encodeURIComponent(sid)}/ws?${q}`
    : `/api/webterminal/connections/${encodeURIComponent(props.connKey)}/ws?${q}`
  // 附着已有会话时，服务端会先回放缓冲，需要在收到 s 消息后清屏避免内容重复
  needReset = !!sid
  const sock = new WebSocket(`${proto}://${location.host}${path}`)
  sock.binaryType = 'arraybuffer'
  ws = sock
  sock.onopen = () => {
    startHeartbeat()
  }
  sock.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      onControl(ev.data)
      return
    }
    term?.write(new Uint8Array(ev.data as ArrayBuffer))
  }
  sock.onclose = () => {
    if (ws !== sock) {
      return
    }
    ws = undefined
    stopHeartbeat()
    if (ended || disposed) {
      return
    }
    if (!sid) {
      writeNote(t('terminal.pane.failed'), '31')
      setStatus('closed')
      return
    }
    scheduleRetry()
  }
}

function onControl(raw: string) {
  let m: { t: string; d?: string }
  try {
    m = JSON.parse(raw)
  } catch {
    term?.write(raw)
    return
  }
  if (m.t === 's' && m.d) {
    if (needReset) {
      term?.reset()
      needReset = false
    }
    sid = m.d
    attempts.value = 0
    emit('session', sid)
    setStatus('connected')
    if (term) {
      sendJSON({ t: 'r', c: term.cols, r: term.rows })
    }
  } else if (m.t === 'n') {
    writeNote(m.d ?? '')
    emit('notice', m.d ?? '')
  } else if (m.t === 'x') {
    ended = true
    writeNote(t('terminal.pane.endedReason', { reason: m.d ?? '' }), '31')
    setStatus('ended')
  }
}

// scheduleRetry 按指数退避（1s、2s、4s … 最长 30s）重新附着到会话。
function scheduleRetry() {
  attempts.value++
  if (attempts.value > MAX_RETRY) {
    writeNote(t('terminal.pane.retryGiveUp'), '31')
    setStatus('closed')
    return
  }
  setStatus('reconnecting')
  const delay = Math.min(30000, 1000 * Math.pow(2, attempts.value - 1))
  retryTimer = setTimeout(async () => {
    retryTimer = undefined
    if (disposed || ended) {
      return
    }
    if (attempts.value >= 2) {
      // 多次失败后确认会话是否仍然存在，已被回收则不再重连
      try {
        await api.get(`/api/webterminal/sessions/${encodeURIComponent(sid)}`)
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) {
          ended = true
          writeNote(t('terminal.pane.sessionGone'), '31')
          setStatus('ended')
          return
        }
      }
    }
    connect()
  }, delay)
}

// reconnect 立即重连当前会话（用于手动重连、网络恢复、页面重新可见）。
function reconnect() {
  if (disposed || ended) {
    return
  }
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }
  attempts.value = sid ? 1 : 0
  connect()
}

// restart 丢弃旧会话信息，新建一个全新的会话。
function restart() {
  generation++
  ws?.close()
  ws = undefined
  sid = ''
  ended = false
  attempts.value = 0
  term?.reset()
  connect()
}

function onWake() {
  if (document.visibilityState === 'visible') {
    reconnect()
  }
}

// applyModifiers 把软键盘输入与粘滞的 Ctrl / Alt 合并。
function applyModifiers(d: string): string {
  if (!props.ctrl && !props.alt) {
    return d
  }
  if (d.length !== 1) {
    return d
  }
  let out = d
  if (props.ctrl) {
    out = ctrlChar(out)
    emit('update:ctrl', false)
  }
  if (props.alt) {
    out = '\x1b' + out
    emit('update:alt', false)
  }
  return out
}

// sendKey 发送虚拟按键，自动带上粘滞修饰键。
function sendKey(name: KeyName) {
  const seq = keySequence(name, {
    appCursor: !!term?.modes.applicationCursorKeysMode,
    ctrl: props.ctrl,
    alt: props.alt,
  })
  if (props.ctrl) {
    emit('update:ctrl', false)
  }
  if (props.alt) {
    emit('update:alt', false)
  }
  sendInput(seq)
}

// sendText 发送文本；单字符会合并粘滞修饰键，多字符原样发送。
function sendText(text: string) {
  sendInput(applyModifiers(text))
}

function doFit() {
  if (fitFrame) {
    return
  }
  fitFrame = requestAnimationFrame(() => {
    fitFrame = 0
    const el = host.value
    if (!el || !term || el.clientWidth < 20 || el.clientHeight < 20) {
      return
    }
    try {
      fit?.fit()
    } catch {
      // 容器尚未完成布局时忽略
    }
  })
}

// ---- 触屏手势：单指滚动回看历史，双指缩放字号 ----
let tMode: '' | 'scroll' | 'pinch' = ''
let tStartY = 0
let tLastY = 0
let tAcc = 0
let tMoved = false
let tDist0 = 1
let tSize0 = 13

function touchDist(e: TouchEvent): number {
  const a = e.touches[0]
  const b = e.touches[1]
  return Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY) || 1
}

function onTouchStart(e: TouchEvent) {
  if (e.touches.length === 2) {
    tMode = 'pinch'
    tDist0 = touchDist(e)
    tSize0 = props.fontSize
    return
  }
  if (e.touches.length === 1) {
    tMode = 'scroll'
    tStartY = e.touches[0].clientY
    tLastY = tStartY
    tAcc = 0
    tMoved = false
  }
}

function onTouchMove(e: TouchEvent) {
  if (!term) {
    return
  }
  if (tMode === 'pinch' && e.touches.length === 2) {
    e.preventDefault()
    e.stopPropagation()
    const size = Math.min(FONT_MAX, Math.max(FONT_MIN, Math.round((tSize0 * touchDist(e)) / tDist0)))
    if (size !== props.fontSize) {
      emit('zoom', size)
    }
    return
  }
  if (tMode === 'scroll' && e.touches.length === 1) {
    const y = e.touches[0].clientY
    if (Math.abs(y - tStartY) > 6) {
      tMoved = true
    }
    if (!tMoved) {
      return
    }
    e.preventDefault()
    e.stopPropagation()
    tAcc += tLastY - y
    tLastY = y
    const lineH = Math.max(8, (host.value?.clientHeight ?? 0) / Math.max(1, term.rows))
    const lines = Math.trunc(tAcc / lineH)
    if (lines !== 0) {
      tAcc -= lines * lineH
      if (term.buffer.active.type === 'alternate') {
        // vim、less 等全屏程序没有回滚缓冲，改为发送方向键翻动
        const seq = lines > 0 ? '\x1b[B' : '\x1b[A'
        sendInput(seq.repeat(Math.min(Math.abs(lines), 10)))
      } else {
        term.scrollLines(lines)
      }
    }
  }
}

function onTouchEnd(e: TouchEvent) {
  if (tMode === 'scroll' && !tMoved) {
    term?.focus()
  }
  if (e.touches.length === 0) {
    tMode = ''
  }
}

let lastDataAt = 0
let lastData = ''

// iOS 第三方键盘（如搜狗英文模式）常以 keyCode=229 且无 composition 事件的方式提交字符，
// xterm 自身的 textarea 差值检测可能漏掉它们。这里监听 input 事件，
// 若 xterm 在短时间内没有通过 onData 发出该字符，则手动补发，避免丢字也避免重复。
function setupIOSImeFallback() {
  const ta = term?.textarea
  if (!ta) {
    return
  }
  const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  let composing = false
  let lastKeyCode = 0
  ta.addEventListener('keydown', (e) => {
    lastKeyCode = e.keyCode
  }, true)
  ta.addEventListener('compositionstart', () => {
    composing = true
  }, true)
  ta.addEventListener('compositionend', () => {
    composing = false
  }, true)
  ta.addEventListener('input', (e) => {
    const ie = e as InputEvent
    if (!isIOS || composing || lastKeyCode !== 229 || ie.inputType !== 'insertText' || !ie.data) {
      return
    }
    const data = ie.data
    const t0 = Date.now()
    setTimeout(() => {
      // xterm 已经在此期间发出同样的数据，无需补发
      if (lastDataAt >= t0 && lastData.includes(data)) {
        return
      }
      term?.input(data, true)
    }, 60)
  }, true)
}

onMounted(() => {
  const el = host.value!
  term = new Terminal({
    cursorBlink: true,
    fontSize: props.fontSize,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
    theme: findTheme(props.themeName).theme,
    scrollback: 5000,
    allowProposedApi: true,
  })
  fit = new FitAddon()
  search = new SearchAddon()
  term.loadAddon(fit)
  term.loadAddon(search)
  term.loadAddon(new WebLinksAddon())
  term.open(el)
  fit.fit()

  term.onData((d) => {
    lastDataAt = Date.now()
    lastData = d
    sendInput(applyModifiers(d))
  })
  setupIOSImeFallback()
  term.onResize(({ cols, rows }) => sendJSON({ t: 'r', c: cols, r: rows }))

  observer = new ResizeObserver(() => doFit())
  observer.observe(el)
  el.addEventListener('touchstart', onTouchStart, { passive: true, capture: true })
  el.addEventListener('touchmove', onTouchMove, { passive: false, capture: true })
  el.addEventListener('touchend', onTouchEnd, { passive: true, capture: true })
  window.addEventListener('online', reconnect)
  document.addEventListener('visibilitychange', onWake)

  connect()
})

watch(
  () => props.fontSize,
  (v) => {
    if (term) {
      term.options.fontSize = v
      doFit()
    }
  },
)
// 选中“跟随界面主题”时，界面换肤也会让配色重新计算
watch(
  () => findTheme(props.themeName).theme,
  (th) => {
    if (term) {
      term.options.theme = th
    }
  },
)
watch(
  () => props.heartbeat,
  () => {
    if (hbTimer) {
      startHeartbeat()
    }
  },
)

onBeforeUnmount(() => {
  disposed = true
  generation++
  if (retryTimer) {
    clearTimeout(retryTimer)
  }
  stopHeartbeat()
  observer?.disconnect()
  window.removeEventListener('online', reconnect)
  document.removeEventListener('visibilitychange', onWake)
  const el = host.value
  el?.removeEventListener('touchstart', onTouchStart, { capture: true })
  el?.removeEventListener('touchmove', onTouchMove, { capture: true })
  el?.removeEventListener('touchend', onTouchEnd, { capture: true })
  ws?.close()
  term?.dispose()
  if (fitFrame) {
    cancelAnimationFrame(fitFrame)
  }
})

// ---- 暴露给父组件的操作 ----
function focus() {
  term?.focus()
}
function refit() {
  nextTick(doFit)
}
function getSelection(): string {
  return term?.getSelection() ?? ''
}
// getScreenText 返回当前可视屏幕的文本，用于没有选区时的复制兜底。
function getScreenText(): string {
  if (!term) {
    return ''
  }
  const buf = term.buffer.active
  const lines: string[] = []
  for (let i = 0; i < term.rows; i++) {
    const line = buf.getLine(buf.viewportY + i)
    lines.push(line ? line.translateToString(true) : '')
  }
  return lines.join('\n').replace(/\s+$/, '')
}
function paste(text: string) {
  term?.paste(text)
}
function clear() {
  term?.clear()
}
function findNext(q: string): boolean {
  return search?.findNext(q, { incremental: false }) ?? false
}
function findPrev(q: string): boolean {
  return search?.findPrevious(q) ?? false
}
function clearSearch() {
  search?.clearDecorations()
  term?.clearSelection()
}

defineExpose({ focus, refit, getSelection, getScreenText, paste, clear, findNext, findPrev, clearSearch, sendKey, sendText, sendInput, reconnect, restart })
</script>

<style scoped>
.term-pane {
  position: relative;
  height: 100%;
  width: 100%;
  padding: 6px 0 6px 8px;
  box-sizing: border-box;
  overflow: hidden;
  /* 单指拖动由脚本接管为回滚，避免浏览器自带的页面滚动 */
  touch-action: none;
}
.term-host {
  height: 100%;
  width: 100%;
}
.term-badge {
  position: absolute;
  top: 8px;
  right: 14px;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  border-radius: 14px;
  font-size: 12px;
  color: #fff;
  background: rgba(0, 0, 0, 0.7);
}
.term-badge.closed,
.term-badge.ended {
  background: rgba(190, 60, 60, 0.9);
}
.badge-btn {
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  background: transparent;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}
</style>
