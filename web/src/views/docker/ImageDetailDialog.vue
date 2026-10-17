<template>
  <el-dialog :model-value="modelValue" :title="t('docker.image.detail.title')" width="820px" :fullscreen="isMobile" top="5vh" append-to-body @update:model-value="(v: boolean) => emit('update:modelValue', v)" @open="load">
    <div v-loading="loading" class="detail">
      <template v-if="d">
        <div class="grid">
          <Field :label="t('docker.image.detail.tags')" wide mono>{{ d.tags?.join(', ') || t('docker.image.detail.noneParen') }}</Field>
          <Field label="ID" wide mono>{{ d.id }}</Field>
          <Field :label="t('docker.image.detail.size')">{{ formatBytes(d.size) }}</Field>
          <Field :label="t('docker.image.detail.created')">{{ formatTime(d.created) }}</Field>
          <Field :label="t('docker.image.detail.osArch')">{{ d.os }} / {{ d.arch }}</Field>
          <Field :label="t('docker.image.detail.author')">{{ d.author || '-' }}</Field>
          <Field label="Entrypoint" wide mono>{{ d.entrypoint?.join(' ') || '-' }}</Field>
          <Field label="Cmd" wide mono>{{ d.cmd?.join(' ') || '-' }}</Field>
          <Field :label="t('docker.image.detail.workdir')" mono>{{ d.workdir || '-' }}</Field>
          <Field :label="t('docker.image.detail.user')" mono>{{ d.user || '-' }}</Field>
          <Field :label="t('docker.image.detail.exposedPorts')" wide mono>{{ d.exposedPorts?.join(', ') || '-' }}</Field>
        </div>
        <el-divider>{{ t('docker.image.detail.env') }}</el-divider>
        <div v-for="(e, i) in d.env ?? []" :key="i" class="line mono">{{ e }}</div>
        <div v-if="!d.env?.length" class="muted">{{ t('common.none') }}</div>
        <el-divider>{{ t('docker.image.detail.tags') }}</el-divider>
        <div v-for="(v, k) in d.labels ?? {}" :key="k" class="line mono">{{ k }}={{ v }}</div>
        <div v-if="!Object.keys(d.labels ?? {}).length" class="muted">{{ t('common.none') }}</div>
        <el-divider>{{ t('docker.image.detail.layers', { count: d.layers?.length ?? 0 }) }}</el-divider>
        <div v-for="(l, i) in d.layers ?? []" :key="i" class="layer">
          <div class="l-meta">{{ formatBytes(l.size) }} · {{ formatTime(l.created) }}</div>
          <div class="l-cmd mono">{{ l.createdBy }}</div>
        </div>
      </template>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { api } from '../../api'
import Field from '../../components/Field.vue'
import { useIsMobile } from '../../composables/useIsMobile'
import { formatBytes, formatTime } from './format'
import type { ImageDetail } from './types'

const { t } = useI18n()

const props = defineProps<{ modelValue: boolean; imageId?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()
const isMobile = useIsMobile()
const d = ref<ImageDetail>()
const loading = ref(false)

async function load() {
  d.value = undefined
  if (!props.imageId) {
    return
  }
  loading.value = true
  try {
    d.value = await api.get<ImageDetail>(`/api/docker/images/${encodeURIComponent(props.imageId)}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
    emit('update:modelValue', false)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.detail {
  min-height: 120px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px 20px;
}
.line {
  font-size: 12px;
  line-height: 1.7;
  word-break: break-all;
}
.layer {
  padding: 6px 0;
  border-bottom: 1px dashed var(--g-border);
}
.l-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.l-cmd {
  font-size: 12px;
  word-break: break-all;
  line-height: 1.5;
}
@media (max-width: 767.98px) {
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
