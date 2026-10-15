<template>
  <div>
    <el-row :gutter="16">
      <el-col :xs="24" :md="12">
        <el-card shadow="never" header="运行信息">
          <el-descriptions v-if="status" :column="1" border>
            <el-descriptions-item label="版本">{{ status.version }} ({{ status.commit }})</el-descriptions-item>
            <el-descriptions-item label="构建时间">{{ status.buildDate }}</el-descriptions-item>
            <el-descriptions-item label="平台">{{ status.os }}/{{ status.arch }} · {{ status.goVersion }}</el-descriptions-item>
            <el-descriptions-item label="已运行">{{ uptime }}</el-descriptions-item>
            <el-descriptions-item label="内存">{{ formatBytes(status.memAlloc) }}</el-descriptions-item>
            <el-descriptions-item label="协程">{{ status.goroutines }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never" header="透明度承诺">
          <ul class="promise">
            <li>不连接任何项目方服务器，不上报设备信息</li>
            <li>所有对外地址（公网 IP 查询、STUN 服务器）都可见、可改</li>
            <li>配置为明文 JSON，可直接纳入 git 审计</li>
            <li>源码 Apache-2.0 开源，欢迎审查与贡献</li>
          </ul>
        </el-card>
      </el-col>
    </el-row>
    <el-card shadow="never" header="系统日志" class="logs">
      <LogViewer url="/api/logs" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api, formatBytes, type Status } from '../api'
import LogViewer from '../components/LogViewer.vue'

const status = ref<Status>()
const now = ref(Date.now())
let timer: number | undefined

const uptime = computed(() => {
  if (!status.value) {
    return ''
  }
  let s = Math.floor((now.value - new Date(status.value.startTime).getTime()) / 1000)
  const d = Math.floor(s / 86400)
  s %= 86400
  const h = Math.floor(s / 3600)
  s %= 3600
  const m = Math.floor(s / 60)
  return `${d} 天 ${h} 小时 ${m} 分`
})

onMounted(async () => {
  status.value = await api.get<Status>('/api/status')
  timer = window.setInterval(() => (now.value = Date.now()), 30000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.promise {
  margin: 0;
  padding-left: 18px;
  line-height: 2;
}
.logs {
  margin-top: 16px;
}
</style>
