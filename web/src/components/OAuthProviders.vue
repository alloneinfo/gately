<template>
  <div>
    <div v-for="(p, i) in model" :key="i" class="provider">
      <div class="provider-row">
        <el-select v-model="p.type" class="w-120" @change="onType(p)">
          <el-option v-for="t in types" :key="t.value" :label="t.label" :value="t.value" />
        </el-select>
        <el-input v-model="p.key" placeholder="标识，如 github" class="w-120" />
        <el-input v-model="p.name" placeholder="显示名" class="w-120" />
        <el-button text type="danger" :icon="Trash2" aria-label="删除" @click="model.splice(i, 1)" />
      </div>
      <div class="provider-row">
        <el-input v-model="p.clientId" placeholder="Client ID（QQ 为 APP ID）" />
        <el-input v-model="p.clientSecret" type="password" show-password placeholder="Client Secret（QQ 为 APP Key）" />
      </div>
      <div class="form-help">{{ hintOf(p) }}</div>
      <div v-if="p.type === 'oidc'" class="provider-row">
        <el-input v-model="p.issuer" placeholder="Issuer，如 https://login.example.com" />
      </div>
    </div>
    <el-button size="small" :icon="Plus" @click="add">添加登录方式</el-button>
  </div>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import type { OAuthProvider } from '../api'

const model = defineModel<OAuthProvider[]>({ required: true })

const types = [
  { value: 'github', label: 'GitHub' },
  { value: 'google', label: 'Google' },
  { value: 'qq', label: 'QQ' },
  { value: 'oidc', label: '通用 OIDC' },
]
const names: Record<string, string> = { github: 'GitHub', google: 'Google', qq: 'QQ', oidc: '' }
const hints: Record<string, string> = {
  github: '在 GitHub → Settings → Developer settings → OAuth Apps 创建应用；身份可写登录名或数字 ID（数字 ID 不会因改名失效）。',
  google: '在 Google Cloud Console 创建 OAuth 客户端（Web 应用）；身份可写已验证的邮箱或 sub。',
  qq: '在 QQ 互联创建网站应用；openid 对每个应用不同，首次登录被拒绝时页面会提示账号标识，加入允许列表即可。',
  oidc: '任意支持 OIDC 发现文档的服务（Authentik、Keycloak、Casdoor 等）；身份可写已验证的邮箱或 sub。',
}

function hintOf(p: OAuthProvider) {
  return hints[p.type] ?? ''
}

// 切换类型时，标识与显示名还是空或仍为上一个类型的默认值，则同步为新类型的默认值
function onType(p: OAuthProvider) {
  const prevDefaults = Object.keys(names)
  if (!p.key || prevDefaults.includes(p.key)) {
    p.key = p.type
  }
  if (!p.name || Object.values(names).includes(p.name)) {
    p.name = names[p.type]
  }
}

function add() {
  const used = new Set(model.value.map((p) => p.key))
  const type = ['github', 'google', 'qq'].find((t) => !used.has(t)) ?? 'oidc'
  model.value.push({ key: type === 'oidc' ? '' : type, name: names[type], type, clientId: '', clientSecret: '', issuer: '' })
}
</script>

<style scoped>
.w-120 {
  width: 120px;
}
.provider {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
@media (max-width: 767.98px) {
  .provider-row {
    flex-wrap: wrap;
  }
  .provider-row .el-input,
  .provider-row .el-select {
    width: calc(50% - 4px) !important;
    flex: 1 1 calc(50% - 4px);
  }
}
</style>
