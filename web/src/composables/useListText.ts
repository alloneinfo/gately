// 列表字段（域名、地址、请求头等 string[]）与输入框文本的双向绑定。
//
// 若直接用 computed 的 get/set 往返 join/split，用户每敲一个键都会被"拆分 → 过滤空项 → 重新拼接"，
// 刚输入的回车、逗号、空格会立刻被吞掉，表现为"回车没反应、无法换行""逗号打不出来"。
// 这里让输入框保留用户的原始文本，只把解析后的列表写回数据；仅当数据被外部替换（打开另一条记录、
// 重新加载、重置等）时，才按列表重新生成文本。
import { computed, ref, toRaw, watch, type WritableComputedRef } from 'vue'

export interface ListTextOptions {
  // 把输入文本解析为列表（负责去空白、过滤空项）
  split: (text: string) => string[]
  // 由列表生成文本时使用的分隔符，默认换行
  separator?: string
}

// 按换行拆分
export const splitLines = (v: string) => v.split('\n').map((s) => s.trim()).filter(Boolean)
// 按换行或逗号拆分
export const splitLinesOrCommas = (v: string) => v.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
// 按逗号或任意空白拆分
export const splitCommasOrSpaces = (v: string) => v.split(/[,\s]+/).filter(Boolean)

const sameList = (a: string[], b: string[]) => a.length === b.length && a.every((s, i) => s === b[i])

export function useListText(
  source: () => string[] | null | undefined,
  assign: (list: string[]) => void,
  options: ListTextOptions,
): WritableComputedRef<string> {
  const { split, separator = '\n' } = options
  const raw = ref('')
  // 最近一次由输入框写回的列表，用来区分"本输入框自己写入"与"外部替换"
  let lastWritten: string[] | undefined

  watch(
    source,
    (list) => {
      const items = list ?? []
      // 数据正是本输入框刚写回的列表，且与当前文本解析结果一致：保留用户原始文本（含空行、行尾分隔符）
      if (toRaw(list ?? undefined) === lastWritten && sameList(split(raw.value), items)) {
        return
      }
      lastWritten = undefined
      raw.value = items.join(separator)
    },
    { immediate: true, deep: true },
  )

  return computed({
    get: () => raw.value,
    set: (v: string) => {
      raw.value = v
      const list = split(v)
      lastWritten = list
      assign(list)
    },
  })
}
