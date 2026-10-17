import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import HomeLanding from './components/HomeLanding.vue'
import DownloadList from './components/DownloadList.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    // 用户在语言菜单里手动切换语言后记住选择，之后不再按浏览器语言自动跳转
    if (typeof document !== 'undefined') {
      document.addEventListener(
        'click',
        (e) => {
          const a = (e.target as HTMLElement | null)?.closest?.('.VPNavBarTranslations a, .VPNavScreenTranslations a') as HTMLAnchorElement | null
          if (!a) {
            return
          }
          const dir = new URL(a.href, location.href).pathname.split('/')[1] ?? ''
          const known = ['en', 'zh-TW', 'ja', 'ko', 'es', 'fr', 'de', 'ru', 'pt-BR']
          localStorage.setItem('gately.site.locale', known.includes(dir) ? dir : 'root')
        },
        true
      )
    }
    app.component('HomeLanding', HomeLanding)
    app.component('DownloadList', DownloadList)
  }
} satisfies Theme
