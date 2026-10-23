<template>
  <el-popover
    trigger="click"
    :width="width"
    placement="bottom-end"
    popper-class="theme-popper"
    @before-enter="onOpen"
    @before-leave="onClose"
  >
    <template #reference>
      <button class="picker-btn" :aria-label="t('components.themePicker.switchTheme')" :title="t('components.themePicker.switchTheme')">
        <Palette :size="18" />
      </button>
    </template>
    <div class="picker">
      <div class="head">
        <span class="picker-title">{{ t('components.themePicker.title') }}</span>
        <!-- 亮暗模式：亮色 / 暗色 / 跟随系统 -->
        <div class="modes" role="radiogroup" :aria-label="t('components.themePicker.mode')">
          <button
            v-for="m in modes"
            :key="m.id"
            class="mode"
            role="radio"
            :aria-checked="mode === m.id"
            :class="{ active: mode === m.id }"
            :title="t(m.key)"
            @click="setMode(m.id, $event.clientX, $event.clientY)"
          >
            <component :is="m.icon" :size="14" />
            <span>{{ t(m.key) }}</span>
          </button>
        </div>
      </div>

      <div class="groups" @mouseleave="endHover">
        <section v-for="g in groups" :key="g.id" class="group">
          <div class="group-title">{{ t(g.key) }}</div>
          <div class="grid">
            <button
              v-for="f in g.items"
              :key="f.id"
              class="swatch"
              :class="{ active: family === f.id }"
              :style="swatchVars(f)"
              :aria-pressed="family === f.id"
              :aria-label="familyName(f)"
              @pointerenter="startHover(f.id, $event)"
              @focus="hoverId = f.id"
              @click="setFamily(f.id, $event.clientX, $event.clientY)"
            >
              <!-- 迷你预览：侧栏 + 顶栏 + 卡片 + 主色/第二色 -->
              <span class="preview">
                <span class="pv-side" />
                <span class="pv-main">
                  <span class="pv-bar" />
                  <span class="pv-card"><i class="a" /><i class="b" /></span>
                </span>
                <span v-if="singleMode(f)" class="pv-only">
                  <Moon v-if="singleMode(f) === 'dark'" :size="10" />
                  <Sun v-else :size="10" />
                </span>
                <span v-if="family === f.id" class="pv-check"><Check :size="11" :stroke-width="3" /></span>
              </span>
              <span class="sw-name">{{ familyName(f) }}</span>
            </button>
          </div>
        </section>
      </div>

      <!-- 底部说明：显示悬停或当前色系的描述 -->
      <div class="foot">
        <span class="foot-name">{{ familyName(info) }}</span>
        <span class="foot-desc">{{ t(info.descKey) }}</span>
        <span v-if="infoOnly" class="foot-only">{{ t(infoOnly === 'dark' ? 'components.themePicker.darkOnly' : 'components.themePicker.lightOnly') }}</span>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, MonitorSmartphone, Moon, Palette, Sun } from 'lucide-vue-next'
import { findFamily, resolveTheme, singleMode, themeFamilies, type ThemeFamily, type ThemeMode } from '../themes'
import { useTheme } from '../composables/useTheme'
import { useScrollLock } from '../composables/useScrollLock'

const { t } = useI18n()
const { family, mode, wantDark, setFamily, setMode, setPreview } = useTheme()
// 面板打开期间锁定页面滚动，滚轮和触摸滑动不穿透到下方页面
const lockScroll = useScrollLock('.theme-popper')

const modes: { id: ThemeMode; key: string; icon: typeof Sun }[] = [
  { id: 'light', key: 'components.themePicker.modeLight', icon: Sun },
  { id: 'dark', key: 'components.themePicker.modeDark', icon: Moon },
  { id: 'auto', key: 'components.themePicker.modeAuto', icon: MonitorSmartphone },
]

const groups = [
  { id: 'original', key: 'components.themePicker.groupOriginal', items: themeFamilies.filter((f) => f.group === 'original') },
  { id: 'classic', key: 'components.themePicker.groupClassic', items: themeFamilies.filter((f) => f.group === 'classic') },
]

