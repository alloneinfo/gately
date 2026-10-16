<template>
  <div class="terminal-root" :class="{ mobile: isMobile }">
    <PageHeader v-if="!inWorkspace" title="Web 终端" desc="在浏览器中连接 SSH、Telnet 服务器或本机 Shell，会话可断线重连" :icon="SquareTerminal">
      <el-button :icon="ListTree" @click="sessionsVisible = true">会话列表</el-button>
      <el-button :icon="Zap" @click="shortcutVisible = true">快捷命令</el-button>
      <el-button :icon="SlidersHorizontal" @click="settingsVisible = true">设置</el-button>
      <el-button type="primary" :icon="Plus" @click="openEditor(null)">添加连接</el-button>
    </PageHeader>

    <div class="terminal-page" v-show="!inWorkspace || !isMobile">
      <div class="side">
        <button v-if="isMobile && tabs.length" class="back-banner" @click="mobileView = 'term'">
          <SquareTerminal :size="18" />返回终端（{{ tabs.length }} 个会话）<ChevronRight :size="16" />
        </button>
        <ConnectionList
          :conns="conns"
          :is-mobile="isMobile"
          :open-counts="openCounts"
          @connect="connect"
          @edit="openEditor"
          @sftp="openSftp"
          @quick="openQuick"
          @reload="loadConns"
        />
      </div>
    </div>

    <div
      v-show="!isMobile || inWorkspace"
      class="workspace g-card"
      :class="{ full: fullscreen && !isMobile, 'm-full': isMobile, empty: !tabs.length }"
      :style="wsStyle"
    >
      <template v-if="tabs.length">
        <div class="ws-head">
          <el-button v-if="isMobile" class="back" text :icon="ArrowLeft" aria-label="返回连接列表" @click="mobileView = 'list'" />
          <TermTabs
            :tabs="tabs"
            :active="active"
            :conns="conns"
            :split-mode="splitMode"
            :split-ids="splitIds"
            :is-mobile="isMobile"
            @select="selectTab"
            @close="closeTab"
            @command="onTabCommand"
            @add="onAddTab"
            @add-list="mobileView = 'list'"
          />
          <TermToolbar
            v-if="isMobile"
            v-model:theme-name="themeName"
            :is-mobile="isMobile"
            :sftp-enabled="sftpEnabled"
            :font-size="fontSize"
            :shortcuts="shortcuts"
            :split-mode="splitMode"
            :fullscreen="fullscreen"
            @action="onAction"
            @shortcut="sendShortcut"
          />
        </div>
        <div v-if="!isMobile" class="ws-tools">
          <TermToolbar
            v-model:theme-name="themeName"
            :is-mobile="isMobile"
            :sftp-enabled="sftpEnabled"
            :font-size="fontSize"
            :shortcuts="shortcuts"
            :split-mode="splitMode"
            :fullscreen="fullscreen"
            @action="onAction"
            @shortcut="sendShortcut"
          />
        </div>

        <div class="panes" :class="[splitMode && !isMobile ? 'split split-' + splitDir : 'single']">
          <div
            v-for="t in tabs"
            v-show="isShown(t)"
            :key="t.id"
            class="cell"
            :class="{ focus: splitMode && !isMobile && t.id === active }"
            @pointerdown="active = t.id"
          >
            <div v-if="splitMode && !isMobile" class="cell-head">
              <span class="dot" :class="t.status" />
              <span class="ellipsis">{{ t.title }}</span>
              <el-button v-if="splitIds.length > 1" text size="small" :icon="X" aria-label="移出分屏" @click.stop="leaveSplit(t.id)" />
            </div>
            <div class="cell-body">
              <TermPane
                :ref="(el) => setPane(t.id, el)"
                :conn-key="t.connKey"
                :session-id="t.sessionId"
                :font-size="fontSize"
                :theme-name="themeName"
                :heartbeat="settings?.heartbeatInterval ?? 30"
                :ctrl="t.id === active && mods.ctrl"
                :alt="t.id === active && mods.alt"
                @session="(id: string) => onSession(t, id)"
                @status="(s: PaneStatus) => (t.status = s)"
                @update:ctrl="(v: boolean) => (mods.ctrl = v)"
                @update:alt="(v: boolean) => (mods.alt = v)"
                @zoom="setFont"
              />
            </div>
          </div>

          <div v-if="searchVisible" class="search-bar">
            <el-input ref="searchInput" v-model="searchText" size="small" placeholder="搜索终端内容" clearable @input="doSearch(true)" @keyup.enter.exact="doSearch(true)" @keyup.shift.enter="doSearch(false)" @keyup.esc="closeSearch" />
            <el-button class="sb" text :icon="ChevronUp" aria-label="上一个" @click="doSearch(false)" />
            <el-button class="sb" text :icon="ChevronDown" aria-label="下一个" @click="doSearch(true)" />
            <el-button class="sb" text :icon="X" aria-label="关闭搜索" @click="closeSearch" />
          </div>
        </div>

        <MobileKeybar
          v-if="isMobile"
          :ctrl="mods.ctrl"
          :alt="mods.alt"
          @key="(k: KeyName) => activePane()?.sendKey(k)"
          @text="(s: string) => activePane()?.sendText(s)"
          @toggle-ctrl="mods.ctrl = !mods.ctrl"
          @toggle-alt="mods.alt = !mods.alt"
          @shortcuts="shortcutSheet = true"
          @focus="activePane()?.focus()"
        />
      </template>
      <div v-else class="placeholder">
        <SquareTerminal :size="40" :stroke-width="1.5" />
        <div>选择一个连接开始会话</div>
        <div class="meta">点击「连接」或双击连接项；会话在服务端保留时，可从「会话列表」找回</div>
      </div>
    </div>

    <ConnectionEditor v-model="editorVisible" :conn="editingConn" :allow-local-shell="!!settings?.allowLocalShell" @saved="loadConns" />
    <QuickAccessDialog v-model="quickVisible" :conn="quickConn" @saved="loadConns" />
    <SettingsDialog v-model="settingsVisible" @saved="onSettingsSaved" />
    <ShortcutManager v-model="shortcutVisible" :shortcuts="shortcuts" @saved="(l: Shortcut[]) => (shortcuts = l)" />
    <ShortcutSheet v-model="shortcutSheet" :shortcuts="shortcuts" @send="sendShortcut" @manage="manageFromSheet" />
    <SessionList
      v-model="sessionsVisible"
      :open-ids="tabs.map((t) => t.sessionId).filter((s) => s)"
      @attach="attachSession"
      @killed="onSessionKilled"
      @remark="onRemarkChanged"
    />
    <SftpPanel v-model="sftpVisible" :conn="sftpConn" @quick="openQuick(sftpConn)" @quick-saved="loadConns" />
    <SecurityGate />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, ChevronDown, ChevronRight, ChevronUp, ListTree, Plus, SlidersHorizontal, SquareTerminal, X, Zap } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import { useIsMobile } from '../composables/useIsMobile'
