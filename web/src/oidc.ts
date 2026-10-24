// OIDC 身份提供方的编辑模型。接口返回的客户端密钥为脱敏占位符：
// 编辑时密钥框显示为空（hasSecret 标记已设置），保存时由 oidcToAPI 还原占位符表示不修改
import { SECRET_MASK } from './sharedUsers'

export interface OIDCClient {
  id: string
  name: string
  secret: string
  public: boolean
  redirectUris: string[]
  allow: string[]
  remark: string
  hasSecret?: boolean
}

export interface OIDCSettings {
  enable: boolean
  issuer: string
  clients: OIDCClient[]
}

// OIDC 协议路径，与后端 oidc.go 保持一致
export const OIDC_DISCOVERY = '/.well-known/openid-configuration'
export const OIDC_ENDPOINTS = {
  authorize: '/__gately_oidc/authorize',
  token: '/__gately_oidc/token',
  userinfo: '/__gately_oidc/userinfo',
  jwks: '/__gately_oidc/jwks',
  logout: '/__gately_oidc/logout',
}

// oidcFromAPI 把接口数据转换为编辑用的格式；老版本没有该字段时返回未启用的空配置
export function oidcFromAPI(s: Partial<OIDCSettings> | null | undefined): OIDCSettings {
  return {
    enable: s?.enable ?? false,
    issuer: s?.issuer ?? '',
    clients: (s?.clients ?? []).map((c) => ({
      ...c,
      redirectUris: c.redirectUris ?? [],
      allow: c.allow ?? [],
      hasSecret: c.secret === SECRET_MASK,
      secret: c.secret === SECRET_MASK ? '' : c.secret,
    })),
  }
}

// oidcToAPI 去掉编辑用字段；已设置密钥且未输入新密钥时提交占位符
export function oidcToAPI(s: OIDCSettings): OIDCSettings {
  return {
    ...s,
    issuer: s.issuer.trim(),
    clients: s.clients.map(({ hasSecret, ...c }) => ({ ...c, secret: c.secret === '' && hasSecret && !c.public ? SECRET_MASK : c.secret })),
  }
}

// randomSecret 生成 32 字节随机密钥（base64url，43 个字符）
export function randomSecret(): string {
  const b = new Uint8Array(32)
  crypto.getRandomValues(b)
  return btoa(String.fromCharCode(...b))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

// newClient 返回新客户端的默认值，ID 取一个未被使用的 app、app-2……
export function newClient(used: string[]): OIDCClient {
  let id = 'app'
  for (let i = 2; used.includes(id); i++) {
    id = `app-${i}`
  }
  return { id, name: '', secret: randomSecret(), public: false, redirectUris: [], allow: [], remark: '' }
}
