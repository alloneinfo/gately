<template>
  <div>
    <el-form-item :label="`获取 IPv${family}`">
      <el-switch v-model="model.enable" />
    </el-form-item>
    <template v-if="model.enable">
      <el-form-item label="获取方式">
        <el-radio-group v-model="model.type">
          <el-radio-button value="url">查询接口</el-radio-button>
          <el-radio-button value="netInterface">网卡</el-radio-button>
          <el-radio-button value="command">命令</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="model.type === 'url'" label="查询地址">
        <el-input v-model="urlsText" type="textarea" :rows="4" placeholder="每行一个，按顺序尝试" class="mono" />
      </el-form-item>
      <el-form-item v-if="model.type === 'netInterface'" label="网卡">
        <el-select v-model="model.interface" filterable allow-create style="width: 100%" placeholder="选择网卡">
          <el-option v-for="i in interfaces" :key="i.name" :label="i.name" :value="i.name">
            <span>{{ i.name }}</span>
            <span class="iface-addrs">{{ i.addrs.join('  ') }}</span>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item v-if="model.type === 'command'" label="命令">
        <el-input v-model="model.command" placeholder="输出中包含 IP 即可，例如 curl -s https://4.ipw.cn" class="mono" />
      </el-form-item>
      <el-form-item label="筛选正则">
        <el-input v-model="model.regex" placeholder="可选；留空自动选择第一个公网地址" class="mono" />
      </el-form-item>
      <el-form-item>
        <el-button size="small" :loading="testing" @click="test">测试获取</el-button>
        <span v-if="result" class="result" :class="{ err: resultError }">{{ result }}</span>
      </el-form-item>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../api'

export interface IPSource {
  enable: boolean
  type: string
  urls: string[] | null
  interface: string
  regex: string
  command: string
}

export interface NetInterface {
  name: string
  addrs: string[]
}

const props = defineProps<{ family: 4 | 6; interfaces: NetInterface[] }>()
const model = defineModel<IPSource>({ required: true })
const testing = ref(false)
const result = ref('')
const resultError = ref(false)
const interfaces = computed(() => props.interfaces)

const urlsText = computed({
  get: () => (model.value.urls ?? []).join('\n'),
  set: (v: string) => (model.value.urls = v.split('\n').map((s) => s.trim()).filter(Boolean)),
})

async function test() {
  testing.value = true
  result.value = ''
  try {
    const r = await api.post<{ ip: string }>('/api/ddns/iptest', { source: model.value, family: props.family })
    result.value = `获取成功：${r.ip}`
    resultError.value = false
  } catch (e) {
    result.value = (e as Error).message
    resultError.value = true
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.iface-addrs {
  float: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 16px;
}
.result {
  margin-left: 12px;
  word-break: break-all;
  font-size: 13px;
  color: var(--el-color-success);
}
.result.err {
  color: var(--el-color-danger);
}
</style>
