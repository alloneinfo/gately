// 主题定义：每个主题只声明少量基础色，其余 Element Plus 变量由 buildThemeVars 推导。

export interface ThemeDef {
  id: string
  nameKey: string // 主题名称文案键
  descKey: string // 主题描述文案键
  dark: boolean
  primary: string // 主色
  chart2: string // 图表第二色
  page: string // 页面背景
  card: string // 卡片/面板背景
  sidebar: string // 侧栏背景
  border: string // 边框
  text: string // 主文字
  textSub: string // 次要文字
  radius: number // 卡片圆角
  logo: [string, string, string] // 标志渐变起/止色与点缀色
  hero: [string, string, string] // 登录页左侧渐变三色
  fontMono?: boolean // 朋克风：标题使用等宽字体
}

export const themes: ThemeDef[] = [
  {
    id: 'emerald',
    nameKey: 'themes.emerald.name',
    descKey: 'themes.emerald.desc',
    dark: false,
    primary: '#0e7c66',
    chart2: '#6366f1',
    page: '#f4f6f8',
    card: '#ffffff',
    sidebar: '#ffffff',
    border: '#e6eaee',
    text: '#1f2937',
    textSub: '#6b7280',
    radius: 10,
    logo: ['#18a888', '#0b5f4f', '#9ff3dc'],
    hero: ['#0f8a71', '#0b5f4f', '#083f35'],
  },
  {
    id: 'ocean',
    nameKey: 'themes.ocean.name',
    descKey: 'themes.ocean.desc',
    dark: false,
    primary: '#0b73c9',
    chart2: '#14b8a6',
    page: '#eef5fb',
    card: '#ffffff',
    sidebar: '#f8fbfe',
    border: '#d9e6f2',
    text: '#12263a',
    textSub: '#5b7288',
    radius: 12,
    logo: ['#2a9df4', '#0a4f94', '#aee6ff'],
    hero: ['#1580d6', '#0a5aa6', '#06335f'],
  },
  {
    id: 'fresh',
    nameKey: 'themes.fresh.name',
    descKey: 'themes.fresh.desc',
    dark: false,
    primary: '#2fa84f',
    chart2: '#f59e0b',
    page: '#f3f9ee',
    card: '#ffffff',
    sidebar: '#fbfef8',
    border: '#e0eed5',
    text: '#233421',
    textSub: '#6a7f65',
    radius: 14,
    logo: ['#6fd36b', '#2a9a48', '#f7ffb0'],
    hero: ['#45bb5a', '#2a9a48', '#176b33'],
  },
  {
    id: 'sakura',
    nameKey: 'themes.sakura.name',
    descKey: 'themes.sakura.desc',
    dark: false,
    primary: '#d6457c',
    chart2: '#8b5cf6',
    page: '#fdf3f6',
    card: '#ffffff',
    sidebar: '#fffafb',
    border: '#f4dce4',
    text: '#3a1f2b',
    textSub: '#8f6b79',
    radius: 14,
    logo: ['#f27aa5', '#c23a6e', '#ffe0ec'],
    hero: ['#e0568a', '#be3a6c', '#7e2147'],
  },
  {
    id: 'sunset',
    nameKey: 'themes.sunset.name',
    descKey: 'themes.sunset.desc',
    dark: false,
    primary: '#d9640f',
    chart2: '#0e9aa7',
    page: '#fbf5ee',
    card: '#fffdfa',
    sidebar: '#fffaf4',
    border: '#efe1d2',
    text: '#3a2616',
    textSub: '#8a6f58',
    radius: 8,
    logo: ['#ff9a3c', '#c24a0a', '#ffe3b8'],
    hero: ['#e8791f', '#c24a0a', '#7a2b05'],
  },
  {
    id: 'dark',
    nameKey: 'themes.dark.name',
    descKey: 'themes.dark.desc',
    dark: true,
    primary: '#2bb596',
    chart2: '#818cf8',
    page: '#0f1113',
    card: '#16181b',
    sidebar: '#16181b',
    border: '#2a2e33',
    text: '#e5eaf0',
    textSub: '#9aa4b0',
    radius: 10,
    logo: ['#2bb596', '#0e7c66', '#a6f5e0'],
    hero: ['#12806a', '#0b5f4f', '#07352c'],
  },
  {
    id: 'midnight',
    nameKey: 'themes.midnight.name',
    descKey: 'themes.midnight.desc',
    dark: true,
    primary: '#7c8cff',
    chart2: '#38bdf8',
    page: '#0b1020',
    card: '#121830',
    sidebar: '#0f1528',
    border: '#232b4a',
    text: '#e4e8ff',
    textSub: '#8e98c4',
    radius: 12,
    logo: ['#8a9bff', '#4453d6', '#dfe4ff'],
    hero: ['#4a5ae0', '#2c389f', '#141b55'],
  },
  {
    id: 'punk',
    nameKey: 'themes.punk.name',
    descKey: 'themes.punk.desc',
    dark: true,
    primary: '#ff2e97',
    chart2: '#00e5ff',
    page: '#0a0612',
    card: '#130a22',
    sidebar: '#0e0719',
    border: '#3b1d63',
    text: '#f5e9ff',
    textSub: '#a58bc4',
    radius: 2,
    logo: ['#ff2e97', '#7a1fd1', '#00e5ff'],
    hero: ['#c4157a', '#5a1597', '#1a0838'],
    fontMono: true,
  },
]

