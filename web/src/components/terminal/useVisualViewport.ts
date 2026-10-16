// 跟踪可视视口（visualViewport），软键盘弹出时可据此调整终端区域高度。
import { onBeforeUnmount, onMounted, ref } from 'vue'

export function useVisualViewport() {
  const height = ref(0)
  const top = ref(0)
  const supported = typeof window !== 'undefined' && !!window.visualViewport

  function update() {
    const vv = window.visualViewport
    if (!vv) {
      return
    }
    height.value = Math.round(vv.height)
    top.value = Math.round(vv.offsetTop)
  }

  onMounted(() => {
    update()
    window.visualViewport?.addEventListener('resize', update)
    window.visualViewport?.addEventListener('scroll', update)
  })
  onBeforeUnmount(() => {
    window.visualViewport?.removeEventListener('resize', update)
    window.visualViewport?.removeEventListener('scroll', update)
  })
  return { height, top, supported }
}
