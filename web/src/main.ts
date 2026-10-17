import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import { i18n, setupI18n } from './i18n'
import { initTheme } from './composables/useTheme'
import './style.css'

// 挂载前应用主题，避免首屏亮暗闪烁
initTheme()

// 先加载界面语言文案，再挂载，避免首屏出现未翻译的键名
setupI18n().then(() => {
  createApp(App).use(i18n).use(router).use(ElementPlus).mount('#app')
})
