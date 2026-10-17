<template>
  <el-dialog :model-value="modelValue" :title="t('docker.image.build.title')" width="680px" :fullscreen="isMobile" top="5vh" append-to-body @update:model-value="(v: boolean) => emit('update:modelValue', v)" @open="onOpen">
    <el-form :label-position="labelPosition" label-width="90px" @submit.prevent>
      <el-form-item :label="t('docker.image.build.tag')" required><el-input v-model="tag" :placeholder="t('docker.image.build.tagPlaceholder')" class="mono" /></el-form-item>
      <el-form-item :label="t('docker.image.build.source')">
        <el-radio-group v-model="mode">
          <el-radio-button value="dockerfile">{{ t('docker.image.build.inlineDockerfile') }}</el-radio-button>
          <el-radio-button value="git">{{ t('docker.image.build.gitUrl') }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="mode === 'dockerfile'" label="Dockerfile">
        <el-input v-model="dockerfile" type="textarea" :rows="isMobile ? 10 : 12" spellcheck="false" class="mono editor" placeholder="FROM alpine:latest&#10;RUN echo hello" />
      </el-form-item>
      <el-form-item v-else :label="t('docker.image.build.gitUrl')">
        <el-input v-model="gitUrl" placeholder="https://github.com/user/repo.git#branch:subdir" class="mono" />
      </el-form-item>
      <el-form-item :label="t('docker.image.build.buildArgs')">
        <div class="full">
          <div v-for="(a, i) in args" :key="i" class="row">
            <el-input v-model="a.k" placeholder="ARG" class="mono" />
            <el-input v-model="a.v" :placeholder="t('docker.image.build.valuePlaceholder')" class="mono" />
            <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="args.splice(i, 1)" />
          </div>
          <el-button :icon="Plus" @click="args.push({ k: '', v: '' })">{{ t('docker.image.build.addArg') }}</el-button>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :icon="Hammer" :loading="busy" @click="build">{{ t('docker.image.build.start') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Hammer, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import { trackTask } from './tasks'

const { t } = useI18n()

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()
const { isMobile, labelPosition } = useFormLayout()

const tag = ref('')
const mode = ref('dockerfile')
const dockerfile = ref('')
const gitUrl = ref('')
const args = ref<{ k: string; v: string }[]>([])
const busy = ref(false)

function onOpen() {
  tag.value = ''
  dockerfile.value = ''
  gitUrl.value = ''
  args.value = []
  mode.value = 'dockerfile'
}

async function build() {
  if (!tag.value.trim()) {
    ElMessage.warning(t('docker.image.build.tagRequired'))
    return
  }
  const body: Record<string, unknown> = { tag: tag.value.trim() }
  if (mode.value === 'dockerfile') {
    if (!dockerfile.value.trim()) {
      ElMessage.warning(t('docker.image.build.dockerfileRequired'))
      return
    }
    body.dockerfile = dockerfile.value
  } else {
    if (!gitUrl.value.trim()) {
      ElMessage.warning(t('docker.image.build.gitUrlRequired'))
      return
    }
    body.gitUrl = gitUrl.value.trim()
  }
  const ba: Record<string, string> = {}
  for (const a of args.value) {
    if (a.k.trim()) {
      ba[a.k.trim()] = a.v
    }
  }
  if (Object.keys(ba).length) {
    body.buildArgs = ba
  }
  busy.value = true
  try {
    const res = await api.post<{ task: string }>('/api/docker/images/build', body)
    emit('update:modelValue', false)
    trackTask(res.task, t('docker.image.build.task', { tag: tag.value.trim() }), () => emit('done'))
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
.row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.row .el-input {
  flex: 1;
  min-width: 0;
}
.editor :deep(textarea) {
  font-family: var(--g-mono);
  font-size: 13px;
  white-space: pre;
}
@media (max-width: 767.98px) {
  .row {
    flex-direction: column;
    padding: 10px;
    border: 1px solid var(--g-border);
    border-radius: var(--g-radius-sm);
  }
  .row .el-button {
    align-self: flex-end;
    min-height: 40px;
  }
  :deep(input),
  .editor :deep(textarea) {
    font-size: 16px;
  }
}
</style>
