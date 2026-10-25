// 终端剪贴板辅助：写入复用 useClipboard 的实现（含 HTTP 下的降级），读取在非安全上下文下不可用。
export { writeClipboard as copyText } from '../../composables/useClipboard'

// readClipboardText 读取剪贴板文本，不可用时返回 null（由调用方改用手动粘贴框）。
export async function readClipboardText(): Promise<string | null> {
  if (navigator.clipboard && navigator.clipboard.readText && window.isSecureContext) {
    try {
      return await navigator.clipboard.readText()
    } catch {
      return null
    }
  }
  return null
}
