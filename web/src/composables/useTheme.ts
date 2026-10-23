// 换肤：色系与亮暗模式分别保存在 localStorage；模式为“跟随系统”时按 prefers-color-scheme 取色系的亮色或暗色调色板。
import { computed, ref, watch } from 'vue'
import {
  DEFAULT_FAMILY,
  buildThemeVars,
  customizeTheme,
  findFamily,
  isHexColor,
  legacyTheme,
  logoSvg,
  resolveTheme,
  themeFamilies,
  type RadiusPref,
  type ThemeDef,
  type ThemeMode,
} from '../themes'

const familyKey = 'gately-theme-family'
const modeKey = 'gately-theme-mode'
const legacyKey = 'gately-theme' // 旧版只保存单一主题 ID
const accentKey = 'gately-theme-accent'
const radiusKey = 'gately-theme-radius'

// family/mode 是用户的选择；preview 是选择面板里悬停预览的色系，为 null 时不预览
const family = ref<string>(DEFAULT_FAMILY)
const mode = ref<ThemeMode>('auto')
const preview = ref<string | null>(null)
// accent 为自定义主色（null 表示用色系自带主色），radius 为圆角偏好
const accent = ref<string | null>(null)
const radius = ref<RadiusPref>('')
const systemDark = ref(false)
const wantDark = computed(() => (mode.value === 'auto' ? systemDark.value : mode.value === 'dark'))
// 悬停预览展示色系原本的主色，便于比较；圆角偏好始终生效
const current = computed<ThemeDef>(() =>
  customizeTheme(resolveTheme(findFamily(preview.value ?? family.value), wantDark.value), preview.value ? null : accent.value, radius.value),
)
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

// load 读取保存的选择；新键不存在时从旧版的单一主题 ID 迁移。
function load() {
  const savedAccent = localStorage.getItem(accentKey)
  if (isHexColor(savedAccent)) {
    accent.value = savedAccent
  }
  const savedRadius = localStorage.getItem(radiusKey)
  if (savedRadius === 'sharp' || savedRadius === 'round') {
    radius.value = savedRadius
  }
  const savedFamily = localStorage.getItem(familyKey)
  const savedMode = localStorage.getItem(modeKey)
  if (savedFamily || savedMode) {
    if (savedFamily && themeFamilies.some((f) => f.id === savedFamily)) {
      family.value = savedFamily
    }
    if (savedMode === 'light' || savedMode === 'dark' || savedMode === 'auto') {
      mode.value = savedMode
    }
    return
  }
  const legacy = legacyTheme(localStorage.getItem(legacyKey) ?? '')
  if (legacy) {
    family.value = legacy.family
    mode.value = legacy.mode
    localStorage.setItem(familyKey, legacy.family)
    localStorage.setItem(modeKey, legacy.mode)
    localStorage.removeItem(legacyKey)
  }
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
  })
  load()
  apply()
  // 生效主题的任何来源（选择、预览、系统亮暗）变化都同步刷新；flush: 'sync' 让视图过渡回调里的修改立即落到 DOM
  watch(current, apply, { flush: 'sync' })
}

// withReveal 用视图过渡从 (x, y) 处以圆形扩散展开新主题；浏览器不支持或用户偏好减少动效时直接切换。
function withReveal(update: () => void, x?: number, y?: number) {
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (!document.startViewTransition || reduce || x === undefined || y === undefined) {
    update()
    return
  }
  const root = document.documentElement
  root.classList.add('theme-vt')
  const vt = document.startViewTransition(update)
  const r = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y))
  vt.ready
    .then(() => {
      root.animate(
        { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${r}px at ${x}px ${y}px)`] },
        { duration: 450, easing: 'ease-in-out', pseudoElement: '::view-transition-new(root)' },
      )
    })
    .catch(() => {})
  vt.finished.finally(() => root.classList.remove('theme-vt'))
}

export function useTheme() {
  install()
  // setFamily 选定色系，同时清除自定义主色（选色系即选用它的整套配色）；传入点击坐标时带圆形过渡
  function setFamily(id: string, x?: number, y?: number) {
    // 悬停预览已经展示了该色系，此时无需再过渡
    const shown = preview.value === id && !accent.value
    const update = () => {
      preview.value = null
      family.value = id
      accent.value = null
    }
    localStorage.setItem(familyKey, id)
    localStorage.removeItem(accentKey)
    if (shown || (id === family.value && !accent.value)) {
      update()
      return
    }
    withReveal(update, x, y)
  }
  function setMode(m: ThemeMode, x?: number, y?: number) {
    if (m === mode.value) {
      return
    }
    localStorage.setItem(modeKey, m)
    withReveal(() => {
      mode.value = m
    }, x, y)
  }
  // setAccent 设置自定义主色，传 null 恢复色系自带主色
  function setAccent(c: string | null) {
    accent.value = isHexColor(c) ? c.toLowerCase() : null
    if (accent.value) {
      localStorage.setItem(accentKey, accent.value)
    } else {
      localStorage.removeItem(accentKey)
    }
  }
  function setRadius(r: RadiusPref) {
    radius.value = r
    if (r) {
      localStorage.setItem(radiusKey, r)
    } else {
      localStorage.removeItem(radiusKey)
    }
  }
  // setPreview 临时展示某个色系，传 null 恢复用户的选择
  function setPreview(id: string | null) {
    preview.value = id
  }
  return { family, mode, accent, radius, wantDark, current, isDark, setFamily, setMode, setAccent, setRadius, setPreview }
}

// initTheme 在应用挂载前调用，避免首屏闪烁。
export function initTheme() {
  install()
}
