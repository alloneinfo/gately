// 子路径部署支持：经飞牛 fnOS 统一网关访问时，页面位于 /app/gately/ 之类的前缀下，
// 后端会在 index.html 中注入 <meta name="gately-base">。这里读取该前缀，
// 并给发往本站的绝对路径请求（fetch、XHR、WebSocket、下载链接、新窗口）统一补上前缀。
// 未注入前缀时（Docker、二进制等直接访问）不做任何改动。

/** 路径前缀，例如 /app/gately；直接访问时为空字符串。 */
export const basePath = (document.querySelector('meta[name="gately-base"]')?.getAttribute('content') ?? '').replace(/\/+$/, '')

/** 管理 API 的完整地址前缀（协议 + 主机 + 路径前缀），用于展示与复制。 */
export const apiOrigin = window.location.origin + basePath

/** 给以 / 开头的站内路径补上前缀；已带前缀、协议相对地址（//host）与其他形式保持不变。 */
export function withBase(url: string): string {
  if (!basePath || !url.startsWith('/') || url.startsWith('//')) {
    return url
  }
  if (url === basePath || url.startsWith(basePath + '/')) {
    return url
  }
  return basePath + url
}

// 给指向本站的完整 URL（如 WebSocket 地址）补上前缀
function withBaseAbsolute(url: string): string {
  try {
    const u = new URL(url)
    if (u.host !== window.location.host) {
      return url
    }
    const path = withBase(u.pathname)
    if (path === u.pathname) {
      return url
    }
    u.pathname = path
    return u.toString()
  } catch {
    return url
  }
}

function rewrite(url: string | URL): string | URL {
  if (typeof url !== 'string') {
    return url instanceof URL ? new URL(withBaseAbsolute(url.toString())) : url
  }
  return url.startsWith('/') ? withBase(url) : withBaseAbsolute(url)
}

function install() {
  const origFetch = window.fetch.bind(window)
  window.fetch = (input: RequestInfo | URL, init?: RequestInit) => {
    if (typeof input === 'string' || input instanceof URL) {
      return origFetch(rewrite(input), init)
    }
    return origFetch(input, init)
  }

  const origOpen = XMLHttpRequest.prototype.open
  XMLHttpRequest.prototype.open = function (this: XMLHttpRequest, method: string, url: string | URL, ...rest: unknown[]) {
    return (origOpen as (...args: unknown[]) => void).call(this, method, rewrite(url), ...rest)
  } as typeof XMLHttpRequest.prototype.open

  const OrigWebSocket = window.WebSocket
  window.WebSocket = class extends OrigWebSocket {
    constructor(url: string | URL, protocols?: string | string[]) {
      super(rewrite(url), protocols)
    }
  }

  const origWindowOpen = window.open.bind(window)
  window.open = (url?: string | URL, target?: string, features?: string) => origWindowOpen(url === undefined ? url : rewrite(url), target, features)

  // 下载链接：模板中的 <a href="/api/..."> 与脚本里创建后直接 click() 的临时链接
  const fixAnchor = (a: HTMLAnchorElement) => {
    const href = a.getAttribute('href')
    if (href && href.startsWith('/') && !href.startsWith('//')) {
      const next = withBase(href)
      if (next !== href) {
        a.setAttribute('href', next)
      }
    }
  }
  document.addEventListener(
    'click',
    (e) => {
      const a = (e.target as Element | null)?.closest?.('a')
      if (a) {
        fixAnchor(a)
      }
    },
    true,
  )
  const origClick = HTMLAnchorElement.prototype.click
  HTMLAnchorElement.prototype.click = function (this: HTMLAnchorElement) {
    fixAnchor(this)
    origClick.call(this)
  }
}

if (basePath) {
  install()
}
