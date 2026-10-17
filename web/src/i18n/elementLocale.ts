// Element Plus 组件库自带文案（分页、日期选择器等）与界面语言对应。
import type { Language } from 'element-plus/es/locale'

const loaders: Record<string, () => Promise<{ default: Language }>> = {
  en: () => import('element-plus/es/locale/lang/en'),
  'zh-CN': () => import('element-plus/es/locale/lang/zh-cn'),
  'zh-TW': () => import('element-plus/es/locale/lang/zh-tw'),
  ja: () => import('element-plus/es/locale/lang/ja'),
  ko: () => import('element-plus/es/locale/lang/ko'),
  es: () => import('element-plus/es/locale/lang/es'),
  fr: () => import('element-plus/es/locale/lang/fr'),
  de: () => import('element-plus/es/locale/lang/de'),
  ru: () => import('element-plus/es/locale/lang/ru'),
  'pt-BR': () => import('element-plus/es/locale/lang/pt-br'),
}

export async function loadElementLocale(code: string): Promise<Language> {
  return (await (loaders[code] ?? loaders.en)()).default
}
