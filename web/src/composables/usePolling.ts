import { onBeforeUnmount } from 'vue'

// usePolling 按固定间隔调用 fn，供页面定时刷新数据：
// - 标签页切到后台时暂停，切回前台时立即刷新一次再继续，后台标签不再持续请求；
// - 上一次调用尚未结束时跳过本轮，网络慢时请求不会堆积；
// - 组件卸载时自动停止。
// 需在 setup 中同步调用；通常在首次加载完成后调用 start()。
export function usePolling(fn: () => unknown, interval: number) {
  let timer: number | undefined
  let running = false
  let started = false
  // 组件已卸载；首次加载是异步的，卸载后才调用 start 时不再开始轮询
  let disposed = false

  async function tick() {
    if (running || document.hidden) {
      return
    }
    running = true
    try {
      await fn()
    } catch {
      // 轮询失败由下一轮重试，错误提示交给 fn 自己处理
    } finally {
      running = false
    }
  }

  function onVisibility() {
    if (!started) {
      return
    }
    if (document.hidden) {
      window.clearInterval(timer)
      timer = undefined
      return
    }
    if (timer === undefined) {
      tick()
      timer = window.setInterval(tick, interval)
    }
  }

  // start 开始轮询（不会立即调用 fn，首次加载由调用方负责）；重复调用无副作用
  function start() {
    if (started || disposed) {
      return
    }
    started = true
    document.addEventListener('visibilitychange', onVisibility)
    if (!document.hidden) {
      timer = window.setInterval(tick, interval)
    }
  }

  function stop() {
    started = false
    window.clearInterval(timer)
    timer = undefined
    document.removeEventListener('visibilitychange', onVisibility)
  }

  onBeforeUnmount(() => {
    disposed = true
    stop()
  })
  return { start, stop }
}
