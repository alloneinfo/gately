// 换肤：主题 ID 保存在 localStorage；选择“跟随系统”时按 prefers-color-scheme 在翠绿/暗黑间切换。
import { computed, ref } from 'vue'
import { AUTO_ID, buildThemeVars, findTheme, logoSvg, themes, type ThemeDef } from '../themes'

const storageKey = 'gately-theme'
// choice 是用户的选择（含 auto），current 是实际生效的主题
const choice = ref<string>(AUTO_ID)
const systemDark = ref(false)
const current = computed<ThemeDef>(() => {
  if (choice.value === AUTO_ID) {
    return findTheme(systemDark.value ? 'dark' : 'emerald')
  }
  return findTheme(choice.value)
})
const isDark = computed(() => current.value.dark)
let installed = false
// 记录上次写入的变量名，切换时清理，避免残留
let appliedKeys: string[] = []

function setLink(rel: string, attrs: Record<string, string>, selector: string) {
  let el = document.head.querySelector<HTMLLinkElement>(selector)
  if (!el) {
    el = document.createElement('link')
    el.rel = rel
    document.head.appendChild(el)
  }
  Object.entries(attrs).forEach(([k, val]) => el!.setAttribute(k, val))
}

function apply() {
  const t = current.value
  const root = document.documentElement
  appliedKeys.forEach((k) => root.style.removeProperty(k))
  const vars = buildThemeVars(t)
  appliedKeys = Object.keys(vars)
  appliedKeys.forEach((k) => root.style.setProperty(k, vars[k]))
  root.classList.toggle('dark', t.dark)
  root.dataset.theme = t.id
  root.style.colorScheme = t.dark ? 'dark' : 'light'

  // 浏览器地址栏颜色与网站图标随主题变化
  document.head.querySelector('meta[name="theme-color"]')?.setAttribute('content', t.dark ? t.page : t.primary)
  const href = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(logoSvg(t))
  setLink('icon', { href, type: 'image/svg+xml' }, 'link[rel="icon"][type="image/svg+xml"]')
}

function install() {
  if (installed || typeof window === 'undefined') {
    return
  }
  installed = true
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  systemDark.value = mql.matches
  mql.addEventListener('change', (e) => {
    systemDark.value = e.matches
    if (choice.value === AUTO_ID) {
      apply()
    }
  })
  const saved = localStorage.getItem(storageKey)
  if (saved === 'light') {
    // 兼容旧版本只有亮/暗两种的存储值
    choice.value = 'emerald'
  } else if (saved && (saved === AUTO_ID || themes.some((t) => t.id === saved))) {
    choice.value = saved
  }
  apply()
}

export function useTheme() {
  install()
  function setTheme(id: string) {
    choice.value = id
    localStorage.setItem(storageKey, id)
    apply()
  }
  return { choice, current, isDark, setTheme }
}

// initTheme 在应用挂载前调用，避免首屏闪烁。
export function initTheme() {
  install()
}
