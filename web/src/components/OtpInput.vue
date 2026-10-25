<template>
  <!-- 分格验证码输入：一个透明输入框覆盖在格子上，天然支持粘贴、系统短信/验证器自动填充与退格 -->
  <div class="otp" :class="{ error: shaking, disabled }" @click="focus">
    <input
      ref="inputRef"
      class="otp-native"
      :value="modelValue"
      :maxlength="length"
      :disabled="disabled"
      :aria-label="ariaLabel"
      inputmode="numeric"
      autocomplete="one-time-code"
      pattern="[0-9]*"
      spellcheck="false"
      @input="onInput"
      @focus="focused = true"
      @blur="focused = false"
    />
    <div v-for="i in length" :key="i" class="cell" :class="{ filled: i <= modelValue.length, active: focused && isActive(i) }">
      <span v-if="i <= modelValue.length" class="digit">{{ modelValue[i - 1] }}</span>
      <span v-else-if="focused && isActive(i)" class="caret" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(defineProps<{ modelValue: string; length?: number; disabled?: boolean; ariaLabel?: string }>(), {
  length: 6,
  disabled: false,
  ariaLabel: '',
})
const emit = defineEmits<{
  'update:modelValue': [value: string]
  complete: [value: string]
}>()

const inputRef = ref<HTMLInputElement>()
const focused = ref(false)
const shaking = ref(false)

// 当前光标所在格：已填满时停留在最后一格
function isActive(i: number) {
  return i === Math.min(props.modelValue.length + 1, props.length)
}

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  // 只保留数字，粘贴带空格或横杠的验证码也能识别
  const v = el.value.replace(/\D/g, '').slice(0, props.length)
  if (el.value !== v) {
    el.value = v
  }
  emit('update:modelValue', v)
  if (v.length === props.length) {
    emit('complete', v)
  }
}

function focus() {
  inputRef.value?.focus()
}

// 验证失败时抖动提示，由父组件调用
function shake() {
  shaking.value = false
  requestAnimationFrame(() => {
    shaking.value = true
    setTimeout(() => {
      shaking.value = false
    }, 450)
  })
}

defineExpose({ focus, shake })
</script>

<style scoped>
.otp {
  position: relative;
  display: grid;
  grid-template-columns: repeat(v-bind(length), 1fr);
  gap: 10px;
  cursor: text;
}
.otp.disabled {
  opacity: 0.6;
  pointer-events: none;
}
.otp-native {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  /* 保持 opacity: 1：Bitwarden 等密码管理器把 opacity 过低的输入框当作隐藏字段，不会自动填充；
     文字、背景、光标都设为透明，视觉上仍只显示下面的格子 */
  border: 0;
  padding: 0;
  font-size: 16px; /* 避免 iOS 聚焦时自动缩放 */
  color: transparent;
  -webkit-text-fill-color: transparent;
  background: transparent;
  caret-color: transparent;
  outline: none;
  forced-color-adjust: none; /* 高对比度模式下系统会强制文字颜色，透明会失效 */
  z-index: 1;
}
.otp-native::selection {
  background: transparent;
}
/* 浏览器原生自动填充会强制加底色和文字颜色（!important 无法覆盖），
   用超长的 transition 延迟让它们一直不生效，否则会在格子上盖出一块色块和重叠的数字。
   :autofill 在 Chrome 110+、Safari 15+、Firefox 86+ 可用 */
.otp-native:autofill {
  -webkit-text-fill-color: transparent;
  transition:
    background-color 0s 600000s,
    color 0s 600000s;
}
.cell {
  height: 56px;
  border-radius: 12px;
  border: 1.5px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    border-color 0.15s,
    box-shadow 0.15s,
    transform 0.15s;
}
.cell.filled {
  border-color: color-mix(in srgb, var(--el-color-primary) 45%, var(--el-border-color));
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-fill-color-blank));
}
.cell.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--el-color-primary) 18%, transparent);
}
.digit {
  font-family: var(--g-mono);
  font-size: 24px;
  font-weight: 600;
  animation: pop 0.16s ease-out;
}
.caret {
  width: 2px;
  height: 24px;
  border-radius: 1px;
  background: var(--el-color-primary);
  animation: blink 1s steps(1) infinite;
}
.otp.error .cell {
  border-color: var(--el-color-danger);
}
.otp.error {
  animation: shake 0.42s cubic-bezier(0.36, 0.07, 0.19, 0.97);
}
@keyframes blink {
  50% {
    opacity: 0;
  }
}
@keyframes pop {
  from {
    transform: scale(0.6);
    opacity: 0;
  }
}
@keyframes shake {
  20%,
  60% {
    transform: translateX(-6px);
  }
  40%,
  80% {
    transform: translateX(6px);
  }
}
@media (max-width: 420px) {
  .otp {
    gap: 7px;
  }
  .cell {
    height: 50px;
    border-radius: 10px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .digit,
  .caret,
  .otp.error {
    animation: none;
  }
}
</style>
