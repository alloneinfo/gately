<template>
  <el-dialog
    :model-value="modelValue"
    :title="project ? `编辑 ${project} 的 Compose 文件` : '新建 Compose 项目'"
    width="820px"
    :fullscreen="isMobile"
    top="4vh"
    append-to-body
    destroy-on-close
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="onOpen"
  >
    <div v-loading="loading">
      <el-form :label-position="labelPosition" label-width="80px" @submit.prevent>
        <el-form-item v-if="!project" label="项目名" required>
          <el-input v-model="name" placeholder="小写字母、数字、- 或 _" class="mono name" />
        </el-form-item>
        <el-form-item v-else label="文件">
          <span class="mono path">{{ path }}</span>
          <el-tag v-if="!writable" type="warning" class="gap-l">只读</el-tag>
        </el-form-item>
        <el-form-item label="内容" class="grow">
          <div class="full">
            <div v-if="!project" class="tpl"><el-button size="small" text type="primary" @click="content = sample">插入示例模板</el-button></div>
            <el-input v-model="content" type="textarea" :rows="isMobile ? 18 : 22" spellcheck="false" class="mono editor" :readonly="!!project && !writable" placeholder="services:&#10;  app:&#10;    image: nginx" />
          </div>
        </el-form-item>
      </el-form>
    </div>
    <template #footer>
      <div class="foot">
        <el-button @click="emit('update:modelValue', false)">取消</el-button>
        <template v-if="project">
          <el-button :icon="Save" :loading="saving" :disabled="!writable" @click="save(false)">保存</el-button>
          <el-button type="primary" :icon="Rocket" :loading="saving" :disabled="!writable" @click="save(true)">保存并应用</el-button>
        </template>
        <el-button v-else type="primary" :icon="Rocket" :loading="saving" @click="create">创建并启动</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Rocket, Save } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import { trackTask } from './tasks'

const props = defineProps<{ modelValue: boolean; project?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()
const { isMobile, labelPosition } = useFormLayout()

const sample = `services:
  web:
    image: nginx:latest
    container_name: web
    restart: unless-stopped
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    environment:
      - TZ=Asia/Shanghai
`

const name = ref('')
const content = ref('')
const path = ref('')
const writable = ref(true)
const loading = ref(false)
const saving = ref(false)

async function onOpen() {
  name.value = ''
  content.value = ''
  writable.value = true
  if (!props.project) {
    return
  }
  loading.value = true
  try {
    const r = await api.get<{ path: string; content: string; writable: boolean }>(`/api/docker/compose/${encodeURIComponent(props.project)}/config`)
    path.value = r.path
    content.value = r.content
    writable.value = r.writable
  } catch (e) {
    ElMessage.error((e as Error).message)
    emit('update:modelValue', false)
  } finally {
    loading.value = false
  }
}

async function save(apply: boolean) {
  saving.value = true
  try {
    const r = await api.put<{ task?: string; ok?: boolean }>(`/api/docker/compose/${encodeURIComponent(props.project!)}/config${apply ? '?apply=1' : ''}`, { content: content.value })
    emit('update:modelValue', false)
    if (r?.task) {
      trackTask(r.task, `应用 ${props.project}`, () => emit('done'))
    } else {
      ElMessage.success('已保存')
      emit('done')
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function create() {
  const n = name.value.trim()
  if (!/^[a-z0-9][a-z0-9_-]*$/.test(n)) {
    ElMessage.warning('项目名只能包含小写字母、数字、- 和 _，且以字母或数字开头')
    return
  }
  if (!content.value.trim()) {
    ElMessage.warning('请填写 Compose 内容')
    return
  }
  saving.value = true
  try {
    const r = await api.post<{ task: string }>('/api/docker/compose/projects', { name: n, content: content.value })
    emit('update:modelValue', false)
    trackTask(r.task, `创建项目 ${n}`, () => emit('done'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.name {
  max-width: 320px;
}
.path {
  font-size: 12px;
  word-break: break-all;
}
.gap-l {
  margin-left: 8px;
}
.full {
  width: 100%;
}
.tpl {
  margin-bottom: 4px;
}
.editor :deep(textarea) {
  font-family: var(--g-mono);
  font-size: 13px;
  line-height: 1.5;
  white-space: pre;
  overflow-x: auto;
}
.foot {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  padding-bottom: env(safe-area-inset-bottom);
}
.foot .el-button + .el-button {
  margin-left: 0;
}
@media (max-width: 767.98px) {
  .name {
    max-width: none;
  }
  .foot .el-button {
    flex: 1;
    min-height: 40px;
  }
  .editor :deep(textarea),
  .name :deep(input) {
    font-size: 16px;
  }
}
</style>
