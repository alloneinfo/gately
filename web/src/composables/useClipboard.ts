// 剪贴板辅助：优先使用 Clipboard API；非安全上下文（如 HTTP 访问）下浏览器不提供该接口，
// 或权限被拒绝时，降级为临时 textarea + execCommand('copy')；仍失败时选中页面上的文本供手动复制。
import { ElMessage } from 'element-plus'
import { t } from '../i18n'

// selectNode 选中节点内的文本，便于用户手动复制。
export function selectNode(el?: HTMLElement | null) {
  if (!el) {
    return
  }
  const range = document.createRange()
  range.selectNodeContents(el)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
}

// execCopy 借助临时 textarea 执行同步复制命令，HTTP 页面也可用。
function execCopy(text: string): boolean {
  const active = document.activeElement as HTMLElement | null
  const sel = window.getSelection()
  const ranges: Range[] = []
  if (sel) {
    for (let i = 0; i < sel.rangeCount; i++) {
      ranges.push(sel.getRangeAt(i))
    }
  }
  const ta = document.createElement('textarea')
  ta.value = text
  // readonly 避免移动端弹出键盘；12pt 避免 iOS 聚焦时缩放页面
  ta.setAttribute('readonly', '')
  ta.style.cssText = 'position:fixed;top:0;left:0;width:1px;height:1px;padding:0;border:0;opacity:0;pointer-events:none;font-size:12pt'
  // 对话框会把焦点限制在自身内部，挂到焦点所在的对话框里，否则 textarea 拿不到焦点、复制失败
  const host = active?.closest('[role="dialog"]') ?? document.body
  host.appendChild(ta)
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
  // 还原原来的焦点和选区，避免打断用户正在进行的操作
  active?.focus?.({ preventScroll: true })
  if (sel) {
    sel.removeAllRanges()
    for (const r of ranges) {
      sel.addRange(r)
    }
  }
  return ok
}

// writeClipboard 把文本写入剪贴板，返回是否成功，不弹提示。
export async function writeClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // 权限被拒绝或文档未获得焦点，继续走降级路径
    }
  }
  return execCopy(text)
}

export interface CopyOptions {
  // 页面上展示该文本的节点；自动复制失败时选中它，用户按 Ctrl/Cmd+C 即可
  el?: HTMLElement | null
  // 成功提示，默认「已复制」
  success?: string
}

// copyText 复制文本并提示结果。
export async function copyText(text: string, opts: CopyOptions = {}): Promise<boolean> {
  const ok = await writeClipboard(text)
  if (ok) {
    ElMessage.success(opts.success ?? t('common.copied'))
  } else if (opts.el) {
    selectNode(opts.el)
    ElMessage.warning(t('components.clipboard.copyFailed'))
  } else {
    ElMessage.warning(t('components.clipboard.copyUnavailable'))
  }
  return ok
}
