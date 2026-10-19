// 把仓库根目录 docs/ 中的权威文档同步到官网，避免两份内容各自演化。
import { copyFileSync, existsSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const pairs = [
  ['../docs/MIGRATION.md', 'docs/guide/migration.md'],
  ['../docs/DOCKER_API.md', 'docs/reference/docker-api.md'],
  ['../docs/webterminal-api.md', 'docs/reference/webterminal-api.md'],
  ['../docs/DIAGNOSTICS.md', 'docs/reference/diagnostics.md'],
  ['../docs/MONITORING.md', 'docs/reference/monitoring.md'],
  ['../docs/VERIFY.md', 'docs/guide/verify.md'],
  ['../docs/api/API.md', 'docs/reference/api-reference.md'],
  ['../docs/api/openapi.json', 'public/openapi.json'],
  // 英文版（仓库 docs/en/ 为权威来源），其余语言的参考类页面直接链接到英文版
  ['../docs/en/MIGRATION.md', 'en/docs/guide/migration.md'],
  ['../docs/en/DOCKER_API.md', 'en/docs/reference/docker-api.md'],
  ['../docs/en/webterminal-api.md', 'en/docs/reference/webterminal-api.md'],
  ['../docs/en/DIAGNOSTICS.md', 'en/docs/reference/diagnostics.md'],
  ['../docs/en/MONITORING.md', 'en/docs/reference/monitoring.md'],
  ['../docs/en/VERIFY.md', 'en/docs/guide/verify.md']
]
// 接口清单由后端按各语言生成（make api-docs），直接同步到对应语言目录
for (const lang of ['en', 'zh-TW', 'ja', 'ko', 'es', 'fr', 'de', 'ru', 'pt-BR']) {
  pairs.push([`../docs/api/${lang}/API.md`, `${lang}/docs/reference/api-reference.md`])
}

for (const [from, to] of pairs) {
  const src = resolve(root, from)
  if (!existsSync(src)) {
    console.warn(`跳过：${from} 不存在`)
    continue
  }
  mkdirSync(dirname(resolve(root, to)), { recursive: true })
  copyFileSync(src, resolve(root, to))
  console.log(`已同步 ${from} -> ${to}`)
}
