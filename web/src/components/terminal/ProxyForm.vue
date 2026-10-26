<template>
  <!-- SSH / Telnet 共用的代理设置 -->
  <el-form-item :label="t('terminal.proxy.use')">
    <el-switch v-model="proxy.enabled" />
    <span class="help">{{ t('terminal.proxy.useHelp') }}</span>
  </el-form-item>
  <template v-if="proxy.enabled">
    <el-form-item :label="t('terminal.proxy.type')">
      <el-radio-group v-model="proxy.type">
        <el-radio-button value="socks5">SOCKS5</el-radio-button>
        <el-radio-button value="http">HTTP</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item :label="t('terminal.proxy.addr')">
      <div class="row">
        <el-input v-model="proxy.host" :placeholder="t('terminal.proxy.hostPlaceholder')" />
        <el-input-number v-model="proxy.port" :min="1" :max="65535" controls-position="right" />
      </div>
    </el-form-item>
    <el-form-item :label="t('terminal.proxy.account')">
      <div class="row">
        <el-input v-model="proxy.username" :placeholder="t('terminal.proxy.userPlaceholder')" />
        <SecretInput v-model="proxy.password" :placeholder="t('terminal.proxy.passPlaceholder')" :reveal="reveal" />
      </div>
    </el-form-item>
  </template>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ProxyConf } from './types'
import SecretInput from '../SecretInput.vue'

const { t } = useI18n()

// reveal 取回已保存的代理密码明文，新建连接时不提供
defineProps<{ proxy: ProxyConf; reveal?: () => Promise<string | undefined> }>()
</script>

<style scoped>
.row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.row > :first-child {
  flex: 1;
  min-width: 0;
}
.row .el-input-number {
  width: 130px;
}
@media (max-width: 767.98px) {
  .row {
    flex-wrap: wrap;
  }
  .row > * {
    width: 100% !important;
    flex: 1 1 100%;
  }
}
</style>
