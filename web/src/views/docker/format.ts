// Docker 界面的格式化与小工具；字节格式化复用 api.ts 中的 formatBytes。
import { formatBytes } from '../../api'

export { formatBytes }

// 把秒级时间戳、毫秒级时间戳或 RFC3339 字符串统一格式化为本地时间。
export function formatTime(t: number | string | undefined): string {
  if (t === undefined || t === '' || t === 0) {
    return '-'
  }
  let d: Date
  if (typeof t === 'number') {
    d = new Date(t < 1e12 ? t * 1000 : t)
  } else {
    d = new Date(t)
  }
  if (isNaN(d.getTime()) || d.getFullYear() < 1971) {
    return '-'
  }
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// 容器卷大小可能为 -1 表示未知。
export function formatSize(n: number | undefined): string {
  if (n === undefined || n < 0) {
    return '未知'
  }
  return formatBytes(n)
}

// 从 "8080→80/tcp" 形式的端口文本取出宿主机端口，没有映射时返回 0。
export function publicPort(p: string): number {
  const m = /^(\d+)→/.exec(p)
  return m ? Number(m[1]) : 0
}

// 只有 tcp 端口才适合用浏览器打开。
export function isTcpPort(p: string): boolean {
  return p.endsWith('/tcp') && publicPort(p) > 0
}

// 在新窗口打开 http://当前主机:端口。
export function openPort(p: string) {
  const port = publicPort(p)
  if (port > 0) {
    window.open(`http://${location.hostname}:${port}`, '_blank', 'noopener')
  }
}

// 触发浏览器下载（同源 Cookie 鉴权，直接使用链接即可）。
export function download(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

// 截取镜像 id 的短形式。
export function shortId(id: string): string {
  return id.replace(/^sha256:/, '').slice(0, 12)
}

// 把多行 KEY=VALUE 文本解析为数组，忽略空行与 # 注释。
export function parseEnvText(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l !== '' && !l.startsWith('#') && l.includes('='))
}

// 带进度的上传：用 XHR 直接发送 File（浏览器流式读取，不会整体载入内存）。
export function uploadFile(url: string, file: File, contentType: string, onProgress?: (percent: number) => void): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)
    xhr.withCredentials = true
    xhr.setRequestHeader('Content-Type', contentType)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    }
    xhr.onerror = () => reject(new Error('网络错误，上传失败'))
    xhr.onload = () => {
      let data: Record<string, unknown> = {}
      try {
        data = xhr.responseText ? JSON.parse(xhr.responseText) : {}
      } catch {
        data = {}
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(data)
        return
      }
      reject(new Error((data.error as string) ?? `请求失败 (${xhr.status})`))
    }
    xhr.send(file)
  })
}

// 容器状态到徽标的映射。
export function containerBadge(state: string): { type: 'running' | 'off' | 'warn' | 'pending' | 'error' | 'info'; text: string } {
  const map: Record<string, { type: 'running' | 'off' | 'warn' | 'pending' | 'error' | 'info'; text: string }> = {
    running: { type: 'running', text: '运行中' },
    exited: { type: 'off', text: '已停止' },
    created: { type: 'off', text: '已创建' },
    paused: { type: 'warn', text: '已暂停' },
    restarting: { type: 'pending', text: '重启中' },
    dead: { type: 'error', text: '异常' },
    removing: { type: 'pending', text: '删除中' },
  }
  return map[state] ?? { type: 'info', text: state }
}