// 窄屏下面板宽度不超出视口
const width = ref(372)
function onOpen() {
  width.value = Math.min(372, window.innerWidth - 32)
  lockScroll(true)
}
function onClose() {
  endHover()
  lockScroll(false)
}

// hoverId 驱动底部说明，立即更新；整页预览稍作延迟，避免鼠标划过网格时界面连续闪动
const hoverId = ref<string | null>(null)
let hoverTimer: ReturnType<typeof setTimeout> | undefined
function startHover(id: string, e: PointerEvent) {
  hoverId.value = id
  // 触屏没有悬停，点按即选择，不做预览
  if (e.pointerType !== 'mouse') {
    return
  }
  clearTimeout(hoverTimer)
  hoverTimer = setTimeout(() => setPreview(id), 140)
}
function endHover() {
  clearTimeout(hoverTimer)
  hoverId.value = null
  setPreview(null)
}
onBeforeUnmount(endHover)

const info = computed(() => findFamily(hoverId.value ?? family.value))
const infoOnly = computed(() => singleMode(info.value))

function familyName(f: ThemeFamily) {
  return f.label ?? t(f.nameKey!)
}

// swatchVars 按当前亮暗模式取色系的调色板，供迷你预览使用
function swatchVars(f: ThemeFamily) {
  const p = resolveTheme(f, wantDark.value)
  return {
    '--p': p.primary,
    '--c2': p.chart2,
    '--bg': p.page,
    '--card': p.card,
    '--side': p.sidebar,
    '--bd': p.border,
    '--tx': p.text,
    '--r': Math.min(p.radius, 6) + 'px',
  }
}
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
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.picker-title {
  font-weight: 600;
  font-size: 14px;
  white-space: nowrap;
}
/* 分段控件 */
.modes {
  display: inline-flex;
  padding: 2px;
  gap: 2px;
  border-radius: 8px;
  background: var(--el-fill-color);
  min-width: 0;
}
.mode {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition:
    background-color 0.15s,
    color 0.15s;
}
.mode:hover {
  color: var(--el-text-color-primary);
}
.mode.active {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}
.group + .group {
  margin-top: 10px;
}
.group-title {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.3px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}
.swatch {
  padding: 4px 4px 5px;
  border: 1.5px solid transparent;
  border-radius: var(--g-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  transition:
    border-color 0.15s,
    background-color 0.15s;
}
.swatch:hover,
.swatch:focus-visible {
  background: var(--el-fill-color-light);
  outline: none;
}
.swatch.active {
  border-color: var(--el-color-primary);
}
.preview {
  position: relative;
  display: flex;
  height: 38px;
  border-radius: var(--r);
  overflow: hidden;
  border: 1px solid var(--bd);
  background: var(--bg);
}
.pv-side {
  width: 12px;
  background: var(--side);
  border-right: 1px solid var(--bd);
}
.pv-main {
  flex: 1;
  padding: 4px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.pv-bar {
  height: 4px;
  width: 55%;
  border-radius: 2px;
  background: var(--p);
}
.pv-card {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  padding: 2px;
  background: var(--card);
  border: 1px solid var(--bd);
  border-radius: calc(var(--r) - 1px);
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
/* 只有单一模式的色系在左下角标出 */
.pv-only {
  position: absolute;
  left: 1px;
  bottom: 1px;
  display: inline-flex;
  color: var(--tx);
  opacity: 0.75;
}
.pv-check {
  position: absolute;
  right: 3px;
  top: 3px;
  width: 15px;
  height: 15px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--p);
  color: #fff;
}
.sw-name {
  font-size: 12px;
  line-height: 1.2;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.swatch.active .sw-name {
  font-weight: 600;
  color: var(--el-color-primary);
}
.foot {
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid var(--g-border);
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 2px 8px;
  min-height: 20px;
  font-size: 12px;
}
.foot-name {
  font-weight: 600;
}
.foot-desc {
  color: var(--el-text-color-secondary);
}
.foot-only {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  padding: 0 6px;
  border-radius: 4px;
  background: var(--el-fill-color);
}
/* 窄屏：分段控件只显示图标 */
@media (max-width: 400px) {
  .mode span {
    display: none;
  }
}
</style>
