<template>
  <div>
    <el-form-item label="启用 Webhook">
      <el-switch v-model="model.enable" />
    </el-form-item>
    <template v-if="model.enable">
      <el-form-item label="请求地址">
        <el-input v-model="model.url" placeholder="https://example.com/notify?ip={ipAddr}" />
      </el-form-item>
      <el-form-item label="请求方法">
        <el-select v-model="model.method" style="width: 120px">
          <el-option v-for="m in ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']" :key="m" :label="m" :value="m" />
        </el-select>
      </el-form-item>
      <el-form-item label="请求头">
        <el-input v-model="headersText" type="textarea" :rows="2" placeholder="每行一个，格式 Key: Value" />
      </el-form-item>
      <el-form-item v-if="model.method !== 'GET'" label="请求体">
        <el-input v-model="model.body" type="textarea" :rows="3" class="mono" />
      </el-form-item>
      <el-form-item label="成功标识">
        <el-input v-model="successText" type="textarea" :rows="2" placeholder="每行一个，响应包含任意一项即成功；留空只检查 2xx 状态码" />
      </el-form-item>
      <el-form-item label="超时(秒)">
        <el-input-number v-model="model.timeout" :min="0" :max="120" />
        <el-checkbox v-model="model.insecureSkipVerify" class="skip">忽略证书错误</el-checkbox>
      </el-form-item>
      <div class="vars">
        可用变量（<span class="mono">{name}</span> 或兼容 Lucky 的 <span class="mono">#{name}</span>）：
        <el-tag v-for="v in variables" :key="v" size="small" class="var mono">{{ '{' + v + '}' }}</el-tag>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface WebhookConfig {
  enable: boolean
  url: string
  method: string
  headers: string[] | null
  body: string
  successContent: string[] | null
  timeout: number
  insecureSkipVerify: boolean
}

const props = defineProps<{ variables: string[] }>()
const model = defineModel<WebhookConfig>({ required: true })

// 多行文本与数组互转
const headersText = computed({
  get: () => (model.value.headers ?? []).join('\n'),
  set: (v: string) => (model.value.headers = v.split('\n').map((s) => s.trim()).filter(Boolean)),
})
const successText = computed({
  get: () => (model.value.successContent ?? []).join('\n'),
  set: (v: string) => (model.value.successContent = v.split('\n').map((s) => s.trim()).filter(Boolean)),
})
const variables = computed(() => props.variables)
</script>

<style scoped>
.skip {
  margin-left: 16px;
}
.vars {
  margin: 0 0 12px 120px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.var {
  margin: 2px 4px 2px 0;
}
</style>
