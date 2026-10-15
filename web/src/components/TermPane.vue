<template>
  <div ref="el" class="term-pane"></div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{ connKey: string }>()
const emit = defineEmits<{ closed: [] }>()
const el = ref<HTMLElement>()
let term: Terminal | undefined
let ws: WebSocket | undefined
let observer: ResizeObserver | undefined
const fit = new FitAddon()

function send(msg: object) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg))
  }
}

onMounted(() => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
    theme: { background: '#1d1f21' },
  })
  term.loadAddon(fit)
  term.open(el.value!)
  fit.fit()
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${location.host}/api/webterminal/connections/${props.connKey}/ws?cols=${term.cols}&rows=${term.rows}`)
  ws.binaryType = 'arraybuffer'
  ws.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      // 文本消息为服务端提示
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
  ws.onclose = () => {
    term?.write('\r\n\x1b[90m[连接已关闭]\x1b[0m\r\n')
    emit('closed')
  }
  term.onData((d) => send({ t: 'i', d }))
  term.onResize(({ cols, rows }) => send({ t: 'r', c: cols, r: rows }))
  observer = new ResizeObserver(() => fit.fit())
  observer.observe(el.value!)
  term.focus()
})

onBeforeUnmount(() => {
  observer?.disconnect()
  ws?.close()
  term?.dispose()
})
</script>

<style scoped>
.term-pane {
  height: 100%;
  width: 100%;
  background: #1d1f21;
  padding: 6px;
  box-sizing: border-box;
}
</style>
