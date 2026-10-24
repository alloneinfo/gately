<template>
  <!-- 编辑 "Key: Value" 列表：逐行模式提供常用头和取值建议，也可手输；文本模式便于批量粘贴。两种模式都会实时校验 -->
  <div class="hdr-editor">
    <template v-if="!textMode">
      <div v-for="(row, i) in rows" :key="i" class="hdr-item">
        <div class="hdr-row">
          <el-autocomplete
            :ref="(el: any) => (keyRefs[i] = el)"
            v-model="row.key"
            :fetch-suggestions="(q: string, cb: (s: any[]) => void) => cb(keySuggestions(q))"
            :placeholder="t('headerEditor.keyPlaceholder')"
            popper-class="hdr-popper"
            class="mono hdr-key"
            @update:model-value="splitPasted(row)"
            @select="(item: Record<string, any>) => onKeySelect(row, item)"
          >
            <template #default="{ item }">
              <div class="sug">
                <span class="mono">{{ item.value }}</span>
                <span class="sug-desc">{{ item.desc }}</span>
              </div>
            </template>
          </el-autocomplete>
          <span class="hdr-colon">:</span>
          <el-autocomplete
            v-model="row.value"
            :fetch-suggestions="(q: string, cb: (s: any[]) => void) => cb(valueSuggestions(row, q))"
            :placeholder="t('headerEditor.valuePlaceholder')"
            popper-class="hdr-popper"
            class="mono hdr-value"
          >
            <template #default="{ item }">
              <span v-if="item.value" class="mono">{{ item.value }}</span>
              <span v-else class="sug-desc">{{ t('headerEditor.removeHeader') }}</span>
            </template>
          </el-autocomplete>
          <el-button text type="danger" :icon="Trash2" @click="rows.splice(i, 1)" />
        </div>
        <div v-for="(it, j) in rowIssues[i]" :key="j" :class="['hdr-issue', `is-${it.level}`]">{{ issueText(it) }}</div>
      </div>
    </template>
    <template v-else>
      <el-input v-model="text" type="textarea" :autosize="{ minRows: 3, maxRows: 12 }" class="mono" :placeholder="placeholder" />
      <div v-for="(it, j) in textIssues" :key="j" :class="['hdr-issue', `is-${it.level}`]">
        {{ t('headerEditor.lineN', { n: it.line }) }}{{ issueText(it) }}
      </div>
    </template>
    <div class="hdr-actions">
      <el-button v-if="!textMode" size="small" :icon="Plus" @click="addRow()">{{ t('headerEditor.add') }}</el-button>
      <el-button v-if="direction === 'response'" size="small" :icon="ShieldCheck" @click="addSecurity">
        {{ t('headerEditor.addSecurity') }}
      </el-button>
      <el-button size="small" text type="primary" @click="toggleMode">
        {{ textMode ? t('headerEditor.rowMode') : t('headerEditor.textMode') }}
      </el-button>
    </div>
    <div v-if="showHelp && placeholder" class="form-help">{{ placeholder }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Plus, ShieldCheck, Trash2 } from 'lucide-vue-next'
import {
  checkHeader,
  findPreset,
  formatHeaderRow,
  parseHeaderLine,
  presetsFor,
  securityBundle,
  type HeaderDirection,
  type HeaderRow,
  type Issue,
} from '../headerPresets'
import { useListText, splitLines } from '../composables/useListText'

const props = withDefaults(
  defineProps<{
    modelValue?: string[] | null
    direction: HeaderDirection
    // 底部说明，同时作为文本模式的占位提示
    placeholder?: string
    // 是否在底部显示说明；嵌在分路径后端等已有说明的位置时可关闭
    showHelp?: boolean
  }>(),
  { modelValue: undefined, placeholder: '', showHelp: true },
)
const emit = defineEmits<{ 'update:modelValue': [list: string[]] }>()
const { t } = useI18n()

const rows = ref<HeaderRow[]>([])
const textMode = ref(false)
const keyRefs: any[] = []

// 两侧都为空的行只在界面上保留，不写回数据
const serialize = (list: HeaderRow[]) => list.filter((r) => r.key.trim() || r.value.trim()).map(formatHeaderRow)
const sameList = (a: string[], b: string[]) => a.length === b.length && a.every((s, i) => s === b[i])

// 数据被外部替换（打开另一个站点、文本模式编辑等）时才重建行，避免打断正在编辑的空行
watch(
  () => props.modelValue,
  (list) => {
    const items = list ?? []
    if (!sameList(items, serialize(rows.value))) {
      rows.value = items.map(parseHeaderLine)
    }
  },
  { immediate: true, deep: true },
)
function emitRows() {
  const out = serialize(rows.value)
  if (!sameList(out, props.modelValue ?? [])) {
    emit('update:modelValue', out)
  }
}
// 文本模式下由文本框写回数据，行只跟随显示；否则缺冒号的行会被立即改写成"删除该头"，错误提示也随之消失
watch(
  rows,
  () => {
    if (!textMode.value) {
      emitRows()
    }
  },
  { deep: true },
)
// 切回逐行编辑时按行规范化一次
watch(textMode, (on) => {
  if (!on) {
    emitRows()
  }
})
// 文本模式里有缺冒号的行时不切回逐行模式：逐行模式会把它当成"删除该头"，悄悄改变配置含义
function toggleMode() {
  if (textMode.value) {
    const bad = textIssues.value.find((it) => it.msg === 'noColon')
    if (bad) {
      ElMessage.warning(t('headerEditor.lineN', { n: bad.line }) + issueText(bad))
      return
    }
  }
  textMode.value = !textMode.value
}

