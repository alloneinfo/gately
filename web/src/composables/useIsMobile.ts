// 移动端断点检测：基于 matchMedia，全局共享一个响应式状态，避免每个组件各自监听。
import { computed, ref } from 'vue'

const query = '(max-width: 767.98px)'
const isMobile = ref(false)
let installed = false

function install() {
  if (installed || typeof window === 'undefined') {
    return
  }
  installed = true
  const mql = window.matchMedia(query)
  isMobile.value = mql.matches
  mql.addEventListener('change', (e) => {
    isMobile.value = e.matches
  })
}

// useIsMobile 返回是否为窄屏（< 768px）。
export function useIsMobile() {
  install()
  return isMobile
}

// useFormLayout 返回表单与对话框在当前屏幕下的推荐布局参数。
export function useFormLayout() {
  const mobile = useIsMobile()
  return {
    isMobile: mobile,
    // 移动端标签放在输入框上方，桌面端放在左侧
    labelPosition: computed(() => (mobile.value ? 'top' : 'right') as 'top' | 'right'),
  }
}
