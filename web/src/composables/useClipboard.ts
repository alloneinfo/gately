// 剪贴板辅助：优先使用 Clipboard API，不可用时（如非 HTTPS）降级为选中文本 + execCommand。
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

// copyText 复制文本；el 为页面上展示该文本的节点，降级时会选中它。
export async function copyText(text: string, el?: HTMLElement | null) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('common.copied'))
    return
  } catch {
    // 剪贴板 API 被拒绝或不可用，继续走降级路径
  }
  let ok = false
  if (el) {
    // 展示节点存在时直接选中其文本再执行复制命令
    selectNode(el)
    try {
      ok = document.execCommand('copy')
    } catch {
      ok = false
    }
  } else {
    // 没有展示节点时借助临时 textarea
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      ok = document.execCommand('copy')
    } catch {
      ok = false
    }
    ta.remove()
  }
  if (ok) {
    ElMessage.success(t('common.copied'))
  } else {
    ElMessage.warning(t('components.clipboard.copyFailed'))
  }
}
