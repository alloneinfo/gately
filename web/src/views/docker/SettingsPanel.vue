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
        <!-- 守护进程自身的加速地址（daemon.json）只读展示，读取 /info 失败时不显示 -->
        <el-form-item v-if="daemonMirrors" :label="t('docker.settings.daemonMirrors')">
          <div class="full">
            <div v-if="daemonMirrors.length" class="tags">
              <el-tag v-for="m in daemonMirrors" :key="m" type="info" class="mono">{{ m }}</el-tag>
            </div>
            <span v-else class="help">{{ t('docker.settings.daemonMirrorsNone') }}</span>
            <div class="form-help">{{ t('docker.settings.daemonMirrorsHelp') }}</div>
          </div>
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
              <SecretInput v-model="r.password" :placeholder="t('docker.settings.passwordPlaceholder')" :reveal="secretOf('/api/docker/settings/secrets', `registry:${r.server}`)" />
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
import SecretInput from '../../components/SecretInput.vue'
import { secretOf } from '../../secrets'
import type { DockerInfo, DockerSettings } from './types'

const { t } = useI18n()

const emit = defineEmits<{ saved: [] }>()
const { labelPosition } = useFormLayout()
const s = reactive<DockerSettings>({ host: '', mirrors: [], registries: [], autoCheckHours: 0 })
const loaded = ref(false)
const loading = ref(false)
const saving = ref(false)
// Docker 守护进程已配置的镜像加速；null 表示未能读取（Docker 不可达等）
const daemonMirrors = ref<string[] | null>(null)

// loadDaemonMirrors 读取守护进程的镜像加速，失败时静默隐藏，不影响设置本身
async function loadDaemonMirrors() {
  try {
    const info = await api.get<DockerInfo>('/api/docker/info')
    daemonMirrors.value = info.daemonMirrors ?? []
  } catch {
    daemonMirrors.value = null
  }
}

async function load() {
  loading.value = true
  void loadDaemonMirrors()
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
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tags .el-tag {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
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
