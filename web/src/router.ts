import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  // 使用 hash 路由，避免反向代理子路径部署时刷新 404
  history: createWebHashHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('./views/Login.vue'), meta: { title: '登录' } },
    {
      path: '/',
      component: () => import('./views/Layout.vue'),
      redirect: '/overview',
      children: [
        { path: 'overview', name: 'overview', component: () => import('./views/Overview.vue'), meta: { title: '概览' } },
        { path: 'ddns', name: 'ddns', component: () => import('./views/DDNS.vue'), meta: { title: '动态域名' } },
        { path: 'stun', name: 'stun', component: () => import('./views/STUN.vue'), meta: { title: 'STUN 穿透' } },
        { path: 'tailscale', name: 'tailscale', component: () => import('./views/Tailscale.vue'), meta: { title: 'Tailscale' } },
        { path: 'frp', name: 'frp', component: () => import('./views/Frp.vue'), meta: { title: 'frp' } },
        { path: 'cloudflared', name: 'cloudflared', component: () => import('./views/Cloudflared.vue'), meta: { title: 'Cloudflare 隧道' } },
        { path: 'ssl', name: 'ssl', component: () => import('./views/SSL.vue'), meta: { title: 'SSL 证书' } },
        { path: 'web', name: 'web', component: () => import('./views/Web.vue'), meta: { title: 'Web 服务' } },
        { path: 'portforward', name: 'portforward', component: () => import('./views/PortForward.vue'), meta: { title: '端口转发' } },
        { path: 'waf', name: 'waf', component: () => import('./views/WAF.vue'), meta: { title: 'WAF 防火墙' } },
        { path: 'wol', name: 'wol', component: () => import('./views/WOL.vue'), meta: { title: '网络唤醒' } },
        { path: 'terminal', name: 'terminal', component: () => import('./views/Terminal.vue'), meta: { title: 'Web 终端' } },
        { path: 'docker', name: 'docker', component: () => import('./views/Docker.vue'), meta: { title: 'Docker' } },
        { path: 'webdav', name: 'webdav', component: () => import('./views/WebDAV.vue'), meta: { title: 'WebDAV' } },
        { path: 'storage', name: 'storage', component: () => import('./views/Storage.vue'), meta: { title: '存储管理' } },
        { path: 'ftp', name: 'ftp', component: () => import('./views/FTP.vue'), meta: { title: 'FTP 服务' } },
        { path: 'cron', name: 'cron', component: () => import('./views/Cron.vue'), meta: { title: '计划任务' } },
        { path: 'nettools', name: 'nettools', component: () => import('./views/NetTools.vue'), meta: { title: '网络诊断' } },
        { path: 'notify', name: 'notify', component: () => import('./views/Notify.vue'), meta: { title: '通知中心' } },
        { path: 'logs', name: 'logs', component: () => import('./views/Logs.vue'), meta: { title: '系统日志' } },
        { path: 'api-tokens', name: 'api-tokens', component: () => import('./views/ApiTokens.vue'), meta: { title: 'API 令牌' } },
        { path: 'api-docs', name: 'api-docs', component: () => import('./views/ApiDocs.vue'), meta: { title: 'API 文档' } },
        { path: 'settings', name: 'settings', component: () => import('./views/Settings.vue'), meta: { title: '设置' } },
        { path: 'about', name: 'about', component: () => import('./views/About.vue'), meta: { title: '关于' } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ],
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} - Gately` : 'Gately'
})

export default router
