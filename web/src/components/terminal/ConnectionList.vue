<template>
  <div class="g-card conn-card">
    <div class="g-card-head">
      <div class="g-card-title"><Cable :size="16" />{{ t('termConn.list.title') }} <span class="meta">{{ t('termConn.list.count', { count: conns.length }) }}</span></div>
      <div class="head-ops">
        <el-button v-if="conns.length > 1" size="small" :type="sortMode ? 'primary' : 'default'" :icon="ArrowUpDown" @click="sortMode = !sortMode">
          {{ sortMode ? t('termConn.list.done') : t('termConn.list.sort') }}
        </el-button>
      </div>
    </div>
    <el-empty v-if="!conns.length" :description="t('termConn.list.empty')" :image-size="60" />
    <div class="list">
      <div
        v-for="(c, i) in conns"
        :key="c.key"
        class="conn"
        :class="{ disabled: !c.enable, dragging: dragKey === c.key, over: overKey === c.key && dragKey !== c.key }"
        :draggable="sortMode && !isMobile"
        @click="onTap(c)"
        @dblclick="onDblClick(c)"
        @dragstart="onDragStart(c, $event)"
        @dragover.prevent="overKey = c.key"
        @dragleave="overKey === c.key && (overKey = '')"
        @drop.prevent="onDrop(c)"
        @dragend="dragKey = ''; overKey = ''"
      >
        <div class="conn-icon" :class="c.type"><component :is="iconOf(c)" :size="18" /></div>
        <div class="conn-main">
          <div class="conn-name">
            <span class="ellipsis">{{ c.name }}</span>
            <el-tag size="small" effect="plain">{{ t(`termConn.type.${c.type}`) }}</el-tag>
            <el-tag v-if="!c.enable" size="small" type="info">{{ t('termConn.list.disabledTag') }}</el-tag>
            <span v-if="openCount(c.key)" class="open-dot" :title="t('termConn.list.openSessions', { count: openCount(c.key) })">{{ openCount(c.key) }}</span>
          </div>
          <div class="conn-addr mono">{{ connAddr(c) }}</div>
          <div v-if="c.remark" class="conn-remark">{{ c.remark }}</div>
        </div>
        <div class="conn-ops" @click.stop>
          <template v-if="sortMode">
            <GripVertical v-if="!isMobile" :size="16" class="grip" />
            <el-button class="op" text :icon="ChevronUp" :disabled="i === 0" :aria-label="t('termConn.list.moveUp')" @click="move(i, -1)" />
            <el-button class="op" text :icon="ChevronDown" :disabled="i === conns.length - 1" :aria-label="t('termConn.list.moveDown')" @click="move(i, 1)" />
          </template>
          <template v-else>
            <el-button v-if="!isMobile" size="small" type="primary" plain :icon="Play" :disabled="!c.enable" @click="$emit('connect', c)">{{ t('termConn.list.connect') }}</el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => onCommand(c, cmd)">
              <el-button class="op" text :icon="Ellipsis" :aria-label="t('termConn.list.more')" />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit" :icon="Pencil">{{ t('termConn.list.edit') }}</el-dropdown-item>
                  <el-dropdown-item command="duplicate" :icon="Copy">{{ t('termConn.list.duplicate') }}</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'telnet'" command="sftp" :icon="FolderOpen">{{ t('termConn.list.sftp') }}</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'telnet'" command="quick" :icon="Star">{{ t('termConn.list.quick') }}</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'local'" command="test" :icon="Plug">{{ t('termConn.list.test') }}</el-dropdown-item>
                  <el-dropdown-item v-if="c.type === 'ssh'" command="resetkey" :icon="KeyRound">{{ t('termConn.list.resetKey') }}</el-dropdown-item>
                  <el-dropdown-item command="delete" :icon="Trash2" divided>{{ t('common.delete') }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowUpDown, Cable, ChevronDown, ChevronUp, Copy, Ellipsis, FolderOpen, GripVertical, KeyRound, Network, Pencil, Play, Plug, Server, Star, Terminal as TerminalIcon, Trash2,
} from 'lucide-vue-next'
import { api } from '../../api'
import { MASK, connAddr, normalizeConn, type Conn } from './types'

const props = defineProps<{ conns: Conn[]; isMobile: boolean; openCounts: Record<string, number> }>()
const emit = defineEmits<{
  connect: [c: Conn]
  edit: [c: Conn | null]
  sftp: [c: Conn]
  quick: [c: Conn]
  reload: []
}>()

const { t } = useI18n()
const sortMode = ref(false)
const dragKey = ref('')
const overKey = ref('')

function iconOf(c: Conn) {
  if (c.type === 'local') {
    return TerminalIcon
  }
  return c.type === 'telnet' ? Network : Server
}

function openCount(key: string): number {
  return props.openCounts[key] ?? 0
}

