// 校验界面文案：源码引用的键是否存在、各语言键集合与占位符是否与 zh-CN 一致。
// 用法：node scripts/check-i18n.mjs [语言代码 ...]（缺省仅校验 zh-CN 与 en 的引用）
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { baseCompile as compile } from '@intlify/message-compiler'

const root = resolve(import.meta.dirname, '..')
const localesDir = join(root, 'src/locales')
const srcDir = join(root, 'src')

function flatten(obj, prefix = '', out = {}) {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object') {
      flatten(v, key, out)
    } else {
      out[key] = String(v)
    }
  }
  return out
}

function loadLocale(code) {
  const dir = join(localesDir, code)
  const out = {}
  try {
    for (const f of readdirSync(dir).filter((x) => x.endsWith('.json'))) {
      const ns = f.replace(/\.json$/, '')
      let data
      try {
        data = JSON.parse(readFileSync(join(dir, f), 'utf8'))
      } catch (e) {
        console.error(`JSON 解析失败 ${code}/${f}: ${e.message}`)
        process.exitCode = 1
        continue
      }
      flatten(data, ns, out)
    }
  } catch {
    // 语言目录不存在
  }
  return out
}

function walk(dir, files = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (name === 'locales') {
      continue
    }
    if (statSync(p).isDirectory()) {
      walk(p, files)
    } else if (/\.(vue|ts)$/.test(name)) {
      files.push(p)
    }
  }
  return files
}

const placeholders = (s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',')
const zh = loadLocale('zh-CN')
const en = loadLocale('en')
let errors = 0

// 1. 源码里的字面量键必须存在于 zh-CN（兼容 t('a.b')、$t('a.b')、keypath="a.b"、te('a.b')）
const used = new Set()
const re = /(?:\bt|\$t|\bte|\btm|\brt)\(\s*(['"`])([\w.-]+)\1|keypath=(['"])([\w.-]+)\3/g
for (const file of walk(srcDir)) {
  const text = readFileSync(file, 'utf8')
  for (const m of text.matchAll(re)) {
    const key = m[2] ?? m[4]
    used.add(key)
    if (!(key in zh)) {
      console.error(`缺少 zh-CN 键: ${key}  (${file.replace(root + '/', '')})`)
      errors++
    }
  }
}

// 2. 各语言与 zh-CN 键集合、占位符对齐（en 必检，其余按参数）
const targets = process.argv.slice(2)
for (const code of ['en', ...targets.filter((c) => c !== 'en' && c !== 'zh-CN')]) {
  const loc = code === 'en' ? en : loadLocale(code)
  for (const [k, v] of Object.entries(zh)) {
    if (!(k in loc)) {
      console.error(`${code} 缺少键: ${k}`)
      errors++
    } else if (placeholders(v) !== placeholders(loc[k])) {
      console.error(`${code} 占位符不一致: ${k}  zh=${placeholders(v)} ${code}=${placeholders(loc[k])}`)
      errors++
    }
  }
  for (const k of Object.keys(loc)) {
    if (!(k in zh)) {
      console.error(`${code} 多余键: ${k}`)
      errors++
    }
  }
}

// 3. 消息语法必须能被 vue-i18n 编译（未转义的 { } @ | 等会在运行时报错）
for (const code of ['zh-CN', 'en', ...targets.filter((c) => c !== 'en' && c !== 'zh-CN')]) {
  const loc = code === 'zh-CN' ? zh : code === 'en' ? en : loadLocale(code)
  for (const [k, v] of Object.entries(loc)) {
    let err = null
    try {
      compile(v, { onError: (e) => { err = e } })
    } catch (e) {
      err = e
    }
    if (err) {
      console.error(`${code} 消息语法错误: ${k}  ${String(err.message ?? err).slice(0, 80)} :: ${v.slice(0, 60)}`)
      errors++
    }
  }
}

// 4. 未被字面量引用的键仅提示（动态拼接的键无法静态识别）
const unused = Object.keys(zh).filter((k) => !used.has(k))
console.log(`zh-CN 共 ${Object.keys(zh).length} 个键，字面量引用 ${used.size} 个，未识别引用 ${unused.length} 个（可能由动态键使用）`)
if (errors) {
  console.error(`共 ${errors} 个问题`)
  process.exitCode = 1
} else {
  console.log('i18n 校验通过')
}
