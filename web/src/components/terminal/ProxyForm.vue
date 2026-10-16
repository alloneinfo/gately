<template>
  <!-- SSH / Telnet 共用的代理设置 -->
  <el-form-item label="使用代理">
    <el-switch v-model="proxy.enabled" />
    <span class="help">通过 SOCKS5 或 HTTP 代理连接目标主机</span>
  </el-form-item>
  <template v-if="proxy.enabled">
    <el-form-item label="代理类型">
      <el-radio-group v-model="proxy.type">
        <el-radio-button value="socks5">SOCKS5</el-radio-button>
        <el-radio-button value="http">HTTP</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="代理地址">
      <div class="row">
        <el-input v-model="proxy.host" placeholder="主机" />
        <el-input-number v-model="proxy.port" :min="1" :max="65535" controls-position="right" />
      </div>
    </el-form-item>
    <el-form-item label="代理账号">
      <div class="row">
        <el-input v-model="proxy.username" placeholder="用户名（可选）" />
        <el-input v-model="proxy.password" type="password" show-password placeholder="密码（可选）" />
      </div>
    </el-form-item>
  </template>
</template>

<script setup lang="ts">
import type { ProxyConf } from './types'

defineProps<{ proxy: ProxyConf }>()
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