// 点击卡片即连接（排序模式下不触发）
function onTap(c: Conn) {
  if (sortMode.value || !c.enable) {
    return
  }
  if (props.isMobile) {
    emit('connect', c)
  }
}

// 桌面端双击卡片连接
function onDblClick(c: Conn) {
  if (sortMode.value || !c.enable || props.isMobile) {
    return
  }
  emit('connect', c)
}

async function saveOrder(keys: string[]) {
  try {
    await api.put('/api/webterminal/connections-order', { keys })
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
  emit('reload')
}

function move(i: number, d: number) {
  const keys = props.conns.map((c) => c.key)
  const j = i + d
  if (j < 0 || j >= keys.length) {
    return
  }
  const tmp = keys[i]
  keys[i] = keys[j]
  keys[j] = tmp
  saveOrder(keys)
}

function onDragStart(c: Conn, e: DragEvent) {
  dragKey.value = c.key
  e.dataTransfer?.setData('text/plain', c.key)
}

function onDrop(target: Conn) {
  const from = dragKey.value
  dragKey.value = ''
  overKey.value = ''
  if (!from || from === target.key) {
    return
  }
  const keys = props.conns.map((c) => c.key).filter((k) => k !== from)
  keys.splice(keys.indexOf(target.key), 0, from)
  saveOrder(keys)
}

// 复制连接：脱敏的密文不能原样提交，需要清空后让用户重新填写。
function duplicate(c: Conn) {
  const copy = normalizeConn(JSON.parse(JSON.stringify(c)))
  copy.key = ''
  copy.name = t('termConn.list.copySuffix', { name: c.name })
  copy.ssh.hostKey = ''
  const clear = (v: string) => (/^\*+$/.test(v) || v === MASK ? '' : v)
  copy.ssh.password = clear(copy.ssh.password)
  copy.ssh.privateKey = clear(copy.ssh.privateKey)
  copy.ssh.passphrase = clear(copy.ssh.passphrase)
  copy.ssh.proxy.password = clear(copy.ssh.proxy.password)
  copy.telnet.password = clear(copy.telnet.password)
  copy.telnet.proxy.password = clear(copy.telnet.proxy.password)
  emit('edit', copy)
  ElMessage.info(t('termConn.list.copyHint'))
}

async function onCommand(c: Conn, cmd: string) {
  try {
    if (cmd === 'edit') {
      emit('edit', c)
    } else if (cmd === 'duplicate') {
      duplicate(c)
    } else if (cmd === 'sftp') {
      emit('sftp', c)
    } else if (cmd === 'quick') {
      emit('quick', c)
    } else if (cmd === 'test') {
      const loading = ElMessage({ message: t('termConn.list.testing'), duration: 0, type: 'info' })
      try {
        const r = await api.post<{ message?: string } | null>(`/api/webterminal/connections/${encodeURIComponent(c.key)}/test`)
        ElMessage.success(r?.message || t('termConn.list.testOk', { name: c.name }))
      } finally {
        loading.close()
      }
    } else if (cmd === 'resetkey') {
      await ElMessageBox.confirm(t('termConn.list.resetConfirm'), t('termConn.list.resetTitle', { name: c.name }), { type: 'warning' })
      await api.del(`/api/webterminal/connections/${encodeURIComponent(c.key)}/hostkey`)
      ElMessage.success(t('termConn.list.resetDone'))
      emit('reload')
    } else if (cmd === 'delete') {
      await ElMessageBox.confirm(t('termConn.list.deleteConfirm', { name: c.name }), t('termConn.list.confirmTitle'), { type: 'warning' })
      await api.del(`/api/webterminal/connections/${encodeURIComponent(c.key)}`)
      emit('reload')
    }
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error((e as Error).message)
    }
  }
}
</script>

<style scoped>
.conn-card {
  overflow: auto;
}
.head-ops {
  display: flex;
  gap: 6px;
}
.conn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--g-border);
  cursor: default;
}
.conn:last-child {
  border-bottom: none;
}
.conn.disabled {
  opacity: 0.55;
}
.conn.dragging {
  opacity: 0.4;
}
.conn.over {
  box-shadow: inset 0 2px 0 var(--el-color-primary);
}
.conn-icon {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  flex-shrink: 0;
}
.conn-icon.telnet {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
.conn-icon.local {
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
}
.conn-main {
  flex: 1;
  min-width: 0;
}
.conn-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}
.conn-addr {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.conn-remark {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.open-dot {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--el-color-success);
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}
.conn-ops {
  display: flex;
  gap: 2px;
  align-items: center;
}
.grip {
  color: var(--el-text-color-placeholder);
  cursor: grab;
}
.op {
  padding: 8px;
}
@media (max-width: 767.98px) {
  .conn {
    padding: 12px;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  .conn:active {
    background: var(--el-fill-color-light);
  }
  .op {
    min-width: 40px;
    height: 40px;
    padding: 0;
  }
}
</style>
