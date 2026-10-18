<template>
  <!-- 列表项卡片：桌面与移动端共用一套模板，字段区按宽度自动换列 -->
  <div class="entity" :class="{ 'is-off': off, 'is-compact': compact }">
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
// off 为 true 时整体置灰，表示该项已停用；compact 为 true 时压缩成单行（仅桌面宽度生效）
defineProps<{ off?: boolean; compact?: boolean }>()
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
/* 紧凑视图：标题、状态、字段、操作排成一行，字段不显示标签，超长内容省略 */
@media (min-width: 768px) {
  .is-compact {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    column-gap: 14px;
    padding: 0 8px 0 14px;
    min-height: 40px;
  }
  .is-compact .entity-head {
    /* 名称、标签、状态同占一列，随卡片宽度加宽，避免名称被挤成省略号 */
    flex: 0 0 clamp(340px, 40%, 480px);
    min-width: 0;
    padding: 0;
    gap: 10px;
  }
  .is-compact .entity-title {
    flex-wrap: nowrap;
    font-size: 14px;
  }
  .is-compact .entity-title :deep(> span:first-child) {
    min-width: 8em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .is-compact .entity-title :deep(> *:not(:first-child)) {
    flex-shrink: 0;
    flex-wrap: nowrap;
    white-space: nowrap;
  }
  .is-compact .entity-sub {
    display: none;
  }
  .is-compact .entity-body {
    flex: 1 1 0;
    min-width: 0;
    display: flex;
    gap: 14px;
    padding: 0;
  }
  .is-compact .entity-body :deep(.field) {
    flex: 1 1 0;
    min-width: 0;
  }
  .is-compact .entity-body :deep(.field.wide) {
    flex: 2 1 0;
  }
  .is-compact .entity-body :deep(.field-label) {
    display: none;
  }
  .is-compact .entity-body :deep(.field-value) {
    font-size: 12px;
    line-height: 1.4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  /* 字段内的多行内容（端口、更新状态、次要说明）并入同一行 */
  .is-compact .entity-body :deep(.field-value > div) {
    display: inline;
    margin: 0 0 0 6px;
  }
  .is-compact .entity-body :deep(.field-value > div:first-child) {
    margin-left: 0;
  }
  .is-compact .entity-extra {
    flex: 1 0 100%;
    order: 9;
    padding: 0 0 6px;
  }
  .is-compact .entity-head + .entity-extra {
    padding-top: 0;
  }
  .is-compact .entity-foot {
    flex: 0 0 auto;
    padding: 0;
    min-height: 0;
    border-top: none;
    flex-wrap: nowrap;
  }
  .is-compact .entity-head + .entity-foot {
    margin-top: 0;
  }
  .is-compact .entity-meta {
    display: none;
  }
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
