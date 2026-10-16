// 亮/暗主题：优先使用用户在 localStorage 中的选择，否则跟随系统 prefers-color-scheme。
import { ref } from 'vue'

const storageKey = 'gately-theme'
const isDark = ref(false)
let installed = false

function apply() {
  document.documentElement.classList.toggle('dark', isDark.value)
  document.documentElement.style.colorScheme = isDark.value ? 'dark' : 'light'
}

function install() {
  if (installed || typeof window === 'undefined') {
    return
  }
  installed = true
  const saved = localStorage.getItem(storageKey)
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  isDark.value = saved ? saved === 'dark' : mql.matches
  // 用户未手动选择过时，跟随系统切换
  mql.addEventListener('change', (e) => {
    if (!localStorage.getItem(storageKey)) {
      isDark.value = e.matches
      apply()
    }
  })
  apply()
}

export function useTheme() {
  install()
  function toggle() {
    isDark.value = !isDark.value
    localStorage.setItem(storageKey, isDark.value ? 'dark' : 'light')
    apply()
  }
  return { isDark, toggle }
}

// initTheme 在应用挂载前调用，避免首屏闪烁。
export function initTheme() {
  install()
}
