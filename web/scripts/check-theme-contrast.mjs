// 校验界面主题配色的对比度（WCAG 2.x 相对亮度算法），每个色系的亮色和暗色调色板分别检查。
// 用法：node scripts/check-theme-contrast.mjs（需要 Node 22.18+ / 23.6+ 原生支持直接导入 .ts）
import { themeFamilies } from '../src/themes.ts'

// 检查项：[说明, 前景, 背景, 最低对比度]
// 正文与次要文字按 AA 级正文标准 4.5；主色作为链接、图标、描边等界面元素至少 3；主按钮上的白字至少 3
function checks(p) {
  return [
    ['正文/卡片', p.text, p.card, 4.5],
    ['正文/页面', p.text, p.page, 4.5],
    ['次要文字/卡片', p.textSub, p.card, 4.5],
    ['次要文字/页面', p.textSub, p.page, 4.5],
    ['主色/卡片', p.primary, p.card, 3],
    ['白字/主色按钮', '#ffffff', p.primary, 3],
  ]
}

function luminance(hex) {
  const h = hex.replace('#', '')
  const [r, g, b] = [0, 2, 4].map((i) => {
    const c = parseInt(h.slice(i, i + 2), 16) / 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contrast(a, b) {
  const [x, y] = [luminance(a), luminance(b)].sort((m, n) => n - m)
  return (x + 0.05) / (y + 0.05)
}

let errors = 0
let total = 0
for (const f of themeFamilies) {
  for (const mode of ['light', 'dark']) {
    const p = f[mode]
    if (!p) {
      continue
    }
    for (const [name, fg, bg, min] of checks(p)) {
      total++
      const r = contrast(fg, bg)
      if (r < min) {
        errors++
        console.error(`${f.id}/${mode} ${name}: ${r.toFixed(2)} < ${min}  (${fg} on ${bg})`)
      }
    }
  }
}
if (errors) {
  console.error(`共 ${total} 项检查，${errors} 项对比度不足`)
  process.exitCode = 1
} else {
  console.log(`共 ${total} 项检查，对比度全部达标`)
}
