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
        { path: 'overview', name: 'overview', component: () => import('./views/Overview.vue'), meta: { title: '总览' } },
        { path: 'ddns', name: 'ddns', component: () => import('./views/DDNS.vue'), meta: { title: '动态域名' } },
        { path: 'portforward', name: 'portforward', component: () => import('./views/PortForward.vue'), meta: { title: '端口转发' } },
        { path: 'settings', name: 'settings', component: () => import('./views/Settings.vue'), meta: { title: '设置' } },
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
