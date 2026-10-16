<template>
  <el-popover trigger="click" :width="316" placement="bottom-end" popper-class="theme-popper">
    <template #reference>
      <button class="picker-btn" aria-label="切换主题" title="切换主题">
        <Palette :size="18" />
      </button>
    </template>
    <div class="picker">
      <div class="picker-title">主题配色</div>
      <div class="grid">
        <button
          v-for="t in themes"
          :key="t.id"
          class="swatch"
          :class="{ active: choice === t.id }"
          :style="{ '--p': t.primary, '--c2': t.chart2, '--bg': t.page, '--card': t.card, '--side': t.sidebar, '--bd': t.border, '--r': Math.min(t.radius, 6) + 'px' }"
          @click="setTheme(t.id)"
        >
          <span class="preview">
            <span class="pv-side" />
            <span class="pv-main">
              <span class="pv-bar" />
              <span class="pv-card"><i class="a" /><i class="b" /></span>
            </span>
          </span>
          <span class="sw-name">
            {{ t.name }}
            <Check v-if="choice === t.id" :size="13" class="sw-check" />
          </span>
          <span class="sw-desc">{{ t.desc }}</span>
        </button>
      </div>
      <button class="auto" :class="{ active: choice === AUTO_ID }" @click="setTheme(AUTO_ID)">
        <MonitorSmartphone :size="15" />
        跟随系统（翠绿 / 暗黑）
        <Check v-if="choice === AUTO_ID" :size="14" class="sw-check" />
      </button>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { Check, MonitorSmartphone, Palette } from 'lucide-vue-next'
import { AUTO_ID, themes } from '../themes'
import { useTheme } from '../composables/useTheme'

const { choice, setTheme } = useTheme()
</script>

<style scoped>
.picker-btn {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    color 0.15s,
    border-color 0.15s;
}
.picker-btn:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
}
.picker-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 10px;
}
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.swatch {
  text-align: left;
  padding: 8px;
  border: 1.5px solid var(--g-border);
  border-radius: var(--g-radius-sm);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 2px;
  transition: border-color 0.15s;
}
.swatch:hover {
  border-color: var(--el-color-primary-light-5);
}
.swatch.active {
  border-color: var(--el-color-primary);
}
/* 迷你预览：侧栏 + 顶栏 + 卡片 + 主色/第二色 */
.preview {
  display: flex;
  height: 46px;
  margin-bottom: 4px;
  border-radius: var(--r);
  overflow: hidden;
  border: 1px solid var(--bd);
  background: var(--bg);
}
.pv-side {
  width: 14px;
  background: var(--side);
  border-right: 1px solid var(--bd);
}
.pv-main {
  flex: 1;
  padding: 5px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.pv-bar {
  height: 5px;
  width: 55%;
  border-radius: 3px;
  background: var(--p);
}
.pv-card {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 3px;
  padding: 3px;
  background: var(--card);
  border: 1px solid var(--bd);
  border-radius: var(--r);
}
.pv-card i {
  flex: 1;
  border-radius: 1px;
}
.pv-card .a {
  height: 100%;
  background: var(--p);
}
.pv-card .b {
  height: 55%;
  background: var(--c2);
}
.sw-name {
  display: flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  font-size: 13px;
}
.sw-check {
  color: var(--el-color-primary);
}
.sw-desc {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}
.auto {
  width: 100%;
  margin-top: 8px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 13px;
  border: 1.5px dashed var(--g-border);
  border-radius: var(--g-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.auto.active {
  border-style: solid;
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
</style>
