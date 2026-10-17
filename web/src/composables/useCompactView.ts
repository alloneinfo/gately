// 列表“紧凑视图”开关：按页面键名持久化到 localStorage，刷新后保持上次选择。
import { ref, watch } from 'vue'

const prefix = 'gately.compact.'

export function useCompactView(key: string) {
  let initial = false
  try {
    initial = localStorage.getItem(prefix + key) === '1'
  } catch {
    // 隐私模式等场景下无法读取存储，使用默认值
  }
  const compact = ref(initial)
  watch(compact, (v) => {
    try {
      localStorage.setItem(prefix + key, v ? '1' : '0')
    } catch {
      // 写入失败时仅本次会话生效
    }
  })
  return compact
}
