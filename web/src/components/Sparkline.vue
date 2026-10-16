<template>
  <!-- 迷你折线面积图：多条序列共用一个纵轴，用于展示最近一段时间的趋势 -->
  <svg class="spark" :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" :style="{ height: height + 'px' }">
    <g v-for="(s, i) in paths" :key="i">
      <!-- 颜色用 style 设置，才能使用 CSS 变量跟随主题 -->
      <path :d="s.area" :style="{ fill: s.color }" fill-opacity="0.12" />
      <path :d="s.line" :style="{ stroke: s.color }" stroke-width="1.6" fill="none" vector-effect="non-scaling-stroke" stroke-linejoin="round" />
    </g>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ series: { values: number[]; color: string }[]; height?: number; capacity?: number; max?: number }>(), {
  height: 48,
  capacity: 60,
  max: undefined,
})

const W = 300
const H = 100

const paths = computed(() => {
  const all = props.series.flatMap((s) => s.values)
  // 纵轴上限：指定值优先，否则取序列最大值并留出顶部余量
  const top = props.max ?? Math.max(1, ...all) * 1.15
  const step = W / Math.max(1, props.capacity - 1)
  return props.series.map((s) => {
    // 数据靠右对齐，新数据从右侧进入
    const offset = props.capacity - s.values.length
    const pts = s.values.map((v, i) => [(offset + i) * step, H - (Math.min(v, top) / top) * (H - 4) - 2])
    // 少于两个点无法成线，等下一次采样
    if (pts.length < 2) {
      return { line: '', area: '', color: s.color }
    }
    const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ')
    const area = `${line} L${pts[pts.length - 1][0].toFixed(1)},${H} L${pts[0][0].toFixed(1)},${H} Z`
    return { line, area, color: s.color }
  })
})
</script>

<style scoped>
.spark {
  display: block;
  width: 100%;
}
</style>