import ConnectionEditor from '../components/terminal/ConnectionEditor.vue'
import ConnectionList from '../components/terminal/ConnectionList.vue'
import MobileKeybar from '../components/terminal/MobileKeybar.vue'
import QuickAccessDialog from '../components/terminal/QuickAccessDialog.vue'
import SecurityGate from '../components/terminal/SecurityGate.vue'
import SessionList from '../components/terminal/SessionList.vue'
import SettingsDialog from '../components/terminal/SettingsDialog.vue'
import SftpPanel from '../components/terminal/SftpPanel.vue'
import ShortcutManager from '../components/terminal/ShortcutManager.vue'
import ShortcutSheet from '../components/terminal/ShortcutSheet.vue'
import TermPane from '../components/terminal/TermPane.vue'
import TermTabs from '../components/terminal/TermTabs.vue'
import TermToolbar from '../components/terminal/TermToolbar.vue'
import { copyText, readClipboardText } from '../components/terminal/clipboard'
import { useTermPrefs } from '../components/terminal/prefs'
import type { KeyName } from '../components/terminal/terminalKeys'
import { normalizeConn, type Conn, type PaneStatus, type SessionInfo, type Shortcut, type SplitLayout, type SplitMode, type TermSettings, type TermTab } from '../components/terminal/types'
import { useVisualViewport } from '../components/terminal/useVisualViewport'
import '../components/terminal/terminal.css'

