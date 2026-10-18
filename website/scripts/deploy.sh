#!/usr/bin/env bash
# 构建官网并同步到 hk_main（nginx 以只读方式将宿主机 /data/gately-website 挂载为容器内 /data/gately，无需重载）。
# 用法：cd website && pnpm deploy:site
set -euo pipefail

HOST="${GATELY_DEPLOY_HOST:-root@home.vanjay.cn}"
PORT="${GATELY_DEPLOY_PORT:-6021}"
DEST="${GATELY_DEPLOY_DEST:-/data/gately-website/site/}"

cd "$(dirname "$0")/.."
pnpm build
# --delete 保证删除的页面同步消失；ACME 验证目录在 /data/gately-website/acme，不受影响
rsync -az --delete -e "ssh -p ${PORT}" .vitepress/dist/ "${HOST}:${DEST}"
echo "已部署到 ${HOST}:${DEST}"
