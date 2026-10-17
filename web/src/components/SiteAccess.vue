<template>
  <div>
    <el-form-item v-if="site.webAuth || site.basicAuth" label="认证范围">
      <el-checkbox v-model="site.authOptional">仅对下方标记为「必须认证」的路径启用认证，其余路径公开</el-checkbox>
    </el-form-item>

    <el-form-item label="路径规则">
      <div class="rows">
        <div v-for="(r, i) in site.rules" :key="i" class="row">
          <el-input v-model="r.prefix" placeholder="/admin" class="w-prefix mono" />
          <el-select v-model="r.auth" class="w-auth">
            <el-option label="认证：沿用站点" value="" />
            <el-option label="无需认证" value="off" />
            <el-option label="必须认证" value="on" />
          </el-select>
          <el-select v-model="r.ipMode" class="w-ip">
            <el-option label="IP：沿用站点" value="" />
            <el-option label="IP 白名单" value="whitelist" />
            <el-option label="IP 黑名单" value="blacklist" />
          </el-select>
          <el-input v-if="r.ipMode" :model-value="(r.ipList ?? []).join(', ')" placeholder="192.168.1.0/24, 10.0.0.1" class="mono flex" @update:model-value="(v: string) => (r.ipList = v.split(/[,\s]+/).filter(Boolean))" />
          <el-button text type="danger" :icon="Trash2" @click="site.rules.splice(i, 1)" />
        </div>
        <el-button text :icon="Plus" @click="site.rules.push({ prefix: '/', auth: '', ipMode: '', ipList: [] })">添加规则</el-button>
        <div class="form-help">按路径前缀匹配，最长前缀优先；路径会先规范化（消除 ../ 与重复斜杠）再匹配</div>
      </div>
    </el-form-item>

    <el-form-item label="国家过滤">
      <el-radio-group v-model="site.countryMode">
        <el-radio-button value="">关闭</el-radio-button>
        <el-radio-button value="allow">仅允许</el-radio-button>
        <el-radio-button value="deny">拒绝</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item v-if="site.countryMode" label="国家代码">
      <el-input :model-value="(site.countries ?? []).join(', ')" class="mono" placeholder="两位 ISO 代码，逗号分隔，如 CN, HK, US" @update:model-value="(v: string) => (site.countries = v.split(/[,\s]+/).filter(Boolean))" />
      <div class="form-help">
        内网地址始终放行。
        <span v-if="geoLoaded">GeoIP 数据库已加载。</span>
        <span v-else class="warn">尚未导入 GeoIP 数据库，过滤暂不生效（在「全局设置」中导入）。</span>
      </div>
    </el-form-item>

    <el-form-item v-if="site.webAuth || site.basicAuth" label="分享链接">
      <div class="rows">
        <div v-for="(l, i) in site.shareLinks" :key="l.key ?? i" class="row">
          <el-input v-model="l.name" placeholder="名称，如 给张三" class="w-name" />
          <el-date-picker v-model="l.expiresAt" type="datetime" placeholder="不过期" value-format="YYYY-MM-DDTHH:mm:ssZ" class="w-date" />
          <el-input-number v-model="l.maxUses" :min="0" controls-position="right" class="w-uses" />
          <el-input v-model="l.pathPrefix" placeholder="路径限制，如 /docs" class="mono w-path" />
          <el-button text :icon="Copy" :disabled="!l.token" :title="l.token ? '复制链接' : '保存后可复制'" @click="copyLink(l)" />
          <el-button text type="danger" :icon="Trash2" @click="site.shareLinks.splice(i, 1)" />
        </div>
        <el-button text :icon="Plus" @click="addLink">添加分享链接</el-button>
        <div class="form-help">持链接访问一次即可获得最长 24 小时的免登录会话。次数为 0 表示不限；兑换次数在重启后清零；删除链接后已发出的会话立即失效</div>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Copy, Plus, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import { copyText } from '../composables/useClipboard'

export interface PathRule {
  prefix: string
  auth: string
  ipMode: string
  ipList: string[]
}

export interface ShareLink {
  key?: string
  name: string
  token: string
  expiresAt?: string
  maxUses: number
  pathPrefix: string
}

export interface AccessFields {
  webAuth: boolean
  basicAuth: boolean
  domains: string[] | null
  authOptional: boolean
  countryMode: string
  countries: string[] | null
  rules: PathRule[]
  shareLinks: ShareLink[]
}

const props = defineProps<{ tls: boolean; port: number }>()
const site = defineModel<AccessFields>('site', { required: true })
const geoLoaded = ref(true)

onMounted(async () => {
  try {
    geoLoaded.value = (await api.get<{ loaded: boolean }>('/api/webservice/geoip')).loaded
  } catch {
    geoLoaded.value = false
  }
})

function addLink() {
  site.value.shareLinks.push({ name: '', token: '', maxUses: 0, pathPrefix: '' })
}

// copyLink 用站点的首个域名与监听端口拼出分享地址
function copyLink(l: ShareLink) {
  const host = (site.value.domains ?? []).find((d) => d !== '*' && !d.startsWith('*.')) ?? location.hostname
  const defaultPort = props.tls ? 443 : 80
  const port = props.port === defaultPort ? '' : `:${props.port}`
  copyText(`${props.tls ? 'https' : 'http'}://${host}${port}/?_gs=${l.token}`)
}
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
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}
.flex {
  flex: 1;
  min-width: 140px;
}
.w-prefix {
  width: 130px;
}
.w-auth,
.w-ip {
  width: 140px;
}
.w-name {
  width: 130px;
}
.w-date {
  width: 180px;
}
.w-uses {
  width: 100px;
}
.w-path {
  width: 140px;
}
.warn {
  color: var(--el-color-warning);
}
</style>
