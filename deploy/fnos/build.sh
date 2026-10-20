#!/bin/bash
# 构建飞牛 fnOS 离线安装包（.fpk）。
#
# 用法：deploy/fnos/build.sh <版本号> <x86|arm> <gately 二进制路径> [输出目录]
#   版本号  形如 0.5.2 或 v0.5.2（自动去掉前缀 v；带后缀的 v0.5.2-rc1 同样可用）
#   二进制  对应架构的 linux 版 gately（x86 用 linux/amd64，arm 用 linux/arm64）
# 依赖官方打包工具 fnpack：https://developer.fnnas.com/docs/cli/fnpack
set -euo pipefail

VERSION="${1:-}"
PLATFORM="${2:-}"
BINARY="${3:-}"
OUT_DIR="${4:-.}"

if [ -z "${VERSION}" ] || [ -z "${PLATFORM}" ] || [ -z "${BINARY}" ]; then
    echo "用法: $0 <版本号> <x86|arm> <gately 二进制路径> [输出目录]" >&2
    exit 2
fi
if [ "${PLATFORM}" != "x86" ] && [ "${PLATFORM}" != "arm" ]; then
    echo "平台只能是 x86 或 arm" >&2
    exit 2
fi
if [ ! -f "${BINARY}" ]; then
    echo "找不到二进制: ${BINARY}" >&2
    exit 1
fi
if ! command -v fnpack >/dev/null 2>&1; then
    echo "未找到 fnpack，请先安装: https://developer.fnnas.com/docs/cli/fnpack" >&2
    exit 1
fi

VERSION="${VERSION#v}"
SRC_DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "${OUT_DIR}"
OUT_DIR="$(cd "${OUT_DIR}" && pwd)"

# 在临时目录组装工程，不污染源码目录
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT
PKG_DIR="${WORK_DIR}/gately"
mkdir -p "${PKG_DIR}"
cp -R "${SRC_DIR}/cmd" "${SRC_DIR}/config" "${SRC_DIR}/wizard" "${SRC_DIR}/app" "${PKG_DIR}/"
cp "${SRC_DIR}/ICON.PNG" "${SRC_DIR}/ICON_256.PNG" "${PKG_DIR}/"
sed -e "s/@VERSION@/${VERSION}/" -e "s/@PLATFORM@/${PLATFORM}/" "${SRC_DIR}/manifest" > "${PKG_DIR}/manifest"
install -m 0755 "${BINARY}" "${PKG_DIR}/app/gately"
chmod +x "${PKG_DIR}"/cmd/*

(cd "${WORK_DIR}" && fnpack build gately)

OUT_FILE="${OUT_DIR}/gately_${VERSION}_fnos_${PLATFORM}.fpk"
mv "${WORK_DIR}/gately.fpk" "${OUT_FILE}"
echo "已生成 ${OUT_FILE}"