type PaneExpose = InstanceType<typeof TermPane>

const TABS_KEY = 'gately-term-tabs'
// 桌面端分屏最多同时显示的窗格数
const MAX_SPLIT = 4

const isMobile = useIsMobile()
const vv = useVisualViewport()
const { fontSize, themeName, setFont, incFont, decFont } = useTermPrefs()

const conns = ref<Conn[]>([])
const settings = ref<TermSettings>()
const shortcuts = ref<Shortcut[]>([])
const tabs = ref<TermTab[]>([])
const active = ref('')
const mobileView = ref<'list' | 'term'>('list')
const fullscreen = ref(false)
const mods = reactive({ ctrl: false, alt: false })
let seq = 0

// 分屏状态（仅桌面端）
const splitMode = ref(false)
const splitDir = ref<SplitMode>('h')
const splitIds = ref<string[]>([])

const editorVisible = ref(false)
const editingConn = ref<Conn | null>(null)
const quickVisible = ref(false)
const quickConn = ref<Conn | null>(null)
const settingsVisible = ref(false)
const shortcutVisible = ref(false)
const shortcutSheet = ref(false)
const sessionsVisible = ref(false)
const sftpVisible = ref(false)
const sftpConn = ref<Conn | null>(null)

const searchVisible = ref(false)
const searchText = ref('')
const searchInput = ref<{ focus: () => void }>()

const panes = new Map<string, PaneExpose>()

const inWorkspace = computed(() => isMobile.value && mobileView.value === 'term' && tabs.value.length > 0)
const activeTab = computed(() => tabs.value.find((t) => t.id === active.value))
const sftpEnabled = computed(() => !!activeTab.value && activeTab.value.type !== 'telnet')
const openCounts = computed(() => {
  const m: Record<string, number> = {}
  for (const t of tabs.value) {
    m[t.connKey] = (m[t.connKey] ?? 0) + 1
  }
  return m
})

// 移动端终端区域跟随可视视口，软键盘弹出时自动缩小
const wsStyle = computed(() => {
  if (isMobile.value && vv.supported && vv.height.value > 0) {
    return { top: vv.top.value + 'px', height: vv.height.value + 'px' }
  }
  return {}
})

function setPane(id: string, el: unknown) {
  if (el) {
    panes.set(id, el as PaneExpose)
  } else {
    panes.delete(id)
  }
}

function activePane(): PaneExpose | undefined {
  return panes.get(active.value)
}

function isShown(t: TermTab): boolean {
  if (splitMode.value && !isMobile.value) {
    return splitIds.value.includes(t.id)
  }
  return t.id === active.value
}

