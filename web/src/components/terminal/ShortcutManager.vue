<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('terminal.shortcut.title')"
    width="680px"
    :fullscreen="isMobile"
    class="term-dialog"
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => $emit('update:modelValue', v)"
    @open="onOpen"
  >
    <div class="tip">{{ t('terminal.shortcut.tip') }}</div>
    <div v-for="(s, i) in rows" :key="i" class="row">
      <div class="cell name"><label>{{ t('terminal.shortcut.name') }}</label><el-input v-model="s.name" :placeholder="t('terminal.shortcut.namePlaceholder')" /></div>
      <div class="cell cmd"><label>{{ t('terminal.shortcut.command') }}</label><el-input v-model="s.command" class="mono" placeholder="df -h" /></div>
      <div class="cell enter"><label>{{ t('terminal.shortcut.enter') }}</label><el-switch v-model="s.sendEnter" /></div>
      <div class="cell ops">
        <el-button text :icon="ChevronUp" :disabled="i === 0" :aria-label="t('terminal.shortcut.moveUp')" @click="swap(i, -1)" />
        <el-button text :icon="ChevronDown" :disabled="i === rows.length - 1" :aria-label="t('terminal.shortcut.moveDown')" @click="swap(i, 1)" />
        <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="rows.splice(i, 1)" />
      </div>
    </div>
    <el-empty v-if="!rows.length" :description="t('terminal.shortcut.empty')" :image-size="60" />
    <el-button :icon="Plus" @click="rows.push({ name: '', command: '', sendEnter: true })">{{ t('terminal.shortcut.add') }}</el-button>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useIsMobile } from '../../composables/useIsMobile'
import type { Shortcut } from './types'

const props = defineProps<{ modelValue: boolean; shortcuts: Shortcut[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [list: Shortcut[]] }>()

const { t } = useI18n()
const isMobile = useIsMobile()
const rows = ref<Shortcut[]>([])
const saving = ref(false)

function onOpen() {
  rows.value = props.shortcuts.map((s) => ({ ...s }))
}

function swap(i: number, d: number) {
  const j = i + d
  const tmp = rows.value[i]
  rows.value[i] = rows.value[j]
  rows.value[j] = tmp
}

async function save() {
  const list = rows.value
    .map((s) => ({ name: s.name.trim(), command: s.command, sendEnter: s.sendEnter }))
    .filter((s) => s.command.trim())
  for (const s of list) {
    if (!s.name) {
      s.name = s.command.length > 20 ? s.command.slice(0, 20) + '…' : s.command
    }
  }
  saving.value = true
  try {
    await api.put('/api/webterminal/globalshortcuts', list)
    emit('saved', list)
    emit('update:modelValue', false)
    ElMessage.success(t('common.saved'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.tip {
  margin-bottom: 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.row {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr) 60px 120px;
  gap: 8px;
  align-items: end;
  padding: 8px 0;
  border-bottom: 1px solid var(--g-border);
}
.cell label {
  display: block;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cell.ops {
  display: flex;
  justify-content: flex-end;
}
@media (max-width: 767.98px) {
  .row {
    grid-template-columns: 1fr auto;
    padding: 12px 0;
  }
  .cell.cmd {
    grid-column: 1 / -1;
    grid-row: 2;
  }
  .cell.enter {
    grid-row: 1;
    grid-column: 2;
  }
  .cell.ops {
    grid-column: 1 / -1;
    grid-row: 3;
  }
}
</style>
