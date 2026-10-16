<template>
  <!-- 设置页底部的保存栏：有未保存修改时高亮并吸附在底部 -->
  <div class="save-bar" :class="{ dirty }">
    <div class="sb-text">
      <template v-if="dirty"><CircleDot :size="14" />有未保存的修改</template>
      <template v-else><CircleCheck :size="14" />所有修改已保存</template>
    </div>
    <div class="sb-ops">
      <el-button :disabled="!dirty" @click="$emit('reset')">撤销</el-button>
      <el-button type="primary" :icon="Save" :disabled="!dirty" :loading="saving" @click="$emit('save')">保存设置</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleCheck, CircleDot, Save } from 'lucide-vue-next'

defineProps<{ dirty: boolean; saving?: boolean }>()
defineEmits<{ save: []; reset: [] }>()
</script>

<style scoped>
.save-bar {
  position: sticky;
  bottom: 12px;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
  padding: 10px 12px 10px 16px;
  border-radius: var(--g-radius);
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
}
.save-bar.dirty {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 8px 24px -12px rgba(14, 124, 102, 0.45);
}
.sb-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.dirty .sb-text {
  color: var(--el-color-primary);
  font-weight: 500;
}
.sb-ops {
  display: flex;
  gap: 8px;
}
.sb-ops .el-button + .el-button {
  margin-left: 0;
}
@media (max-width: 767.98px) {
  .save-bar {
    flex-wrap: wrap;
    bottom: 8px;
  }
  .sb-ops {
    width: 100%;
  }
  .sb-ops .el-button {
    flex: 1;
  }
}
</style>