// ---- 数据加载 ----
async function loadConns() {
  try {
    const list = await api.get<Partial<Conn>[] | null>('/api/webterminal/connections')
    conns.value = (list ?? []).map((c) => normalizeConn(c))
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function loadSettings() {
  try {
    const r = await api.get<{ settings: TermSettings }>('/api/webterminal/settings')
    settings.value = r.settings
  } catch {
    // 设置读取失败时使用默认心跳间隔
  }
}

async function loadShortcuts() {
  try {
    shortcuts.value = (await api.get<Shortcut[] | null>('/api/webterminal/globalshortcuts')) ?? []
  } catch {
    shortcuts.value = []
  }
}

function onSettingsSaved(s: TermSettings) {
  settings.value = s
}

// ---- 标签 ----
function uniqueTitle(base: string): string {
  const same = tabs.value.filter((t) => t.connName === base).length
  return same > 0 ? `${base} (${same + 1})` : base
}

function addTab(info: { connKey: string; connName: string; type: string; sessionId?: string; title?: string; remark?: string }): TermTab {
  const tab: TermTab = {
    id: `t${++seq}`,
    connKey: info.connKey,
    connName: info.connName,
    type: info.type,
    sessionId: info.sessionId ?? '',
    title: info.title || uniqueTitle(info.connName),
    remark: info.remark ?? '',
    status: 'connecting',
  }
  const prevActive = active.value
  tabs.value.push(tab)
  active.value = tab.id
  mobileView.value = 'term'
  if (splitMode.value && !isMobile.value) {
    if (splitIds.value.length < MAX_SPLIT) {
      splitIds.value.push(tab.id)
    } else {
      // 窗格已满时替换原先聚焦的窗格
      splitIds.value.splice(Math.max(0, splitIds.value.indexOf(prevActive)), 1, tab.id)
    }
  }
  return tab
}

function connect(c: Conn) {
  if (!c.enable) {
    ElMessage.warning('该连接已停用')
    return
  }
  if (c.type === 'local' && settings.value && !settings.value.allowLocalShell) {
    ElMessage.warning('请先在「设置」中开启「允许本机 Shell」')
    return
  }
  addTab({ connKey: c.key, connName: c.name, type: c.type })
}

function onAddTab(connKey: string) {
  const c = conns.value.find((x) => x.key === connKey)
  if (c) {
    connect(c)
  }
}

function selectTab(id: string) {
  active.value = id
  if (splitMode.value && !isMobile.value && !splitIds.value.includes(id)) {
    joinSplit(id)
  }
  nextTick(() => panes.get(id)?.focus())
}

function closeTab(id: string) {
  const i = tabs.value.findIndex((t) => t.id === id)
  if (i < 0) {
    return
  }
  tabs.value.splice(i, 1)
  splitIds.value = splitIds.value.filter((x) => x !== id)
  if (active.value === id) {
    active.value = tabs.value.length ? tabs.value[Math.max(0, i - 1)].id : ''
  }
  if (!tabs.value.length) {
    mobileView.value = 'list'
    fullscreen.value = false
    searchVisible.value = false
    if (splitMode.value) {
      exitSplit()
    }
  }
}

function onSession(t: TermTab, id: string) {
  t.sessionId = id
}

async function onTabCommand(id: string, cmd: string) {
  const t = tabs.value.find((x) => x.id === id)
  if (!t) {
    return
  }
  try {
    if (cmd === 'rename') {
      const { value } = await ElMessageBox.prompt('标签名称', '重命名', { inputValue: t.title, inputValidator: (v) => !!v.trim() || '名称不能为空' })
      t.title = value.trim()
    } else if (cmd === 'remark') {
      if (!t.sessionId) {
        ElMessage.warning('会话尚未建立，请稍后再试')
        return
      }
      const { value } = await ElMessageBox.prompt('会话备注（保存在服务端，便于在会话列表中识别）', '会话备注', { inputValue: t.remark })
      await api.put(`/api/webterminal/sessions/${encodeURIComponent(t.sessionId)}/remark`, { remark: value ?? '' })
      t.remark = value ?? ''
    } else if (cmd === 'duplicate') {
      onAddTab(t.connKey)
    } else if (cmd === 'join') {
      joinSplit(id)
    } else if (cmd === 'leave') {
      leaveSplit(id)
    } else if (cmd === 'reconnect') {
      const p = panes.get(id)
      if (t.status === 'ended') {
        p?.restart()
      } else {
        p?.reconnect()
      }
    } else if (cmd === 'kill') {
      await ElMessageBox.confirm(`结束会话「${t.title}」？会话中运行的程序会被终止。`, '确认', { type: 'warning' })
      if (t.sessionId) {
        await api.del(`/api/webterminal/sessions/${encodeURIComponent(t.sessionId)}`).catch(() => undefined)
      }
      closeTab(id)
    }
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}

// ---- 会话列表 ----
function attachSession(s: SessionInfo) {
  const exist = tabs.value.find((t) => t.sessionId === s.id)
  if (exist) {
    selectTab(exist.id)
    mobileView.value = 'term'
  } else {
    addTab({ connKey: s.connKey, connName: s.connName || s.connKey, type: s.type, sessionId: s.id, remark: s.remark })
  }
  sessionsVisible.value = false
}

function onSessionKilled(id: string) {
  const t = tabs.value.find((x) => x.sessionId === id)
  if (t) {
    closeTab(t.id)
  }
}

function onRemarkChanged(id: string, remark: string) {
  const t = tabs.value.find((x) => x.sessionId === id)
  if (t) {
    t.remark = remark
  }
}

// ---- 连接管理 ----
function openEditor(c: Conn | null) {
  editingConn.value = c
  editorVisible.value = true
}

function openQuick(c: Conn | null) {
  quickConn.value = c
  quickVisible.value = true
}

function openSftp(c: Conn) {
  if (c.type === 'telnet') {
    ElMessage.warning('Telnet 连接不支持文件管理')
    return
  }
  sftpConn.value = c
  sftpVisible.value = true
}

// 目录列表刷新后，同步面板使用的连接对象（快速访问目录可能已变更）
watch(conns, (list) => {
  if (sftpConn.value) {
    sftpConn.value = list.find((c) => c.key === sftpConn.value?.key) ?? sftpConn.value
  }
  if (quickConn.value) {
    quickConn.value = list.find((c) => c.key === quickConn.value?.key) ?? quickConn.value
  }
})

// ---- 工具栏动作 ----
async function onAction(name: string) {
  const p = activePane()
  switch (name) {
    case 'sftp': {
      const c = conns.value.find((x) => x.key === activeTab.value?.connKey)
      if (c) {
        openSftp(c)
      }
      break
    }
    case 'search':
      searchVisible.value = !searchVisible.value
      if (searchVisible.value) {
        nextTick(() => searchInput.value?.focus())
      } else {
        closeSearch()
      }
      break
    case 'copy': {
      let text = p?.getSelection() ?? ''
      let tip = '已复制'
      if (!text) {
        text = p?.getScreenText() ?? ''
        tip = '未选中内容，已复制当前屏幕文本'
      }
      if (!text) {
        ElMessage.info('没有可复制的内容')
        break
      }
      if (await copyText(text)) {
        ElMessage.success(tip)
      } else {
        ElMessage.error('复制失败，请手动选择文字后使用浏览器复制')
      }
      break
    }
    case 'paste':
      await pasteToTerminal()
      break
    case 'font-inc':
      incFont()
      break
    case 'font-dec':
      decFont()
      break
    case 'clear':
      p?.clear()
      break
    case 'sessions':
      sessionsVisible.value = true
      break
    case 'fullscreen':
      fullscreen.value = !fullscreen.value
      nextTick(() => p?.refit())
      break
    case 'shortcuts-manage':
      shortcutVisible.value = true
      break
    case 'split-h':
      enterSplit('h')
      break
    case 'split-v':
      enterSplit('v')
      break
    case 'split-grid':
      enterSplit('grid')
      break
    case 'split-off':
      exitSplit()
      break
  }
}

async function pasteToTerminal() {
  const p = activePane()
  if (!p) {
    return
  }
  let text = await readClipboardText()
  if (text === null) {
    // 浏览器不允许读取剪贴板（非 HTTPS 或被拒绝）时，改为手动粘贴
    try {
      const r = await ElMessageBox.prompt('浏览器未授权读取剪贴板，请在此粘贴要发送的内容', '粘贴', { inputType: 'textarea' })
      text = r.value
    } catch {
      return
    }
  }
  if (text) {
    p.paste(text)
    p.focus()
  }
}

function sendShortcut(s: Shortcut) {
  const p = activePane()
  if (!p) {
    ElMessage.warning('请先打开一个终端会话')
    return
  }
  // 多行命令按回车分隔发送
  const cmd = s.command.replace(/\r?\n/g, '\r') + (s.sendEnter ? '\r' : '')
  p.sendInput(cmd)
  shortcutSheet.value = false
  p.focus()
}

function manageFromSheet() {
  shortcutSheet.value = false
  shortcutVisible.value = true
}

// ---- 搜索 ----
function doSearch(forward: boolean) {
  const p = activePane()
  if (!p || !searchText.value) {
    return
  }
  const found = forward ? p.findNext(searchText.value) : p.findPrev(searchText.value)
  if (!found) {
    ElMessage.info('没有找到匹配内容')
  }
}

function closeSearch() {
  searchVisible.value = false
  activePane()?.clearSearch()
  activePane()?.focus()
}

// ---- 分屏 ----
function enterSplit(mode: SplitMode) {
  if (isMobile.value) {
    return
  }
  splitDir.value = mode
  if (!splitMode.value) {
    splitMode.value = true
    // 初始把已打开的标签（活动标签优先）放入分屏
    const ids = tabs.value.map((t) => t.id)
    const first = ids.includes(active.value) ? [active.value] : []
    splitIds.value = [...first, ...ids.filter((x) => x !== active.value)].slice(0, MAX_SPLIT)
    if (splitIds.value.length < 2) {
      ElMessage.info('再打开一个连接即可自动加入分屏，也可点击其他标签加入')
    }
  }
}

function exitSplit() {
  splitMode.value = false
  splitIds.value = []
  api.del('/api/webterminal/splitlayout').catch(() => undefined)
}

function joinSplit(id: string) {
  if (splitIds.value.includes(id)) {
    return
  }
  if (splitIds.value.length < MAX_SPLIT) {
    splitIds.value.push(id)
  } else {
    ElMessage.info(`分屏最多 ${MAX_SPLIT} 个窗格，已替换当前窗格`)
    const i = splitIds.value.indexOf(active.value)
    splitIds.value.splice(i < 0 ? splitIds.value.length - 1 : i, 1, id)
  }
}

function leaveSplit(id: string) {
  if (splitIds.value.length <= 1) {
    return
  }
  splitIds.value = splitIds.value.filter((x) => x !== id)
  if (active.value === id) {
    active.value = splitIds.value[0]
  }
}

// 分屏布局变化时防抖保存到后端，用于刷新页面后恢复
let layoutTimer: ReturnType<typeof setTimeout> | undefined
let restoring = true

function buildLayout(): SplitLayout | null {
  const items = splitIds.value
    .map((id) => tabs.value.find((t) => t.id === id))
    .filter((t): t is TermTab => !!t && !!t.sessionId)
    .map((t) => ({ sessionId: t.sessionId, title: t.title }))
  if (!splitMode.value || !items.length) {
    return null
  }
  return { version: 1, mode: splitDir.value, items, active: activeTab.value?.sessionId ?? '' }
}

watch(
  () => JSON.stringify(buildLayout()),
  (json) => {
    if (restoring || isMobile.value || json === 'null') {
      return
    }
    if (layoutTimer) {
      clearTimeout(layoutTimer)
    }
    layoutTimer = setTimeout(() => {
      api.put('/api/webterminal/splitlayout', JSON.parse(json)).catch(() => undefined)
    }, 600)
  },
)

// 切到窄屏时不再分屏（不删除已保存的布局）
watch(isMobile, (m) => {
  if (m) {
    splitMode.value = false
  }
})

// ---- 标签持久化与恢复 ----
watch(
  () => tabs.value.map((t) => ({ connKey: t.connKey, connName: t.connName, type: t.type, sessionId: t.sessionId, title: t.title, remark: t.remark })),
  (list) => {
    if (restoring) {
      return
    }
    localStorage.setItem(TABS_KEY, JSON.stringify(list.filter((t) => t.sessionId)))
  },
  { deep: true },
)

// restoreTabs 页面刷新后，把仍存活的会话重新附着为标签，并恢复分屏布局。
async function restoreTabs() {
  try {
    const saved = JSON.parse(localStorage.getItem(TABS_KEY) ?? '[]') as Pick<TermTab, 'connKey' | 'connName' | 'type' | 'sessionId' | 'title' | 'remark'>[]
    if (!saved.length) {
      return
    }
    const live = (await api.get<SessionInfo[] | null>('/api/webterminal/sessions')) ?? []
    const alive = new Set(live.map((s) => s.id))
    const restored: TermTab[] = []
    for (const s of saved) {
      if (s.sessionId && alive.has(s.sessionId)) {
        const info = live.find((x) => x.id === s.sessionId)
        restored.push(addTab({ ...s, remark: info?.remark ?? s.remark }))
      }
    }
    if (!restored.length) {
      return
    }
    // 窄屏恢复后停留在连接列表，由用户点击返回终端
    mobileView.value = isMobile.value ? 'list' : 'term'
    active.value = restored[0].id
    if (!isMobile.value) {
      const layout = await api.get<SplitLayout | null>('/api/webterminal/splitlayout').catch(() => null)
      if (layout && layout.items?.length) {
        const ids = layout.items.map((it) => restored.find((t) => t.sessionId === it.sessionId)?.id).filter((x): x is string => !!x)
        if (ids.length >= 2) {
          splitDir.value = layout.mode
          splitIds.value = ids.slice(0, MAX_SPLIT)
          splitMode.value = true
          const act = restored.find((t) => t.sessionId === layout.active)
          if (act) {
            active.value = act.id
          }
        }
      }
    }
  } catch {
    // 恢复失败不影响正常使用
  }
}

// 全屏时按 Esc 之外的方式退出；窗口尺寸变化后重新适配
function onResize() {
  activePane()?.refit()
}

onMounted(async () => {
  window.addEventListener('resize', onResize)
  await Promise.all([loadConns(), loadSettings(), loadShortcuts()])
  await restoreTabs()
  restoring = false
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (layoutTimer) {
    clearTimeout(layoutTimer)
  }
})
</script>

<style scoped>
.terminal-page {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}
.side {
  width: 100%;
}
.workspace {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  padding: 8px;
  background: var(--el-bg-color);
}
.back-banner {
  width: 100%;
  min-height: 44px;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border: 1px solid var(--el-color-primary-light-5);
  border-radius: var(--g-radius);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 14px;
  cursor: pointer;
}
.back-banner svg:last-child {
  margin-left: auto;
}
.ws-head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.ws-tools {
  padding: 6px 0;
}
.panes {
  position: relative;
  flex: 1;
  min-height: 0;
  margin-top: 6px;
}
.panes.single .cell {
  position: absolute;
  inset: 0;
}
.panes.split {
  display: grid;
  gap: 6px;
}
.panes.split-h {
  grid-auto-flow: column;
  grid-auto-columns: minmax(0, 1fr);
}
.panes.split-v {
  grid-auto-flow: row;
  grid-auto-rows: minmax(0, 1fr);
}
.panes.split-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-auto-rows: minmax(0, 1fr);
}
.cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  border-radius: 8px;
  overflow: hidden;
}
.split .cell {
  border: 1px solid var(--g-border);
}
.split .cell.focus {
  border-color: var(--el-color-primary);
}
.cell-head {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 4px 0 10px;
  font-size: 12px;
  background: var(--el-fill-color-light);
  flex-shrink: 0;
}
.cell-head .ellipsis {
  flex: 1;
}
.cell-body {
  flex: 1;
  min-height: 0;
}
.dot {
  width: 7px;
  height: 7px;
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
.search-bar {
  position: absolute;
  top: 6px;
  right: 18px;
  z-index: 6;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 6px;
  border: 1px solid var(--g-border);
  border-radius: 8px;
  background: var(--el-bg-color-overlay);
  box-shadow: var(--el-box-shadow-light);
}
.search-bar .el-input {
  width: 180px;
}
.sb {
  padding: 6px;
  min-width: 28px;
}
.placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-secondary);
}

