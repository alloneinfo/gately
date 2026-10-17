<template>
  <div>
    <el-divider content-position="left">自动封禁</el-divider>
    <el-form-item label="认证失败封禁">
      <el-input-number v-model="minutes" :min="0" :max="43200" />
      <span class="inline-help">分钟；认证连续失败达到 10 次后封禁来源 IP，0 为不封禁（仅限流 15 分钟）</span>
    </el-form-item>
    <el-form-item label="封禁列表">
      <div class="rows">
        <el-empty v-if="!bans.length" :image-size="36" description="当前没有被封禁的 IP" />
        <div v-for="b in bans" :key="b.key" class="row">
          <span class="mono key">{{ b.key }}</span>
          <el-tag size="small" :type="b.manual ? 'info' : 'warning'">{{ b.reason }}</el-tag>
          <span class="meta">至 {{ new Date(b.until).toLocaleString() }}</span>
          <el-button text size="small" @click="unban(b.key)">解除</el-button>
        </div>
        <div class="row">
          <el-input v-model="newIp" placeholder="手动封禁 IP" class="mono ip" />
          <el-input-number v-model="newHours" :min="1" :max="720" />
          <span class="meta">小时</span>
          <el-button size="small" :disabled="!newIp" @click="addBan">封禁</el-button>
        </div>
      </div>
    </el-form-item>

    <el-divider content-position="left">GeoIP 数据库（国家过滤）</el-divider>
    <el-form-item label="状态">
      <div class="rows">
        <div v-if="geo.loaded">
          已加载：{{ geo.type }}，构建于 {{ geo.buildTime ? new Date(geo.buildTime).toLocaleDateString() : '-' }}，{{ (geo.size / 1024 / 1024).toFixed(1) }} MB
          <el-button text size="small" type="danger" @click="removeGeo">移除</el-button>
        </div>
        <div v-else class="meta">未导入。支持 MaxMind GeoLite2 / DB-IP 等 .mmdb 国家库或城市库</div>
        <div class="row">
          <el-upload :show-file-list="false" :http-request="upload" accept=".mmdb">
            <el-button size="small" :icon="Upload" :loading="busy">上传 .mmdb</el-button>
          </el-upload>
        </div>
        <div class="row">
          <el-input v-model="geoUrl" placeholder="或填写 .mmdb 下载地址" class="mono flex" />
          <el-button size="small" :loading="busy" :disabled="!geoUrl" @click="download">下载</el-button>
        </div>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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
    ElMessage.success('GeoIP 数据库已导入')
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
    ElMessage.success('GeoIP 数据库已下载并加载')
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
