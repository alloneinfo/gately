// Web 终端前端共用的类型与数据规整函数，字段与 docs/webterminal-api.md 保持一致。

export interface ProxyConf {
  enabled: boolean
  type: 'socks5' | 'http'
  host: string
  port: number
  username: string
  password: string
}

export interface SSHConf {
  host: string
  port: number
  username: string
  authType: 'password' | 'key'
  password: string
  privateKey: string
  keyFile: string
  passphrase: string
  hostKey: string
  keepAlive: number
  workDir: string
  timeout: number
  proxy: ProxyConf
}

export interface TelnetConf {
  host: string
  port: number
  username: string
  password: string
  timeout: number
  proxy: ProxyConf
}

export interface LocalConf {
  shell: string
  workDir: string
  env: Record<string, string>
}

export type ConnType = 'ssh' | 'telnet' | 'local'

export interface Conn {
  key: string
  name: string
  type: ConnType
  remark: string
  enable: boolean
  quickAccessDirs: string[]
  ssh: SSHConf
  telnet: TelnetConf
  local: LocalConf
}

export interface SessionInfo {
  id: string
  connKey: string
  connName: string
  type: string
  remark: string
  createdAt: string | number
  lastActive: string | number
  attached: number
  cols: number
  rows: number
  bytesIn: number
  bytesOut: number
}

export interface SessionStats {
  bytesIn: number
  bytesOut: number
  durationSec: number
  idleSec: number
  attached: number
  cols: number
  rows: number
}

export interface Shortcut {
  name: string
  command: string
  sendEnter: boolean
}

export interface TermSettings {
  allowLocalShell: boolean
  maxSessions: number
  idleTimeout: number
  bufferSize: number
  heartbeatInterval: number
  sessionKeepAlive: number
}

export interface SecurityState {
  agreementAccepted: boolean
  twoFAEnable: boolean
  verified: boolean
}

export type PaneStatus = 'connecting' | 'connected' | 'reconnecting' | 'closed' | 'ended'

// TermTab 是一个终端标签（对应一个持久会话）。
export interface TermTab {
  id: string
  connKey: string
  connName: string
  type: string
  sessionId: string
  title: string
  remark: string
  status: PaneStatus
}

export type SplitMode = 'h' | 'v' | 'grid'

// SplitLayout 是保存到后端的分屏布局，后端原样存取。
export interface SplitLayout {
  version: 1
  mode: SplitMode
  items: { sessionId: string; title: string }[]
  active: string
}

export interface SftpEntry {
  name: string
  path: string
  isDir: boolean
  isLink: boolean
  size: number
  mode: number | string
  modTime: string | number
  owner?: string
}

export interface SftpList {
  path: string
  parent: string
  entries: SftpEntry[] | null
}

export const MASK = '********'

export function emptyProxy(): ProxyConf {
  return { enabled: false, type: 'socks5', host: '', port: 1080, username: '', password: '' }
}

export function emptyConn(): Conn {
  return {
    key: '',
    name: '',
    type: 'ssh',
    remark: '',
    enable: true,
    quickAccessDirs: [],
    ssh: {
      host: '',
      port: 22,
      username: 'root',
      authType: 'password',
      password: '',
      privateKey: '',
      keyFile: '',
      passphrase: '',
      hostKey: '',
      keepAlive: 60,
      workDir: '',
      timeout: 15,
      proxy: emptyProxy(),
    },
    telnet: { host: '', port: 23, username: '', password: '', timeout: 15, proxy: emptyProxy() },
    local: { shell: '', workDir: '', env: {} },
  }
}

// normalizeConn 把后端返回的连接补全默认值（后端可能省略空结构或返回 null）。
export function normalizeConn(raw: Partial<Conn> | null | undefined): Conn {
  const base = emptyConn()
  const r = (raw ?? {}) as Partial<Conn>
  return {
    ...base,
    ...r,
    quickAccessDirs: r.quickAccessDirs ? [...r.quickAccessDirs] : [],
    ssh: { ...base.ssh, ...(r.ssh ?? {}), proxy: { ...base.ssh.proxy, ...(r.ssh?.proxy ?? {}) } },
    telnet: { ...base.telnet, ...(r.telnet ?? {}), proxy: { ...base.telnet.proxy, ...(r.telnet?.proxy ?? {}) } },
    local: { ...base.local, ...(r.local ?? {}), env: { ...(r.local?.env ?? {}) } },
  }
}

// connAddr 返回连接的简短地址描述，用于列表展示。
export function connAddr(c: Conn): string {
  if (c.type === 'local') {
    return c.local.shell ? `本机 ${c.local.shell}` : '本机 Shell'
  }
  if (c.type === 'telnet') {
    return `${c.telnet.username ? c.telnet.username + '@' : ''}${c.telnet.host}:${c.telnet.port}`
  }
  return `${c.ssh.username}@${c.ssh.host}:${c.ssh.port}`
}

export const typeLabel: Record<string, string> = { ssh: 'SSH', telnet: 'Telnet', local: '本机' }

// fmtTime 把后端的时间（ISO 字符串或 Unix 秒/毫秒）格式化为本地时间文本。
export function fmtTime(v: string | number | undefined | null): string {
  if (v === undefined || v === null || v === '' || v === 0) {
    return '-'
  }
  let d: Date
  if (typeof v === 'number') {
    d = new Date(v < 1e12 ? v * 1000 : v)
  } else {
    d = new Date(v)
  }
  if (isNaN(d.getTime())) {
    return String(v)
  }
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
