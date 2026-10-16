<template>
  <EntityCard :off="c.state !== 'running'">
    <template #lead>
      <el-checkbox :model-value="selected" class="sel" @update:model-value="(v: unknown) => emit('select', c.id, !!v)" />
    </template>
    <template #title>
      <span>{{ c.name }}</span>
      <el-tag v-if="c.project && !hideProject" size="small" type="info">{{ c.project }}</el-tag>
    </template>
    <template #sub>{{ c.status }}</template>
    <template #status>
      <StatusBadge v-bind="containerBadge(c.state)" />
    </template>
    <Field label="镜像" mono>
      {{ c.image }}
      <div v-if="c.update" class="upd">
        <StatusBadge v-if="c.update.error" type="error" text="检查失败" :title="c.update.error" />
        <StatusBadge v-else-if="c.update.hasUpdate" type="warn" text="有新版本" />
        <StatusBadge v-else type="ok" text="已是最新" />
      </div>
    </Field>
    <Field label="端口" mono>
      <div v-for="p in c.ports ?? []" :key="p">
        <a v-if="isTcpPort(p)" href="javascript:void(0)" class="port" @click.prevent="openPort(p)">{{ p }}</a>
        <span v-else>{{ p }}</span>
      </div>
      <span v-if="!(c.ports ?? []).length" class="muted">-</span>
    </Field>
    <template #actions>
      <el-button v-if="c.state !== 'running'" text type="primary" :icon="Play" class="btn" :loading="c.busy" @click="emit('command', c, 'start')">启动</el-button>
      <el-button v-else text type="danger" :icon="Square" class="btn" @click="emit('command', c, 'stop')">停止</el-button>
      <el-button v-if="c.state === 'running'" text :icon="RotateCw" class="btn" @click="emit('command', c, 'restart')">重启</el-button>
      <el-button text :icon="PanelRight" class="btn" @click="emit('command', c, 'detail')">详情</el-button>
      <el-dropdown trigger="click" @command="(cmd: string) => emit('command', c, cmd)">
        <el-button text :icon="Ellipsis" class="btn">更多</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="logs" :icon="ScrollText">日志</el-dropdown-item>
            <el-dropdown-item v-if="c.state === 'running'" command="term" :icon="SquareTerminal">终端</el-dropdown-item>
            <el-dropdown-item command="files" :icon="FolderOpen">文件</el-dropdown-item>
            <el-dropdown-item v-if="c.state === 'running'" command="pause" :icon="Pause" divided>暂停</el-dropdown-item>
            <el-dropdown-item v-if="c.state === 'paused'" command="unpause" :icon="Play" divided>恢复</el-dropdown-item>
            <el-dropdown-item v-if="c.state === 'running' || c.state === 'paused'" command="kill" :icon="Zap">强制停止 (kill)</el-dropdown-item>
            <el-dropdown-item command="edit" :icon="Pencil" divided>编辑</el-dropdown-item>
            <el-dropdown-item command="rename" :icon="TextCursorInput">重命名</el-dropdown-item>
            <el-dropdown-item command="commit" :icon="GitCommitHorizontal">提交为镜像</el-dropdown-item>
            <el-dropdown-item command="check" :icon="SearchCheck" divided>检查更新</el-dropdown-item>
            <el-dropdown-item v-if="c.update?.hasUpdate" command="upgrade" :icon="ArrowUpCircle">升级</el-dropdown-item>
            <el-dropdown-item command="delete" :icon="Trash2" divided>删除</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </template>
  </EntityCard>
</template>

<script setup lang="ts">
import { ArrowUpCircle, Ellipsis, FolderOpen, GitCommitHorizontal, PanelRight, Pause, Pencil, Play, RotateCw, ScrollText, SearchCheck, Square, SquareTerminal, TextCursorInput, Trash2, Zap } from 'lucide-vue-next'
import EntityCard from '../../components/EntityCard.vue'
import Field from '../../components/Field.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { containerBadge, isTcpPort, openPort } from './format'
import type { ContainerView } from './types'

defineProps<{ c: ContainerView; selected: boolean; hideProject?: boolean }>()
const emit = defineEmits<{ command: [c: ContainerView, cmd: string]; select: [id: string, v: boolean] }>()
</script>

<style scoped>
.sel {
  height: auto;
}
.upd {
  margin-top: 4px;
}
.port {
  color: var(--el-color-primary);
  text-decoration: none;
}
.port:hover {
  text-decoration: underline;
}
@media (max-width: 767.98px) {
  .btn {
    min-height: 40px;
    padding: 0 8px;
  }
  .sel {
    min-width: 28px;
    min-height: 40px;
  }
}
</style>