// 文本模式：保留原始文本，按行解析写回
const text = useListText(
  () => props.modelValue,
  (list) => emit('update:modelValue', list),
  { split: splitLines },
)

const rowIssues = computed(() => rows.value.map((_, i) => checkHeader(props.direction, rows.value, i)))
// 文本模式只列出错误和警告，行号与文本框中的实际行号一致（空行也计数）
const textIssues = computed(() => {
  const lines: { no: number; raw: string }[] = []
  text.value.split('\n').forEach((raw, i) => {
    if (raw.trim()) {
      lines.push({ no: i + 1, raw: raw.trim() })
    }
  })
  const parsed = lines.map((l) => parseHeaderLine(l.raw))
  const out: (Issue & { line: number })[] = []
  lines.forEach((l, i) => {
    if (!l.raw.includes(':')) {
      out.push({ level: 'error', msg: 'noColon', line: l.no })
      return
    }
    for (const it of checkHeader(props.direction, parsed, i)) {
      if (it.level === 'info') {
        continue
      }
      // duplicate 的参数是行序号，换算成文本框中的行号
      const params = it.msg === 'duplicate' && it.params ? { ...it.params, line: lines[Number(it.params.line) - 1]?.no ?? it.params.line } : it.params
      out.push({ ...it, params, line: l.no })
    }
  })
  return out
})

function issueText(it: Issue) {
  const params: Record<string, string | number> = { ...it.params }
  if (it.msg === 'host') {
    params.option = t('web.optTargetHost')
  } else if (it.msg === 'clientIp') {
    params.option = t('web.optForwarded')
  }
  return t(`headerEditor.issue.${it.msg}`, params)
}

// 名称建议：输入为空或已是某个预设名时列出全部，便于切换；否则按包含关系过滤
function keySuggestions(q: string) {
  const query = q.trim().toLowerCase()
  const all = presetsFor(props.direction)
  const exact = all.some((p) => p.key.toLowerCase() === query)
  return all
    .filter((p) => !query || exact || p.key.toLowerCase().includes(query))
    .map((p) => ({ value: p.key, desc: t(`headerEditor.desc.${p.id}`) }))
}

function valueSuggestions(row: HeaderRow, q: string) {
  const preset = findPreset(props.direction, row.key)
  if (!preset) {
    return []
  }
  const query = q.trim().toLowerCase()
  const exact = preset.values.some((v) => v.toLowerCase() === query)
  return preset.values.filter((v) => !query || exact || v.toLowerCase().includes(query)).map((v) => ({ value: v }))
}

// 选中预设名时，若值为空则填入第一个常用值
function onKeySelect(row: HeaderRow, item: Record<string, any>) {
  const preset = findPreset(props.direction, item.value)
  if (preset && !row.value.trim() && preset.values[0]) {
    row.value = preset.values[0]
  }
}

// 名称框里粘贴了整行 "Key: Value" 时自动拆开
function splitPasted(row: HeaderRow) {
  if (row.key.includes(':') && !row.value.trim()) {
    const p = parseHeaderLine(row.key)
    row.key = p.key
    row.value = p.value
  }
}

function addRow(key = '', value = '') {
  rows.value.push({ key, value })
  if (!key) {
    // 聚焦新行的名称框，顺带弹出常用头列表
    nextTick(() => keyRefs[rows.value.length - 1]?.focus?.())
  }
}

// 补齐尚未配置的常用安全头
function addSecurity() {
  const has = new Set(rows.value.map((r) => r.key.trim().toLowerCase()))
  // 先去掉末尾的空行，避免安全头插在空行之后
  while (rows.value.length && !rows.value[rows.value.length - 1].key.trim() && !rows.value[rows.value.length - 1].value.trim()) {
    rows.value.pop()
  }
  for (const [k, v] of securityBundle) {
    if (!has.has(k.toLowerCase())) {
      rows.value.push({ key: k, value: v })
    }
  }
}
</script>

<style scoped>
.hdr-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.hdr-row {
  display: flex;
  gap: 6px;
  align-items: center;
}
/* el-autocomplete 根节点拿不到 scoped 属性，需从父元素用 :deep 选中 */
.hdr-row :deep(.hdr-key) {
  width: 240px;
  flex: none;
}
.hdr-row :deep(.hdr-value) {
  flex: 1;
  min-width: 0;
}
.hdr-colon {
  color: var(--el-text-color-secondary);
}
.hdr-issue {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.5;
}
.hdr-issue.is-error {
  color: var(--el-color-danger);
}
.hdr-issue.is-warning {
  color: var(--el-color-warning);
}
.hdr-issue.is-info {
  color: var(--el-text-color-secondary);
}
.hdr-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.hdr-actions .el-button + .el-button {
  margin-left: 0;
}
.sug {
  display: flex;
  flex-direction: column;
  line-height: 1.4;
  padding: 4px 0;
}
.sug-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: normal;
}
@media (max-width: 640px) {
  .hdr-row {
    flex-wrap: wrap;
  }
  .hdr-row :deep(.hdr-key) {
    flex: 1;
    width: auto;
    min-width: 0;
  }
  .hdr-colon {
    display: none;
  }
  /* 值框换到第二行，删除按钮留在名称框旁边 */
  .hdr-row :deep(.hdr-value) {
    order: 1;
    flex-basis: 100%;
  }
}
</style>

<style>
/* 建议列表挂在 body 下，带说明的选项需要放开 Element Plus 默认的固定行高 */
.hdr-popper .el-autocomplete-suggestion li {
  line-height: 1.4;
  padding-top: 2px;
  padding-bottom: 2px;
  white-space: normal;
}
.hdr-popper .el-autocomplete-suggestion li > span {
  display: block;
  padding: 6px 0;
}
</style>
