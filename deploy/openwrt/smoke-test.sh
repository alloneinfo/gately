#!/bin/sh
# OpenWrt 安装包冒烟测试：打出 amd64 的 .ipk / .apk，分别在 OpenWrt 24.10（opkg）与 25.12（apk）的官方 rootfs 容器中
# 安装、确认服务启动并登记到包管理器，再卸载确认服务停止。需要 Docker，CI 中运行。
#
# 用法：deploy/openwrt/smoke-test.sh <linux/amd64 的 gately 二进制>
set -eu

# 打包镜像与 release.yml 保持一致；可用环境变量替换为本地已有的镜像
ALPINE=${ALPINE_IMAGE:-alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6}
OPKG_IMAGE=${OPKG_IMAGE:-openwrt/rootfs:x86_64-24.10.4}
APK_IMAGE=${APK_IMAGE:-openwrt/rootfs:x86_64-25.12.0}

if [ $# -lt 1 ] || [ ! -f "$1" ]; then
	echo "用法: $0 <linux/amd64 的 gately 二进制>" >&2
	exit 1
fi
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
WORK=$(mktemp -d)
NAME=gately-openwrt-smoke-$$
trap 'docker rm -f "$NAME" >/dev/null 2>&1 || true; docker run --rm -v "$WORK:/work" "$ALPINE" rm -rf /work/out /work/gately >/dev/null 2>&1 || true; rm -rf "$WORK"' EXIT
cp "$1" "$WORK/gately"

docker run --rm -v "$REPO:/w:ro" -v "$WORK:/work" -w /w "$ALPINE" \
	deploy/openwrt/build.sh v0.0.0-rc1 amd64 /work/gately /work/out

# 在容器中以 procd 为 init 启动 OpenWrt，执行安装、检查与卸载。
# sh -e 不会因 ! 取反的命令失败而退出，卸载后的检查用 if 显式判断
check() {
	image=$1
	script=$2
	echo "==== $image"
	docker run -d --name "$NAME" -v "$WORK/out:/pkgs:ro" "$image" /sbin/init >/dev/null
	sleep 5
	docker exec "$NAME" sh -ec "$script"
	docker rm -f "$NAME" >/dev/null
}

check "$OPKG_IMAGE" '
opkg install /pkgs/*.ipk
sleep 3
pidof gately
/etc/init.d/gately enabled
opkg list-installed | grep "^gately - 0.0.0~rc1-r1"
opkg remove gately
sleep 1
if pidof gately || [ -e /etc/init.d/gately ] || [ -n "$(ls /etc/rc.d | grep gately)" ]; then
	echo "卸载后仍有残留" >&2
	exit 1
fi
'

check "$APK_IMAGE" '
apk add --allow-untrusted /pkgs/*.apk
sleep 3
pidof gately
/etc/init.d/gately enabled
grep -qx "P:gately" /lib/apk/db/installed
apk del gately
sleep 1
if pidof gately || [ -e /etc/init.d/gately ] || [ -n "$(ls /etc/rc.d | grep gately)" ]; then
	echo "卸载后仍有残留" >&2
	exit 1
fi
'

echo "OpenWrt 安装包冒烟测试通过"
