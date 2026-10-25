<template>
  <div class="oidc">
    <el-form-item :label="t('oidc.enable')">
      <el-switch v-model="model.enable" />
    </el-form-item>
    <el-form-item :label="t('oidc.issuer')">
      <el-input v-model="model.issuer" :placeholder="t('oidc.issuerPlaceholder')" />
      <div class="form-help">{{ t('oidc.issuerHelp') }}</div>
    </el-form-item>
    <el-form-item v-if="issuer" :label="t('oidc.discovery')">
      <div class="endpoints">
        <div class="ep main">
          <code ref="discoveryEl" class="mono">{{ issuer + OIDC_DISCOVERY }}</code>
          <el-button size="small" text :icon="Copy" :aria-label="t('oidc.copy')" @click="copyText(issuer + OIDC_DISCOVERY, { el: discoveryEl })" />
        </div>
        <div class="form-help">{{ t('oidc.discoveryHelp') }}</div>
        <el-collapse class="more">
          <el-collapse-item :title="t('oidc.endpoints')" name="ep">
            <div v-for="(path, key) in OIDC_ENDPOINTS" :key="key" class="ep">
              <span class="ep-name">{{ t(`oidc.ep.${key}`) }}</span>
              <code class="mono">{{ issuer + path }}</code>
              <el-button size="small" text :icon="Copy" :aria-label="t('oidc.copy')" @click="copyText(issuer + path)" />
            </div>
            <div class="form-help">{{ t('oidc.scopesHelp') }}</div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-form-item>
    <el-form-item :label="t('oidc.clients')">
      <div class="clients">
        <OIDCClientCard v-for="(_, i) in model.clients" :key="i" v-model="model.clients[i]" @remove="model.clients.splice(i, 1)" />
        <div class="actions">
          <el-button size="small" :icon="Plus" @click="add">{{ t('oidc.addClient') }}</el-button>
          <a :href="docs" target="_blank" rel="noopener" class="doc-link">{{ t('oidc.docs') }}</a>
        </div>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Copy, Plus } from 'lucide-vue-next'
import OIDCClientCard from './OIDCClientCard.vue'
import { copyText } from '../composables/useClipboard'
import { docsUrl } from '../docsUrl'
import { newClient, OIDC_DISCOVERY, OIDC_ENDPOINTS, type OIDCSettings } from '../oidc'

const { t, locale } = useI18n()
const model = defineModel<OIDCSettings>({ required: true })
const discoveryEl = ref<HTMLElement | null>(null)
// 规范化后的签发者地址（去掉末尾斜杠），未填写时不显示端点
const issuer = computed(() => model.value.issuer.trim().replace(/\/+$/, ''))
const docs = computed(() => docsUrl(locale.value, '/docs/modules/oidc'))

function add() {
  model.value.clients.push(newClient(model.value.clients.map((c) => c.id)))
}
</script>

<style scoped>
.oidc {
  width: 100%;
}
.endpoints,
.clients {
  width: 100%;
}
.ep {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.ep code {
  overflow-wrap: anywhere;
  font-size: 12.5px;
}
.ep.main code {
  font-size: 13px;
}
.ep-name {
  flex: none;
  width: 92px;
  color: var(--el-text-color-secondary);
  font-size: 12.5px;
}
.more {
  margin-top: 4px;
  border: none;
}
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.doc-link {
  font-size: 13px;
  color: var(--el-color-primary);
}
</style>
