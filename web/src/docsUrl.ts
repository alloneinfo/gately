// 官网文档地址：简体中文位于根路径，其他语言位于 /<语言>/ 下（与 website/ 目录结构一致）
export function docsUrl(locale: string, path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  return locale === 'zh-CN' ? `https://gately.cn${p}` : `https://gately.cn/${locale}${p}`
}
