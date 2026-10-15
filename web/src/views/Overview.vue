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
    <div class="modules">
      <el-card v-for="m in summaryModules" :key="m.id" shadow="hover" class="module-card" @click="go(m.id)">
        <div class="module-title">{{ m.title }}</div>
        <div v-for="it in m.summary" :key="it.label" class="item">
          <span class="label">{{ it.label }}</span>
          <span :class="['value', it.level]">{{ it.value }}</span>
        </div>
      </el-card>
    </div>
    <el-card shadow="never" header="系统日志" class="logs">
      <LogViewer url="/api/logs" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, formatBytes, type Status } from '../api'
import LogViewer from '../components/LogViewer.vue'

const status = ref<Status>()
const router = useRouter()
// 模块 ID 与页面路由的对应关系
const routes: Record<string, string> = { webservice: '/web' }
const summaryModules = computed(() => (status.value?.modules ?? []).filter((m) => m.summary?.length))

function go(id: string) {
  router.push(routes[id] ?? '/' + id)
}
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
  timer = window.setInterval(async () => {
    now.value = Date.now()
    status.value = await api.get<Status>('/api/status')
  }, 10000)
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
.modules {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
  margin-top: 16px;
}
.module-card {
  cursor: pointer;
}
.module-title {
  font-weight: 600;
  margin-bottom: 8px;
}
.item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  line-height: 1.9;
}
.label {
  color: var(--el-text-color-secondary);
}
.value.ok {
  color: var(--el-color-success);
}
.value.warn {
  color: var(--el-color-warning);
}
.value.error {
  color: var(--el-color-danger);
}
</style>
