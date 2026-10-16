// 剪贴板辅助：优先使用异步 Clipboard API，非安全上下文下回退到 execCommand。

export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // 权限被拒绝时继续使用回退方案
    }
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  ta.style.left = '-1000px'
  document.body.appendChild(ta)
  ta.focus()
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(ta)
  return ok
}

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
