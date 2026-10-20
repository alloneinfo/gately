#!/usr/bin/env bash
# 构建官网并同步到 hk_main：由 Gately 的静态站点直接读取宿主机 /data/gately-website/site（容器内路径相同），无需重载。
# 用法：cd website && pnpm deploy:site
set -euo pipefail

HOST="${GATELY_DEPLOY_HOST:-root@vanjay.cn}"
PORT="${GATELY_DEPLOY_PORT:-6021}"
DEST="${GATELY_DEPLOY_DEST:-/data/gately-website/site/}"

cd "$(dirname "$0")/.."
pnpm build
# --delete 保证删除的页面同步消失；ACME 验证目录在 /data/gately-website/acme，不受影响
rsync -az --delete -e "ssh -p ${PORT}" .vitepress/dist/ "${HOST}:${DEST}"
echo "已部署到 ${HOST}:${DEST}"
