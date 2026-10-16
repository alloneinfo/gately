// 管理 API 封装：会话通过 HttpOnly Cookie 维持，401 时跳转登录页。
import router from './router'

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: Record<string, unknown>,
  ) {
    super(message)
  }
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, credentials: 'same-origin', headers: {} }
  if (body instanceof FormData) {
    init.body = body
  } else if (body !== undefined) {
    ;(init.headers as Record<string, string>)['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }
  const resp = await fetch(url, init)
  const text = await resp.text()
  const data = text ? JSON.parse(text) : null
  if (resp.status === 401 && url !== '/api/login') {
    if (router.currentRoute.value.name !== 'login') {
      router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    }
    throw new ApiError(data?.error ?? '未登录', 401, data)
  }
  if (!resp.ok) {
    throw new ApiError(data?.error ?? `请求失败 (${resp.status})`, resp.status, data)
  }
  return data as T
}

export const api = {
  get: <T>(url: string) => request<T>('GET', url),
  post: <T>(url: string, body?: unknown) => request<T>('POST', url, body),
  put: <T>(url: string, body?: unknown) => request<T>('PUT', url, body),
  // DELETE 允许携带 JSON 请求体（如批量删除文件）
  del: <T>(url: string, body?: unknown) => request<T>('DELETE', url, body),
}

export interface ModuleInfo {
  id: string
  title: string
  summary?: { label: string; value: string; level?: string }[]
}

export interface Status {
  version: string
  commit: string
  buildDate: string
  goVersion: string
  os: string
  arch: string
  startTime: string
  goroutines: number
  memAlloc: number
  modules: ModuleInfo[]
  timeZone: string
  configDir: string
  host: HostStats
}

// HostStats 对应 /api/status 中的 host：主机与进程资源占用，不支持的字段缺省。
export interface HostStats {
  hostname: string
  os: string
  platform?: string
  kernel?: string
  arch: string
  inContainer: boolean
  bootTime?: number
  cpu: { model?: string; cores: number; usage?: number; load?: number[] }
  memory?: { total: number; used: number; available: number; swapTotal: number; swapUsed: number }
  disk?: { path: string; total: number; used: number; free: number }
  interfaces: NetInterface[]
  process: {
    pid: number
    rss?: number
    cpu?: number
    threads?: number
    fds?: number
    goroutines: number
    heapAlloc: number
    heapSys: number
    numGC: number
  }
  sampledAt: string
}

export interface NetInterface {
  name: string
  mac?: string
  mtu: number
  ipv4: string[]
  ipv6: string[]
  rxBytes: number
  txBytes: number
  rxRate?: number
  txRate?: number
}

export interface LogEntry {
  seq: number
  time: string
  level: 'debug' | 'info' | 'warn' | 'error'
  source?: string
  message: string
}

export interface ImportReport {
  module: string
  title: string
  found: number
  imported: number
  warnings?: string[]
}

export interface ImportFileResult {
  file: string
  error?: string
  reports: ImportReport[]
}

// SystemSettings 对应 GET/PUT /api/system/settings。
// OAuthProvider 是第三方登录方式（Web 服务站点与管理界面共用）。
export interface OAuthProvider {
  key: string
  name: string
  type: string
  clientId: string
  clientSecret: string
  issuer: string
}

// ThirdLogin 是管理界面的第三方登录设置。
export interface ThirdLogin {
  enable: boolean
  callbackBase: string
  providers: OAuthProvider[]
  allowed: string[]
  skipTotp: boolean
}

export interface SystemSettings {
  listen: string
  httpsListen: string
  forceHTTPS: boolean
  safePath: string
  lanOnly: boolean
  maxLoginFailures: number
  lockMinutes: number
  // 已设置时为 ********，空字符串表示未启用
  apiToken: string
  timeZone: string
  dnsServers: string[] | null
  logLines: number
  gcPercent: number
  maxProcs: number
  thirdLogin: ThirdLogin
  sessionHours: number
  restartRequired: boolean
}

// SystemInfo 对应 GET /api/system/info。
export interface SystemInfo {
  version: string
  commit: string
  buildDate: string
  goVersion: string
  os: string
  arch: string
  hostname: string
  pid: number
  startTime: string
  configDir: string
  goroutines: number
  memSys: number
  memAlloc: number
  numCPU: number
  maxProcs: number
  timeZone: string
  inDocker: boolean
}

// UpdateInfo 对应 GET /api/system/update。
export interface UpdateInfo {
  current: string
  latest: string
  url: string
  notes: string
  publishedAt: string
  hasUpdate: boolean
}

// formatBytes 把字节数格式化为可读字符串。
export function formatBytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}

// formatRate 把字节/秒格式化为速率字符串。
export function formatRate(n: number | undefined): string {
  if (n === undefined) {
    return '-'
  }
  return formatBytes(Math.round(n)) + '/s'
}

// formatDuration 把秒数格式化为“x 天 x 小时”这类简短时长。
export function formatDuration(sec: number): string {
  let s = Math.max(0, Math.floor(sec))
  const d = Math.floor(s / 86400)
  s %= 86400
  const h = Math.floor(s / 3600)
  s %= 3600
  const m = Math.floor(s / 60)
  if (d > 0) {
    return `${d} 天 ${h} 小时`
  }
  if (h > 0) {
    return `${h} 小时 ${m} 分`
  }
  return `${m} 分钟`
}
