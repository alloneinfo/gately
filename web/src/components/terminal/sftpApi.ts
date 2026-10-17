// SFTP 文件管理的接口封装与路径、权限辅助函数。
import { ApiError, api } from '../../api'
import { t } from '../../i18n'
import { withAccess } from './security'
import type { SftpEntry, SftpList } from './types'

export function sftpBase(key: string): string {
  return `/api/webterminal/sftp/${encodeURIComponent(key)}`
}

export function listDir(key: string, path: string): Promise<SftpList> {
  return withAccess(() => api.get<SftpList>(`${sftpBase(key)}/list?path=${encodeURIComponent(path)}`))
}

export function post<T = unknown>(key: string, action: string, body: unknown): Promise<T> {
  return withAccess(() => api.post<T>(`${sftpBase(key)}/${action}`, body))
}

export function remove(key: string, paths: string[]): Promise<unknown> {
  return withAccess(() => api.del(`${sftpBase(key)}/remove`, { paths }))
}

export function downloadUrl(key: string, path: string): string {
  return `${sftpBase(key)}/read?path=${encodeURIComponent(path)}&download=1`
}

// readText 读取文本预览；兼容后端直接返回文本或返回 {content} 两种形式。
export function readText(key: string, path: string): Promise<string> {
  return withAccess(async () => {
    const resp = await fetch(`${sftpBase(key)}/read?path=${encodeURIComponent(path)}`, { credentials: 'same-origin' })
    const text = await resp.text()
    let data: unknown = null
    try {
      data = text ? JSON.parse(text) : null
    } catch {
      data = null
    }
    if (!resp.ok) {
      const d = (data ?? {}) as Record<string, unknown>
      throw new ApiError(String(d.error ?? t('common.requestFailed', { status: resp.status })), resp.status, d)
    }
    if (data && typeof data === 'object' && typeof (data as { content?: unknown }).content === 'string') {
      return (data as { content: string }).content
    }
    return text
  })
}

// uploadForm 使用 XMLHttpRequest 上传，以便获得进度。
export function uploadForm(key: string, fd: FormData, onProgress: (percent: number) => void): Promise<void> {
  return withAccess(
    () =>
      new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest()
        xhr.open('POST', `${sftpBase(key)}/upload`)
        xhr.withCredentials = true
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            onProgress(Math.round((e.loaded / e.total) * 100))
          }
        }
        xhr.onload = () => {
          let data: Record<string, unknown> | undefined
          try {
            data = xhr.responseText ? JSON.parse(xhr.responseText) : undefined
          } catch {
            data = undefined
          }
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve()
          } else {
            reject(new ApiError(String(data?.error ?? t('sftp.api.uploadFailed', { status: xhr.status })), xhr.status, data))
          }
        }
        xhr.onerror = () => reject(new ApiError(t('sftp.api.networkError'), 0))
        xhr.send(fd)
      }),
  )
}

// ---- 路径辅助（按 POSIX 风格处理） ----
export function joinPath(dir: string, name: string): string {
  if (!dir || dir === '/') {
    return '/' + name
  }
  return dir.replace(/\/+$/, '') + '/' + name
}

export function dirName(p: string): string {
  const i = p.replace(/\/+$/, '').lastIndexOf('/')
  if (i <= 0) {
    return '/'
  }
  return p.slice(0, i)
}

export function baseName(p: string): string {
  const s = p.replace(/\/+$/, '')
  return s.slice(s.lastIndexOf('/') + 1)
}

// breadcrumbs 把路径拆成可点击的面包屑。
export function breadcrumbs(path: string): { name: string; path: string }[] {
  const parts = path.split('/').filter((s) => s)
  const out = [{ name: '/', path: '/' }]
  let cur = ''
  for (const p of parts) {
    cur += '/' + p
    out.push({ name: p, path: cur })
  }
  return out
}

export function isArchive(name: string): boolean {
  return /\.(zip|tar|tar\.gz|tgz)$/i.test(name)
}

// modeBits 提取权限位（后端可能返回数字、八进制字符串或 ls 风格字符串）。
export function modeBits(mode: number | string): number | null {
  if (typeof mode === 'number') {
    return mode & 0o777
  }
  if (/^[0-7]{3,4}$/.test(mode)) {
    return parseInt(mode, 8) & 0o777
  }
  const m = /^[-a-zA-Z]([-rwxsStT]{9})$/.exec(mode)
  if (m) {
    const s = m[1]
    let bits = 0
    for (let i = 0; i < 9; i++) {
      if (s[i] !== '-' && s[i] !== 'S' && s[i] !== 'T') {
        bits |= 1 << (8 - i)
      }
    }
    return bits
  }
  return null
}

export function modeOctal(mode: number | string): string {
  const b = modeBits(mode)
  return b === null ? '' : b.toString(8).padStart(3, '0')
}

// modeText 返回 rwxr-xr-x 形式的权限文本。
export function modeText(mode: number | string): string {
  const b = modeBits(mode)
  if (b === null) {
    return String(mode)
  }
  const chars = 'rwxrwxrwx'
  let out = ''
  for (let i = 0; i < 9; i++) {
    out += b & (1 << (8 - i)) ? chars[i] : '-'
  }
  return out
}

export function sortEntries(list: SftpEntry[], key: 'name' | 'size' | 'modTime', asc: boolean): SftpEntry[] {
  const k = asc ? 1 : -1
  const ts = (e: SftpEntry) => (typeof e.modTime === 'number' ? e.modTime : new Date(e.modTime).getTime() || 0)
  return [...list].sort((a, b) => {
    // 目录始终排在文件前面
    if (a.isDir !== b.isDir) {
      return a.isDir ? -1 : 1
    }
    if (key === 'size') {
      return (a.size - b.size) * k || a.name.localeCompare(b.name)
    }
    if (key === 'modTime') {
      return (ts(a) - ts(b)) * k || a.name.localeCompare(b.name)
    }
    return a.name.localeCompare(b.name, undefined, { numeric: true }) * k
  })
}
