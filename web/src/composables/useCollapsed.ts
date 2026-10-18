// 列表分组“展开/收起”状态：按页面键名把已收起的分组 key 持久化到 localStorage，刷新后保持。
import { computed, ref, watch } from 'vue'

const prefix = 'gately.collapsed.'

export function useCollapsed(key: string) {
  let initial: string[] = []
  try {
    const raw = localStorage.getItem(prefix + key)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) {
        initial = parsed.filter((v) => typeof v === 'string')
      }
    }
  } catch {
    // 存储不可用或内容损坏时使用默认值（全部展开）
  }
  const collapsed = ref(new Set<string>(initial))
  watch(collapsed, (v) => {
    try {
      localStorage.setItem(prefix + key, JSON.stringify([...v]))
    } catch {
      // 写入失败时仅本次会话生效
    }
  })

  function isCollapsed(id: string) {
    return collapsed.value.has(id)
  }

  function setCollapsed(id: string, v: boolean) {
    const next = new Set(collapsed.value)
    if (v) {
      next.add(id)
    } else {
      next.delete(id)
    }
    collapsed.value = next
  }

  // 批量设置：ids 为当前页面全部分组；收起时顺带清理已不存在的分组 key
  function setAll(ids: string[], v: boolean) {
    collapsed.value = v ? new Set(ids) : new Set()
  }

  // 返回一个计算属性：给定分组 key 列表，判断是否全部已收起，供“全部展开/收起”按钮使用
  function allCollapsed(ids: () => string[]) {
    return computed(() => {
      const list = ids()
      return list.length > 0 && list.every((id) => collapsed.value.has(id))
    })
  }

  return { isCollapsed, setCollapsed, setAll, allCollapsed }
}