/* ---- 桌面端：左侧连接列表 + 右侧工作区 ---- */
@media (min-width: 768px) {
  .terminal-root {
    display: grid;
    grid-template-columns: 300px minmax(0, 1fr);
    grid-template-rows: auto 1fr;
    column-gap: 12px;
    height: calc(100dvh - 130px);
    min-height: 420px;
  }
  .terminal-root > :first-child {
    grid-column: 1 / -1;
  }
  .terminal-page {
    min-height: 0;
    overflow: auto;
  }
  .side {
    width: 300px;
  }
  .workspace {
    min-height: 0;
  }
  .workspace.full {
    position: fixed;
    inset: 0;
    z-index: 1500;
    border-radius: 0;
    height: auto;
  }
}

/* ---- 移动端：终端占满可视区域 ---- */
@media (max-width: 767.98px) {
  .workspace.m-full {
    position: fixed;
    left: 0;
    right: 0;
    top: 0;
    height: 100dvh;
    z-index: 1500;
    margin: 0;
    padding: env(safe-area-inset-top) 0 0;
    border: none;
    border-radius: 0;
    overscroll-behavior: contain;
  }
  .workspace.empty {
    display: none;
  }
  .ws-head {
    padding: 4px 4px 0;
  }
  .panes {
    margin-top: 4px;
  }
  .search-bar {
    right: 8px;
  }
  .search-bar .el-input {
    width: 140px;
  }
  .sb {
    min-width: 40px;
    height: 40px;
  }
  .back {
    min-width: 40px;
    height: 40px;
    padding: 0;
  }
}
</style>
