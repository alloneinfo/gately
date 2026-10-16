// 侧边栏导航与模块元信息：Layout 与总览页共用，保证图标与名称一致。
import type { Component } from 'vue'
import {
  ArrowLeftRight,
  CalendarClock,
  Cloud,
  Container,
  FolderSync,
  HardDrive,
  Globe,
  Info,
  LayoutDashboard,
  Network,
  Power,
  Radar,
  ScrollText,
  Server,
  Settings,
  ShieldCheck,
  ShieldHalf,
  SquareTerminal,
  Waypoints,
} from 'lucide-vue-next'

export interface NavItem {
  path: string
  title: string
  icon: Component
  // 对应后端模块 ID（/api/status 中的 modules[].id）
  moduleId?: string
  // 总览页卡片上的一句话说明
  desc?: string
}

export interface NavGroup {
  title: string
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
  {
    title: '',
    items: [{ path: '/overview', title: '概览', icon: LayoutDashboard }],
  },
  {
    title: '网络',
    items: [
      { path: '/portforward', title: '端口转发', icon: ArrowLeftRight, moduleId: 'portforward', desc: 'TCP / UDP 端口转发与负载均衡' },
      { path: '/ddns', title: '动态域名', icon: Globe, moduleId: 'ddns', desc: '公网 IP 变化时自动更新解析' },
      { path: '/stun', title: 'STUN 穿透', icon: Radar, moduleId: 'stun', desc: 'NAT1 网络下无公网 IP 穿透' },
    ],
  },
  {
    title: '组网与穿透',
    items: [
      { path: '/tailscale', title: 'Tailscale', icon: Network, moduleId: 'tailscale', desc: '加入 Tailscale / Headscale 组网' },
      { path: '/frp', title: 'frp', icon: Waypoints, moduleId: 'frp', desc: 'frpc / frps 内网穿透' },
      { path: '/cloudflared', title: 'Cloudflare 隧道', icon: Cloud, moduleId: 'cloudflared', desc: '无需公网 IP 发布内网服务' },
    ],
  },
  {
    title: 'Web',
    items: [
      { path: '/web', title: 'Web 服务', icon: Server, moduleId: 'webservice', desc: '反向代理、重定向与静态站点' },
      { path: '/waf', title: 'WAF 防火墙', icon: ShieldHalf, moduleId: 'waf', desc: '基于 OWASP CRS 的应用防火墙' },
      { path: '/ssl', title: 'SSL 证书', icon: ShieldCheck, moduleId: 'ssl', desc: 'ACME 自动申请与续签' },
    ],
  },
  {
    title: '网络存储',
    items: [
      { path: '/storage', title: '存储管理', icon: HardDrive, moduleId: 'storage', desc: '挂载本地目录或远程 WebDAV，在线管理文件' },
      { path: '/webdav', title: 'WebDAV', icon: FolderSync, moduleId: 'webdav', desc: 'WebDAV 文件共享服务' },
      { path: '/ftp', title: 'FTP 服务', icon: Server, moduleId: 'ftp', desc: 'FTP / FTPS 文件共享服务' },
    ],
  },
  {
    title: '工具',
    items: [
      { path: '/wol', title: '网络唤醒', icon: Power, moduleId: 'wol', desc: '远程唤醒局域网设备' },
      { path: '/terminal', title: 'Web 终端', icon: SquareTerminal, moduleId: 'webterminal', desc: '浏览器中的 SSH 与本机 Shell' },
      { path: '/docker', title: 'Docker', icon: Container, moduleId: 'docker', desc: '容器管理与镜像更新' },
      { path: '/cron', title: '计划任务', icon: CalendarClock, moduleId: 'cron', desc: '定时执行命令或 HTTP 请求' },
    ],
  },
  {
    title: '系统',
    items: [
      { path: '/logs', title: '系统日志', icon: ScrollText },
      { path: '/settings', title: '设置', icon: Settings },
      { path: '/about', title: '关于', icon: Info },
    ],
  },
]

export const navItems: NavItem[] = navGroups.flatMap((g) => g.items)

// navByModule 按模块 ID 查找导航项。
export function navByModule(id: string): NavItem | undefined {
  return navItems.find((i) => i.moduleId === id)
}

// groupByPath 返回路由所在的导航分组名称。
export function groupByPath(path: string): string {
  return navGroups.find((g) => g.items.some((i) => path === i.path || path.startsWith(i.path + '/')))?.title ?? ''
}

// navByPath 按路由路径查找导航项。
export function navByPath(path: string): NavItem | undefined {
  return navItems.find((i) => path === i.path || path.startsWith(i.path + '/'))
}
