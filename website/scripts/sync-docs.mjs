// 把仓库根目录 docs/ 中的权威文档同步到官网，避免两份内容各自演化。
import { copyFileSync, existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const pairs = [
  ['../docs/MIGRATION.md', 'docs/guide/migration.md'],
  ['../docs/DOCKER_API.md', 'docs/reference/docker-api.md'],
  ['../docs/webterminal-api.md', 'docs/reference/webterminal-api.md'],
  ['../docs/api/API.md', 'docs/reference/api-reference.md'],
  ['../docs/api/openapi.json', 'public/openapi.json']
]

for (const [from, to] of pairs) {
  const src = resolve(root, from)
  if (!existsSync(src)) {
    console.warn(`跳过：${from} 不存在`)
    continue
  }
  copyFileSync(src, resolve(root, to))
  console.log(`已同步 ${from} -> ${to}`)
}
