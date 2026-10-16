import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import HomeLanding from './components/HomeLanding.vue'
import DownloadList from './components/DownloadList.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('HomeLanding', HomeLanding)
    app.component('DownloadList', DownloadList)
  }
} satisfies Theme
