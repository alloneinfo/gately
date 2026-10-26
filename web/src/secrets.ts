// 已保存密钥在接口中显示为脱敏占位符，查看明文需调用各模块的 .../secrets 接口（只读账号无权调用）。
import { api } from './api'

// SECRET_MASK 与后端 crud.SecretMask 一致；原样提交时后端保留原值
export const SECRET_MASK = '********'

// fetchSecret 从 url 取回密钥明文表并返回 field 对应的值
export async function fetchSecret(url: string, field: string): Promise<string | undefined> {
  const m = await api.get<Record<string, string>>(url)
  return m?.[field]
}

// secretOf 返回 fetchSecret 的延迟调用，便于在模板中写 :reveal="secretOf(url, 'token')"；
// key 为空（尚未保存的条目）时返回 undefined，此时输入框只切换显示
export function secretOf(url: string | undefined | false, field: string): (() => Promise<string | undefined>) | undefined {
  if (!url) {
    return undefined
  }
  return () => fetchSecret(url, field)
}
