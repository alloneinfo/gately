import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import { initTheme } from './composables/useTheme'
import './style.css'

// 挂载前应用主题，避免首屏亮暗闪烁
initTheme()

createApp(App).use(router).use(ElementPlus, { locale: zhCn }).mount('#app')
