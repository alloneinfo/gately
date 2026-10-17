<template>
  <!-- 令牌权限摘要：全部读写 / 全部只读 / 按模块标签，过多时折叠 -->
  <div class="scope-tags">
    <el-tag v-if="summary.kind === 'all-write'" size="small" type="danger" effect="light">全部读写</el-tag>
    <el-tag v-else-if="summary.kind === 'all-read'" size="small" type="warning" effect="light">全部只读</el-tag>
    <template v-else>
      <el-tag v-for="it in summary.items.slice(0, limit)" :key="it.id" size="small" :type="it.write ? 'primary' : 'info'" effect="light">
        {{ it.title }} {{ it.write ? '读写' : '只读' }}
      </el-tag>
      <el-tooltip v-if="summary.items.length > limit" placement="top">
        <template #content>
          <div class="more-list">
            <div v-for="it in summary.items.slice(limit)" :key="it.id">{{ it.title }} {{ it.write ? '读写' : '只读' }}</div>
          </div>
        </template>
        <el-tag size="small" type="info" effect="plain">+{{ summary.items.length - limit }}</el-tag>
      </el-tooltip>
      <span v-if="!summary.items.length" class="muted">无权限</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ scopes: string[]; titles: Record<string, string>; limit?: number }>(), { limit: 4 })

// 把 scopes 归并为“全部读写 / 全部只读 / 逐模块（写覆盖读）”
const summary = computed(() => {
  const list = props.scopes ?? []
  if (list.includes('*') || list.includes('*:write')) {
    return { kind: 'all-write' as const, items: [] }
  }
  if (list.includes('*:read')) {
    return { kind: 'all-read' as const, items: [] }
  }
  const map = new Map<string, boolean>()
  for (const s of list) {
    const idx = s.lastIndexOf(':')
    const id = idx > 0 ? s.slice(0, idx) : s
    const write = idx > 0 ? s.slice(idx + 1) === 'write' : true
    map.set(id, (map.get(id) ?? false) || write)
  }
  const items = [...map.entries()].map(([id, write]) => ({ id, write, title: props.titles[id] ?? id }))
  return { kind: 'list' as const, items }
})
</script>

<style scoped>
.scope-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}
.more-list {
  line-height: 1.7;
}
</style>
