<template>
  <!-- Content-Type 输入：可从常用类型中选择，也可手输，实时校验格式 -->
  <div class="ct-input">
    <el-autocomplete
      v-model="model"
      :fetch-suggestions="(q: string, cb: (s: any[]) => void) => cb(suggestions(q))"
      placeholder="text/plain; charset=utf-8"
      popper-class="hdr-popper"
      class="mono ct-field"
      clearable
    >
      <template #default="{ item }">
        <span class="mono">{{ item.value }}</span>
      </template>
    </el-autocomplete>
    <div v-for="(it, j) in issues" :key="j" :class="['hdr-issue', `is-${it.level}`]">
      {{ t(`headerEditor.issue.${it.msg}`, it.params ?? {}) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { checkContentType, contentTypePresets } from '../headerPresets'

const model = defineModel<string>({ default: '' })
const { t } = useI18n()

const issues = computed(() => checkContentType(model.value ?? ''))

// 输入为空或已是某个常用类型时列出全部，否则按包含关系过滤
function suggestions(q: string) {
  const query = q.trim().toLowerCase()
  const exact = contentTypePresets.some((v) => v === query)
  return contentTypePresets.filter((v) => !query || exact || v.includes(query)).map((v) => ({ value: v }))
}
</script>

<style scoped>
.ct-input {
  width: 100%;
}
.ct-input :deep(.ct-field) {
  width: 100%;
  max-width: 360px;
}
.hdr-issue {
  margin-top: 4px;
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
</style>
