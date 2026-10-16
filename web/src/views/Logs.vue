<template>
  <div>
    <PageHeader title="系统日志" desc="Gately 运行日志与各模块日志集中查看，仅保存在内存中，重启后清空" :icon="ScrollText">
      <el-segmented v-model="source" :options="sources" class="sources" />
    </PageHeader>
    <div class="g-card">
      <div class="g-card-body">
        <!-- 切换来源时重建组件，重新从头拉取对应日志 -->
        <LogViewer :key="url" :url="url" :clearable="source !== 'system'" height="calc(100vh - 290px)" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ScrollText } from 'lucide-vue-next'
import PageHeader from '../components/PageHeader.vue'
import LogViewer from '../components/LogViewer.vue'
import { navItems } from '../nav'

const route = useRoute()
const router = useRouter()

// 来源：系统日志 + 每个后端模块的日志
const sources = [{ label: '系统', value: 'system' }, ...navItems.filter((i) => i.moduleId).map((i) => ({ label: i.title, value: i.moduleId as string }))]
const initial = String(route.query.source ?? 'system')
const source = ref(sources.some((s) => s.value === initial) ? initial : 'system')
const url = computed(() => (source.value === 'system' ? '/api/logs' : `/api/${source.value}/logs`))

// 来源写入地址栏，刷新或分享链接时保持选择
watch(source, (v) => router.replace({ query: v === 'system' ? {} : { source: v } }))
</script>

<style scoped>
.sources {
  max-width: 100%;
  overflow-x: auto;
}
@media (max-width: 767.98px) {
  .g-card-body {
    padding: 12px;
  }
}
</style>
