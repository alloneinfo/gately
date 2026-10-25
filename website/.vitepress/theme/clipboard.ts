// 剪贴板辅助：优先使用 Clipboard API；非安全上下文（HTTP 预览等）下浏览器不提供该接口，
// 或权限被拒绝时，降级为临时 textarea + execCommand('copy')；仍失败时选中页面上的文本供手动复制。
// 是管理界面 web/src/composables/useClipboard.ts 的简化版：官网没有对话框，不需要把 textarea 挂进对话框，
// 也不还原原来的选区；官网是独立工程，无法直接引用。

function execCopy(text: string): boolean {
  const active = document.activeElement as HTMLElement | null
  const ta = document.createElement('textarea')
  ta.value = text
  // readonly 避免移动端弹出键盘；12pt 避免 iOS 聚焦时缩放页面
  ta.setAttribute('readonly', '')
  ta.style.cssText = 'position:fixed;top:0;left:0;width:1px;height:1px;padding:0;border:0;opacity:0;pointer-events:none;font-size:12pt'
  document.body.appendChild(ta)
  let ok = false
  try {
    ta.focus({ preventScroll: true })
    ta.select()
    // iOS Safari 只认 setSelectionRange
    ta.setSelectionRange(0, text.length)
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  ta.remove()
  active?.focus?.({ preventScroll: true })
  return ok
}

function selectNode(el: HTMLElement) {
  const range = document.createRange()
  range.selectNodeContents(el)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
}

// copyText 复制文本，返回是否成功；el 为页面上展示该文本的节点，失败时选中它供手动复制。
export async function copyText(text: string, el?: HTMLElement | null): Promise<boolean> {
  let ok = false
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      ok = true
    } catch {
      // 权限被拒绝或文档未获得焦点，继续走降级路径
    }
  }
  if (!ok) {
    ok = execCopy(text)
  }
  if (!ok && el) {
    selectNode(el)
  }
  return ok
}
