<template>
  <div class="g-card conn-card">
    <div class="g-card-head">
      <div class="g-card-title"><Cable :size="16" />连接 <span class="meta">{{ conns.length }} 个</span></div>
      <div class="head-ops">
        <el-button v-if="conns.length > 1" size="small" :type="sortMode ? 'primary' : 'default'" :icon="ArrowUpDown" @click="sortMode = !sortMode">
          {{ sortMode ? '完成' : '排序' }}
        </el-button>
      </div>
    </div>
    <el-empty v-if="!conns.length" description="暂无连接，点击右上角「添加连接」" :image-size="60" />
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
            <el-tag size="small" effect="plain">{{ typeLabel[c.type] }}</el-tag>
            <el-tag v-if="!c.enable" size="small" type="info">已停用</el-tag>
            <span v-if="openCount(c.key)" class="open-dot" :title="`已打开 ${openCount(c.key)} 个会话`">{{ openCount(c.key) }}</span>
          </div>
          <div class="conn-addr mono">{{ connAddr(c) }}</div>
          <div v-if="c.remark" class="conn-remark">{{ c.remark }}</div>
        </div>
        <div class="conn-ops" @click.stop>
          <template v-if="sortMode">
            <GripVertical v-if="!isMobile" :size="16" class="grip" />
            <el-button class="op" text :icon="ChevronUp" :disabled="i === 0" aria-label="上移" @click="move(i, -1)" />
            <el-button class="op" text :icon="ChevronDown" :disabled="i === conns.length - 1" aria-label="下移" @click="move(i, 1)" />
          </template>
          <template v-else>
            <el-button v-if="!isMobile" size="small" type="primary" plain :icon="Play" :disabled="!c.enable" @click="$emit('connect', c)">连接</el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => onCommand(c, cmd)">
              <el-button class="op" text :icon="Ellipsis" aria-label="更多" />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit" :icon="Pencil">编辑</el-dropdown-item>
                  <el-dropdown-item command="duplicate" :icon="Copy">复制连接</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'telnet'" command="sftp" :icon="FolderOpen">文件管理</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'telnet'" command="quick" :icon="Star">快速访问目录</el-dropdown-item>
                  <el-dropdown-item v-if="c.type !== 'local'" command="test" :icon="Plug">测试连接</el-dropdown-item>
                  <el-dropdown-item v-if="c.type === 'ssh'" command="resetkey" :icon="KeyRound">重置主机指纹</el-dropdown-item>
                  <el-dropdown-item command="delete" :icon="Trash2" divided>删除</el-dropdown-item>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowUpDown, Cable, ChevronDown, ChevronUp, Copy, Ellipsis, FolderOpen, GripVertical, KeyRound, Network, Pencil, Play, Plug, Server, Star, Terminal as TerminalIcon, Trash2,
} from 'lucide-vue-next'
import { api } from '../../api'
import { MASK, connAddr, normalizeConn, typeLabel, type Conn } from './types'

const props = defineProps<{ conns: Conn[]; isMobile: boolean; openCounts: Record<string, number> }>()
const emit = defineEmits<{
  connect: [c: Conn]
  edit: [c: Conn | null]
  sftp: [c: Conn]
  quick: [c: Conn]
  reload: []
}>()

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
  const t = keys[i]
  keys[i] = keys[j]
  keys[j] = t
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
  copy.name = `${c.name} 副本`
  copy.ssh.hostKey = ''
  const clear = (v: string) => (/^\*+$/.test(v) || v === MASK ? '' : v)
  copy.ssh.password = clear(copy.ssh.password)
  copy.ssh.privateKey = clear(copy.ssh.privateKey)
  copy.ssh.passphrase = clear(copy.ssh.passphrase)
  copy.ssh.proxy.password = clear(copy.ssh.proxy.password)
  copy.telnet.password = clear(copy.telnet.password)
  copy.telnet.proxy.password = clear(copy.telnet.proxy.password)
  emit('edit', copy)
  ElMessage.info('已复制配置，密码、私钥等敏感信息需要重新填写')
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
      const loading = ElMessage({ message: '正在测试连接…', duration: 0, type: 'info' })
      try {
        const r = await api.post<{ message?: string } | null>(`/api/webterminal/connections/${encodeURIComponent(c.key)}/test`)
        ElMessage.success(r?.message || `${c.name} 连接成功`)
      } finally {
        loading.close()
      }
    } else if (cmd === 'resetkey') {
      await ElMessageBox.confirm('重置后，下次连接时会重新记录主机指纹。主机重装或更换密钥后需要执行此操作。', `重置 ${c.name} 的主机指纹`, { type: 'warning' })
      await api.del(`/api/webterminal/connections/${encodeURIComponent(c.key)}/hostkey`)
      ElMessage.success('已重置')
      emit('reload')
    } else if (cmd === 'delete') {
      await ElMessageBox.confirm(`删除连接「${c.name}」？`, '确认', { type: 'warning' })
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
