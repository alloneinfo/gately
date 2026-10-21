<template>
  <!-- 编辑 string[] 的输入框：保留用户输入的原始文本，回车、逗号、空格都能正常输入；其余属性透传给 el-input -->
  <el-input v-model="text" />
</template>

<script setup lang="ts">
import { useListText, splitLines } from '../composables/useListText'

const props = withDefaults(
  defineProps<{
    modelValue?: string[] | null
    // 解析规则，默认按换行拆分
    split?: (text: string) => string[]
    // 由列表生成文本时使用的分隔符，默认换行
    separator?: string
  }>(),
  { modelValue: undefined, split: splitLines, separator: '\n' },
)
const emit = defineEmits<{ 'update:modelValue': [list: string[]] }>()

const text = useListText(
  () => props.modelValue,
  (list) => emit('update:modelValue', list),
  { split: props.split, separator: props.separator },
)
</script>