export const AUTO_ID = 'auto'

export function findTheme(id: string): ThemeDef {
  return themes.find((t) => t.id === id) ?? themes[0]
}

function parseHex(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

// mix 把颜色 a 与 b 按比例混合，t 为 b 的占比（0~1）。
function mix(a: string, b: string, t: number): string {
  const ca = parseHex(a)
  const cb = parseHex(b)
  const out = ca.map((v, i) => Math.round(v * (1 - t) + cb[i] * t))
  return '#' + out.map((v) => v.toString(16).padStart(2, '0')).join('')
}

// buildThemeVars 由基础色推导出全部需要覆盖的 CSS 变量。
export function buildThemeVars(t: ThemeDef): Record<string, string> {
  // 亮色主题的浅阶向白色混合，暗色主题的浅阶向卡片底色混合
  const base = t.dark ? t.card : '#ffffff'
  const lightStep = (n: number) => mix(t.primary, base, t.dark ? 1 - n / 10 : n / 10)
  const fill = (r: number) => mix(t.card, t.text, r)
  const v: Record<string, string> = {
    '--gately-accent': t.primary,
    '--g-radius': `${t.radius}px`,
    '--g-radius-sm': `${Math.max(2, Math.round(t.radius * 0.8))}px`,
    '--g-border': t.border,
    '--g-sidebar-bg': t.sidebar,
    '--g-header-bg': t.sidebar + 'd9',
    '--g-chart-2': t.chart2,
    '--g-logo-from': t.logo[0],
    '--g-logo-to': t.logo[1],
    '--g-logo-dot': t.logo[2],
    '--g-hero-1': t.hero[0],
    '--g-hero-2': t.hero[1],
    '--g-hero-3': t.hero[2],
    '--el-color-primary': t.primary,
    '--el-color-primary-light-3': lightStep(3),
    '--el-color-primary-light-5': lightStep(5),
    '--el-color-primary-light-7': lightStep(7),
    '--el-color-primary-light-8': lightStep(8),
    '--el-color-primary-light-9': t.dark ? mix(t.primary, t.card, 0.86) : mix(t.primary, '#ffffff', 0.9),
    '--el-color-primary-dark-2': t.dark ? mix(t.primary, '#ffffff', 0.2) : mix(t.primary, '#000000', 0.2),
    '--el-bg-color-page': t.page,
    '--el-bg-color': t.card,
    '--el-bg-color-overlay': t.dark ? mix(t.card, '#ffffff', 0.04) : t.card,
    '--el-border-radius-base': `${Math.max(2, Math.round(t.radius * 0.6))}px`,
    '--el-border-color': mix(t.border, t.text, 0.12),
    '--el-border-color-light': mix(t.border, t.card, 0.4),
    '--el-border-color-lighter': mix(t.border, t.card, 0.6),
    '--el-border-color-extra-light': mix(t.border, t.card, 0.75),
    '--el-text-color-primary': t.text,
    '--el-text-color-regular': mix(t.text, t.card, 0.18),
    '--el-text-color-secondary': t.textSub,
    '--el-text-color-placeholder': mix(t.textSub, t.card, 0.4),
    '--el-text-color-disabled': mix(t.textSub, t.card, 0.6),
    '--el-fill-color': fill(0.08),
    '--el-fill-color-light': fill(0.05),
    '--el-fill-color-lighter': fill(0.03),
    '--el-fill-color-extra-light': fill(0.02),
    '--el-fill-color-blank': t.card,
    '--el-mask-color': t.dark ? 'rgba(0, 0, 0, 0.7)' : 'rgba(255, 255, 255, 0.85)',
  }
  // 朋克风格 Element 的悬浮/次级状态色保持霓虹一致
  if (t.id === 'punk') {
    v['--el-color-success'] = '#00f5a0'
    v['--el-color-warning'] = '#ffd23f'
    v['--el-color-danger'] = '#ff4d5e'
  }
  return v
}

// logoSvg 生成与 AppLogo.vue 一致的标志，用于动态网站图标。
export function logoSvg(t: ThemeDef): string {
  return (
    `<svg width="64" height="64" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">` +
    `<defs><linearGradient id="g" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">` +
    `<stop offset="0" stop-color="${t.logo[0]}"/><stop offset="1" stop-color="${t.logo[1]}"/></linearGradient></defs>` +
    `<rect width="32" height="32" rx="${t.id === 'punk' ? 4 : 9}" fill="url(#g)"/>` +
    `<path d="M21.7 10.3A8 8 0 1 0 24 16h-7" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>` +
    `<circle cx="23.6" cy="8.6" r="2.2" fill="${t.logo[2]}"/></svg>`
  )
}
