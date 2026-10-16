<template>
  <!-- 列表项卡片：桌面与移动端共用一套模板，字段区按宽度自动换列 -->
  <div class="entity" :class="{ 'is-off': off }">
    <div class="entity-head">
      <div v-if="$slots.lead" class="entity-lead"><slot name="lead" /></div>
      <div class="entity-titles">
        <div class="entity-title"><slot name="title" /></div>
        <div v-if="$slots.sub" class="entity-sub"><slot name="sub" /></div>
      </div>
      <div v-if="$slots.status" class="entity-status"><slot name="status" /></div>
    </div>
    <div v-if="$slots.default" class="entity-body"><slot /></div>
    <div v-if="$slots.extra" class="entity-extra"><slot name="extra" /></div>
    <div v-if="$slots.actions || $slots.meta" class="entity-foot">
      <div class="entity-meta"><slot name="meta" /></div>
      <div class="entity-actions"><slot name="actions" /></div>
    </div>
  </div>
</template>

<script setup lang="ts">
// off 为 true 时整体置灰，表示该项已停用
defineProps<{ off?: boolean }>()
</script>

<style scoped>
.entity {
  background: var(--el-bg-color);
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
  transition: border-color 0.2s;
  min-width: 0;
}
.entity:hover {
  border-color: var(--el-border-color);
}
.entity-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px 0;
}
.entity-lead {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.entity-titles {
  flex: 1;
  min-width: 0;
}
.entity-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
  font-weight: 600;
  font-size: 15px;
  line-height: 1.4;
  word-break: break-all;
}
.entity-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.entity-status {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 45%;
}
.entity-body {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 12px 20px;
  padding: 14px 16px;
}
.entity-extra {
  padding: 0 16px 14px;
}
.entity-head + .entity-extra {
  padding-top: 14px;
}
.entity-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px 12px;
  padding: 4px 8px 4px 16px;
  border-top: 1px solid var(--g-border);
  min-height: 36px;
}
.entity-head + .entity-foot {
  margin-top: 14px;
}
.entity-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  min-width: 0;
}
.entity-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  margin-left: auto;
}
.entity-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.is-off .entity-title {
  color: var(--el-text-color-secondary);
}
.is-off .entity-body,
.is-off .entity-extra {
  opacity: 0.6;
}
@media (max-width: 767.98px) {
  .entity-head {
    padding: 12px 12px 0;
    gap: 10px;
    align-items: flex-start;
  }
  .entity-lead {
    padding-top: 2px;
  }
  .entity-body {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 12px;
    gap: 10px 12px;
  }
  .entity-extra {
    padding: 0 12px 12px;
  }
  .entity-foot {
    padding: 6px 8px 6px 12px;
  }
}
</style>
