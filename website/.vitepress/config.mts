import { defineConfig } from 'vitepress'

const SITE = 'https://gately.cn'
const REPO = 'https://github.com/wangwanjie/gately'
const DESC = '开源、可审计的网络网关工具集：DDNS、STUN 穿透、端口转发、反向代理、证书、WAF、Tailscale、frp 集成在一个单文件中，受 Lucky 启发的开源实现。'

export default defineConfig({
  lang: 'zh-CN',
  title: 'Gately',
  titleTemplate: ':title - Gately',
  description: DESC,
  cleanUrls: true,
  lastUpdated: true,
  sitemap: { hostname: SITE },
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#0e7c66' }],
    ['meta', { name: 'keywords', content: 'Gately,Lucky 替代,DDNS,STUN,内网穿透,端口转发,反向代理,ACME,WAF,Tailscale,frp,NAS' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'Gately - 开源可审计的网络网关工具集' }],
    ['meta', { property: 'og:description', content: DESC }],
    ['meta', { property: 'og:url', content: SITE }]
  ],
  themeConfig: {
    logo: '/favicon.svg',
    siteTitle: 'Gately',
    nav: [
      { text: '首页', link: '/' },
      { text: '文档', link: '/docs/guide/introduction', activeMatch: '/docs/' },
      { text: '功能模块', link: '/docs/modules/', activeMatch: '/docs/modules/' },
      { text: '下载', link: '/download' },
      { text: '对比 Lucky', link: '/docs/guide/comparison' },
      { text: '更新日志', link: `${REPO}/releases` }
    ],
    sidebar: {
      '/docs/': [
        {
          text: '开始使用',
          items: [
            { text: '项目简介', link: '/docs/guide/introduction' },
            { text: '快速开始', link: '/docs/guide/quick-start' },
            { text: '安装部署', link: '/docs/guide/install' },
            { text: '配置与运行参数', link: '/docs/guide/configuration' },
            { text: '命令行与 MCP', link: '/docs/guide/cli-mcp' },
            { text: '从 Lucky 迁移', link: '/docs/guide/migration' },
            { text: '与 Lucky 对比', link: '/docs/guide/comparison' },
            { text: '常见问题', link: '/docs/guide/faq' }
          ]
        },
        {
          text: '功能模块',
          items: [
            { text: '模块总览', link: '/docs/modules/' },
            { text: '动态域名 DDNS', link: '/docs/modules/ddns' },
            { text: 'STUN 穿透', link: '/docs/modules/stun' },
            { text: '端口转发', link: '/docs/modules/portforward' },
            { text: 'SSL 证书', link: '/docs/modules/ssl' },
            { text: 'Web 服务', link: '/docs/modules/web' },
            { text: 'WAF 防火墙', link: '/docs/modules/waf' },
            { text: 'Tailscale / frp / cloudflared', link: '/docs/modules/tunnels' },
            { text: 'Docker 管理', link: '/docs/modules/docker' },
            { text: 'Web 终端', link: '/docs/modules/terminal' },
            { text: '网络诊断', link: '/docs/modules/nettools' },
            { text: '通知中心', link: '/docs/modules/notify' },
            { text: '配置备份', link: '/docs/modules/backup' },
            { text: '文件与运维工具', link: '/docs/modules/tools' }
          ]
        },
        {
          text: '安全',
          items: [
            { text: '安全模型与隐私承诺', link: '/docs/guide/security' },
            { text: '加固建议', link: '/docs/guide/hardening' }
          ]
        },
        {
          text: '参考',
          items: [
            { text: 'API 令牌与 OpenAPI', link: '/docs/reference/api' },
            { text: '接口清单', link: '/docs/reference/api-reference' },
            { text: 'Docker API', link: '/docs/reference/docker-api' },
            { text: 'Web 终端 API', link: '/docs/reference/webterminal-api' },
            { text: '参与贡献', link: '/docs/guide/contributing' }
          ]
        }
      ]
    },
    socialLinks: [{ icon: 'github', link: REPO }],
    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
          modal: {
            noResultsText: '没有找到相关结果',
            resetButtonTitle: '清除查询',
            footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' }
          }
        }
      }
    },
    editLink: {
      pattern: `${REPO}/edit/main/website/:path`,
      text: '在 GitHub 上编辑此页'
    },
    lastUpdated: { text: '最后更新' },
    outline: { label: '本页目录', level: [2, 3] },
    docFooter: { prev: '上一页', next: '下一页' },
    darkModeSwitchLabel: '外观',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '回到顶部',
    langMenuLabel: '语言',
    footer: {
      message: '以 Apache-2.0 许可证开源发布。Gately 与 Lucky 项目方无任何隶属关系。',
      copyright: `© 2026 Gately Contributors · <a href="${REPO}" target="_blank" rel="noreferrer">GitHub</a>`
    }
  }
})
