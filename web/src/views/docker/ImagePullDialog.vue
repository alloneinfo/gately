<template>
  <el-dialog :model-value="modelValue" :title="t('docker.image.pull.title')" width="560px" :fullscreen="isMobile" append-to-body @update:model-value="(v: boolean) => emit('update:modelValue', v)" @open="onOpen">
    <el-form :label-position="labelPosition" label-width="80px" @submit.prevent>
      <el-form-item :label="t('docker.image.pull.image')" required>
        <el-input v-model="ref" :placeholder="t('docker.image.pull.imagePlaceholder')" class="mono" @keyup.enter="pull" />
      </el-form-item>
      <el-form-item :label="t('common.search')">
        <div class="full">
          <el-input v-model="term" :placeholder="t('docker.image.pull.searchPlaceholder')" :prefix-icon="Search" clearable @keyup.enter="search">
            <template #append><el-button :loading="searching" @click="search">{{ t('common.search') }}</el-button></template>
          </el-input>
          <div class="results">
            <div v-for="r in results" :key="r.name" class="res" @click="pick(r.name)">
              <div class="res-top">
                <b class="mono">{{ r.name }}</b>
                <el-tag v-if="r.official" size="small" type="success">{{ t('docker.image.pull.official') }}</el-tag>
                <span class="stars"><Star :size="12" />{{ r.stars }}</span>
              </div>
              <div v-if="r.description" class="res-desc">{{ r.description }}</div>
            </div>
          </div>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :icon="CloudDownload" :loading="busy" @click="pull">{{ t('docker.image.pull.pull') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref as vref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { CloudDownload, Search, Star } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import { trackTask } from './tasks'
import type { ImageSearchItem } from './types'

const { t } = useI18n()

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()
const { isMobile, labelPosition } = useFormLayout()

const ref = vref('')
const term = vref('')
const results = vref<ImageSearchItem[]>([])
const searching = vref(false)
const busy = vref(false)

function onOpen() {
  ref.value = ''
  term.value = ''
  results.value = []
}

function pick(name: string) {
  ref.value = name.includes(':') ? name : `${name}:latest`
}

async function search() {
  if (!term.value.trim()) {
    return
  }
  searching.value = true
  try {
    results.value = (await api.post<ImageSearchItem[]>('/api/docker/images/search', { term: term.value.trim(), limit: 25 })) ?? []
    if (!results.value.length) {
      ElMessage.info(t('docker.image.pull.noResults'))
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    searching.value = false
  }
}

async function pull() {
  const r = ref.value.trim()
  if (!r) {
    ElMessage.warning(t('docker.image.pull.imageRequired'))
    return
  }
  busy.value = true
  try {
    const res = await api.post<{ task: string }>('/api/docker/images/pull', { ref: r })
    emit('update:modelValue', false)
    trackTask(res.task, t('docker.image.pull.task', { ref: r }), () => emit('done'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.full {
  width: 100%;
}
.results {
  margin-top: 8px;
  max-height: 260px;
  overflow-y: auto;
}
.res {
  padding: 8px 10px;
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius-sm);
  margin-bottom: 6px;
  cursor: pointer;
  min-height: 44px;
}
.res:hover {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-fill-color-lighter);
}
.res-top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
}
.stars {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.res-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  word-break: break-all;
}
@media (max-width: 767.98px) {
  .results {
    max-height: none;
  }
  :deep(input) {
    font-size: 16px;
  }
}
</style>
