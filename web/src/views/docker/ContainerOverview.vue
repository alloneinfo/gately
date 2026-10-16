<template>
  <div v-loading="loading" class="ov">
    <el-alert v-if="error" type="error" :closable="false" show-icon :title="error" />
    <template v-if="d">
      <div class="g-card">
        <div class="g-card-head"><div class="g-card-title"><Info :size="16" />基本信息</div></div>
        <div class="grid">
          <Field label="状态"><StatusBadge v-bind="containerBadge(d.state)" /> <span class="muted">{{ d.status }}</span></Field>
          <Field label="健康检查">{{ d.healthStatus || '-' }}</Field>
          <Field label="重启次数">{{ d.restartCount }}</Field>
          <Field label="退出码">{{ d.exitCode }}</Field>
          <Field label="镜像" mono wide>{{ d.image }}</Field>
          <Field label="镜像 ID" mono wide>{{ d.imageId }}</Field>
          <Field label="创建时间">{{ formatTime(d.created) }}</Field>
          <Field label="启动时间">{{ formatTime(d.started) }}</Field>
          <Field label="启动命令" mono wide>{{ [d.path, ...(d.args ?? [])].filter(Boolean).join(' ') || '-' }}</Field>
          <Field label="Entrypoint" mono>{{ d.entrypoint || '-' }}</Field>
          <Field label="工作目录" mono>{{ d.workdir || '-' }}</Field>
          <Field label="用户" mono>{{ d.user || '-' }}</Field>
          <Field label="主机名" mono>{{ d.hostname || '-' }}</Field>
          <Field label="重启策略">{{ d.restart || 'no' }}</Field>
          <Field label="特权模式">{{ d.privileged ? '是' : '否' }}</Field>
          <Field v-if="d.composeProject" label="Compose">{{ d.composeProject }} / {{ d.composeService }}</Field>
          <Field label="内存限制">{{ d.memoryMB ? d.memoryMB + ' MB' : '不限' }}</Field>
          <Field label="CPU 限制">{{ d.cpus ? d.cpus + ' 核' : '不限' }}</Field>
        </div>
      </div>

      <div class="g-card">
        <div class="g-card-head"><div class="g-card-title"><Network :size="16" />端口与网络</div></div>
        <div class="g-card-body">
          <div v-if="d.ports?.length" class="chips">
            <el-tag v-for="(p, i) in d.ports" :key="i" type="info" class="mono">{{ p.hostIp ? p.hostIp + ':' : '' }}{{ p.host || '随机' }} → {{ p.container }}/{{ p.proto }}</el-tag>
          </div>
          <div v-else class="muted">无端口映射</div>
          <div v-for="n in d.networkDetails ?? []" :key="n.name" class="net">
            <div class="net-name">{{ n.name }}</div>
            <div class="net-info mono">IP {{ n.ip || '-' }} · 网关 {{ n.gateway || '-' }} · MAC {{ n.mac || '-' }}<span v-if="n.aliases?.length"> · 别名 {{ n.aliases.join(', ') }}</span></div>
          </div>
        </div>
      </div>

      <div class="g-card">
        <div class="g-card-head"><div class="g-card-title"><HardDrive :size="16" />挂载 ({{ d.mounts?.length ?? 0 }})</div></div>
        <div class="g-card-body">
          <div v-for="(m, i) in d.mounts ?? []" :key="i" class="mount">
            <el-tag size="small" :type="m.type === 'volume' ? 'success' : 'info'">{{ m.type }}</el-tag>
            <span class="mono path">{{ m.name || m.source }} → {{ m.destination }}</span>
            <el-tag size="small" :type="m.rw ? 'warning' : 'info'">{{ m.rw ? '读写' : '只读' }}</el-tag>
          </div>
          <div v-if="!d.mounts?.length" class="muted">无挂载</div>
        </div>
      </div>

      <div class="g-card">
        <div class="g-card-head"><div class="g-card-title"><Variable :size="16" />环境变量 ({{ envList.length }})</div></div>
        <div class="g-card-body">
          <div v-for="(e, i) in envList" :key="i" class="kv mono"><b>{{ e.k }}</b><span>{{ e.v }}</span></div>
          <div v-if="!envList.length" class="muted">无环境变量</div>
        </div>
      </div>

      <div class="g-card">
        <div class="g-card-head"><div class="g-card-title"><Tags :size="16" />标签 ({{ labelList.length }})</div></div>
        <div class="g-card-body">
          <div v-for="(e, i) in labelList" :key="i" class="kv mono"><b>{{ e.k }}</b><span>{{ e.v }}</span></div>
          <div v-if="!labelList.length" class="muted">无标签</div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { HardDrive, Info, Network, Tags, Variable } from 'lucide-vue-next'
import { api } from '../../api'
import Field from '../../components/Field.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { containerBadge, formatTime } from './format'
import type { ContainerDetail } from './types'

type Detail = ContainerDetail & { env?: string[]; labels?: Record<string, string> }

const props = defineProps<{ containerId: string }>()
const d = ref<Detail>()
const loading = ref(false)
const error = ref('')

const envList = computed(() =>
  (d.value?.env ?? []).map((l) => {
    const i = l.indexOf('=')
    return i < 0 ? { k: l, v: '' } : { k: l.slice(0, i), v: l.slice(i + 1) }
  }),
)
const labelList = computed(() => Object.entries(d.value?.labels ?? {}).map(([k, v]) => ({ k, v })))

onMounted(async () => {
  loading.value = true
  try {
    d.value = await api.get<Detail>(`/api/docker/containers/${props.containerId}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.ov {
  min-height: 120px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 12px 20px;
  padding: 14px 16px;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.net {
  margin-top: 10px;
}
.net-name {
  font-weight: 600;
  font-size: 13px;
}
.net-info {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.mount {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
  padding: 4px 0;
}
.path {
  font-size: 12px;
  word-break: break-all;
  min-width: 0;
}
.kv {
  display: flex;
  gap: 8px;
  font-size: 12px;
  line-height: 1.7;
  word-break: break-all;
}
.kv b {
  flex: 0 0 auto;
  max-width: 45%;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}
.kv span {
  min-width: 0;
}
@media (max-width: 767.98px) {
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 12px;
    gap: 10px 12px;
  }
  .kv {
    flex-direction: column;
    gap: 0;
  }
  .kv b {
    max-width: none;
  }
}
</style>
