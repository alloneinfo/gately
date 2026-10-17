<template>
  <div>
    <el-divider content-position="left">{{ t('accessSettings.autoBan') }}</el-divider>
    <el-form-item :label="t('accessSettings.authFailBan')">
      <el-input-number v-model="minutes" :min="0" :max="43200" />
      <span class="inline-help">{{ t('accessSettings.authFailBanHelp') }}</span>
    </el-form-item>
    <el-form-item :label="t('accessSettings.banList')">
      <div class="rows">
        <el-empty v-if="!bans.length" :image-size="36" :description="t('accessSettings.noBans')" />
        <div v-for="b in bans" :key="b.key" class="row">
          <span class="mono key">{{ b.key }}</span>
          <el-tag size="small" :type="b.manual ? 'info' : 'warning'">{{ b.reason }}</el-tag>
          <span class="meta">{{ t('accessSettings.until', { time: new Date(b.until).toLocaleString() }) }}</span>
          <el-button text size="small" @click="unban(b.key)">{{ t('accessSettings.unban') }}</el-button>
        </div>
        <div class="row">
          <el-input v-model="newIp" :placeholder="t('accessSettings.manualBanIp')" class="mono ip" />
          <el-input-number v-model="newHours" :min="1" :max="720" />
          <span class="meta">{{ t('accessSettings.hours') }}</span>
          <el-button size="small" :disabled="!newIp" @click="addBan">{{ t('accessSettings.ban') }}</el-button>
        </div>
      </div>
    </el-form-item>

    <el-divider content-position="left">{{ t('accessSettings.geoTitle') }}</el-divider>
    <el-form-item :label="t('common.status')">
      <div class="rows">
        <div v-if="geo.loaded">
          {{ t('accessSettings.geoLoaded', { type: geo.type, buildTime: geo.buildTime ? new Date(geo.buildTime).toLocaleDateString() : '-', size: (geo.size / 1024 / 1024).toFixed(1) }) }}
          <el-button text size="small" type="danger" @click="removeGeo">{{ t('accessSettings.remove') }}</el-button>
        </div>
        <div v-else class="meta">{{ t('accessSettings.geoNotImported') }}</div>
        <div class="row">
          <el-upload :show-file-list="false" :http-request="upload" accept=".mmdb">
            <el-button size="small" :icon="Upload" :loading="busy">{{ t('accessSettings.uploadMmdb') }}</el-button>
          </el-upload>
        </div>
        <div class="row">
          <el-input v-model="geoUrl" :placeholder="t('accessSettings.mmdbUrlPlaceholder')" class="mono flex" />
          <el-button size="small" :loading="busy" :disabled="!geoUrl" @click="download">{{ t('common.download') }}</el-button>
        </div>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Upload } from 'lucide-vue-next'
import { api } from '../api'

interface Ban {
  key: string
  until: string
  reason: string
  manual: boolean
}
interface GeoInfo {
  loaded: boolean
  type?: string
  buildTime?: string
  size: number
}

const { t } = useI18n()
const seconds = defineModel<number>('autoBanSeconds', { required: true })
const minutes = computed({ get: () => Math.round((seconds.value ?? 0) / 60), set: (v: number) => (seconds.value = (v ?? 0) * 60) })
const bans = ref<Ban[]>([])
const geo = ref<GeoInfo>({ loaded: false, size: 0 })
const newIp = ref('')
const newHours = ref(24)
const geoUrl = ref('')
const busy = ref(false)

async function loadBans() {
  bans.value = await api.get<Ban[]>('/api/webservice/bans')
}

async function loadGeo() {
  geo.value = await api.get<GeoInfo>('/api/webservice/geoip')
}

async function addBan() {
  try {
    await api.post('/api/webservice/bans', { key: newIp.value, seconds: newHours.value * 3600 })
    newIp.value = ''
    await loadBans()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function unban(key: string) {
  try {
    await api.del('/api/webservice/bans', { key })
    await loadBans()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function upload(opt: { file: File }) {
  busy.value = true
  try {
    const fd = new FormData()
    fd.append('file', opt.file, opt.file.name)
    geo.value = await api.post<GeoInfo>('/api/webservice/geoip/upload', fd)
    ElMessage.success(t('accessSettings.geoImported'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

async function download() {
  busy.value = true
  try {
    geo.value = await api.post<GeoInfo>('/api/webservice/geoip/download', { url: geoUrl.value })
    ElMessage.success(t('accessSettings.geoDownloaded'))
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    busy.value = false
  }
}

async function removeGeo() {
  try {
    geo.value = await api.del<GeoInfo>('/api/webservice/geoip')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(() => {
  Promise.all([loadBans(), loadGeo()]).catch((e) => ElMessage.error((e as Error).message))
})
</script>

<style scoped>
.rows {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.key {
  min-width: 150px;
}
.ip {
  width: 200px;
}
.flex {
  flex: 1;
}
.meta,
.inline-help {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.inline-help {
  margin-left: 10px;
}
</style>
