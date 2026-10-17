<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><SlidersHorizontal :size="16" />{{ t('docker.settings.title') }}</div>
      <el-button type="primary" :icon="Save" :loading="saving" :disabled="!loaded" @click="save">{{ t('common.save') }}</el-button>
    </div>
    <div v-loading="loading" class="g-card-body">
      <el-form v-if="loaded" :label-position="labelPosition" label-width="110px" @submit.prevent>
        <el-form-item :label="t('docker.settings.host')">
          <el-input v-model="s.host" placeholder="unix:///var/run/docker.sock" class="mono" />
          <div class="form-help">{{ t('docker.settings.hostHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('docker.settings.autoCheck')">
          <el-input-number v-model="s.autoCheckHours" :min="0" :max="720" />
          <span class="help">{{ t('docker.settings.autoCheckHelp') }}</span>
        </el-form-item>
        <el-form-item :label="t('docker.settings.mirrors')">
          <div class="full">
            <div v-for="(m, idx) in s.mirrors" :key="idx" class="row">
              <el-input v-model="s.mirrors[idx]" :title="m" :placeholder="t('docker.settings.mirrorPlaceholder')" class="mono" />
              <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="s.mirrors.splice(idx, 1)" />
            </div>
            <el-button :icon="Plus" @click="s.mirrors.push('')">{{ t('docker.settings.addMirror') }}</el-button>
            <div class="form-help">{{ t('docker.settings.mirrorHelp') }}</div>
          </div>
        </el-form-item>
        <el-form-item :label="t('docker.settings.registries')">
          <div class="full">
            <div v-for="(r, i) in s.registries" :key="i" class="row reg">
              <el-input v-model="r.server" :placeholder="t('docker.settings.serverPlaceholder')" class="mono" />
              <el-input v-model="r.username" :placeholder="t('docker.settings.username')" />
              <el-input v-model="r.password" type="password" show-password :placeholder="t('docker.settings.passwordPlaceholder')" />
              <el-button text type="danger" :icon="Trash2" :aria-label="t('common.delete')" @click="s.registries.splice(i, 1)" />
            </div>
            <el-button :icon="Plus" @click="s.registries.push({ server: '', username: '', password: '' })">{{ t('docker.settings.addRegistry') }}</el-button>
            <div class="form-help">{{ t('docker.settings.registryHelp') }}</div>
          </div>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Plus, Save, SlidersHorizontal, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import type { DockerSettings } from './types'

const { t } = useI18n()

const emit = defineEmits<{ saved: [] }>()
const { labelPosition } = useFormLayout()
const s = reactive<DockerSettings>({ host: '', mirrors: [], registries: [], autoCheckHours: 0 })
const loaded = ref(false)
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const r = await api.get<DockerSettings>('/api/docker/settings')
    Object.assign(s, { ...r, mirrors: r.mirrors ?? [], registries: r.registries ?? [] })
    loaded.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await api.put('/api/docker/settings', {
      ...s,
      mirrors: s.mirrors.map((m) => m.trim()).filter(Boolean),
      registries: s.registries.filter((r) => r.server.trim() !== '').map((r) => ({ ...r, server: r.server.trim() })),
    })
    ElMessage.success(t('common.saved'))
    emit('saved')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
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
@media (max-width: 767.98px) {
  .reg {
    flex-direction: column;
    padding: 10px;
    border: 1px solid var(--g-border);
    border-radius: var(--g-radius-sm);
  }
  .reg .el-button {
    align-self: flex-end;
  }
  .row .el-button {
    min-height: 40px;
  }
  :deep(input) {
    font-size: 16px;
  }
}
</style>
