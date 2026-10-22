---
title: Changelog
description: Release announcements and notes for each Gately version.
---

# Changelog

Announcements for major releases are listed below. See [GitHub Releases](https://github.com/wangwanjie/gately/releases) for the full changes and downloads of every version.

## [v0.5.6.2](/en/news/v0.5.6.2)

October 23, 2026 · Redesigned Web service login and error pages with two-step verification in a popup. Fixes the Docker module failing to connect on Docker 29, the overview flashing before the login page, and Cloudflare credentials and renamed DDNS records imported from Lucky not syncing.

## [v0.5.6.1](/en/news/v0.5.6.1)

October 22, 2026 · Fixes firewall rules written by the Docker image crashing an older nft on the host.

## [v0.5.6](/en/news/v0.5.6)

October 22, 2026 · Adds intrusion prevention: fail2ban-style log jails, trap ports and port-scan detection, with bans synced to nftables.

## [v0.5.5.2](/en/news/v0.5.5.2)

October 22, 2026 · Fixes sensitive files in the config directory being readable by other local users on fnOS and similar filesystems.

## [v0.5.5.1](/en/news/v0.5.5.1)

October 21, 2026 · Runtime logs are written to disk and survive restarts and upgrades.

## [v0.5.5](/en/news/v0.5.5)

October 21, 2026 · Import nginx configs, auto deploy certificates to other hosts and containers, large uploads and WebDAV behind the WAF, subscribed IP lists, and new self-check items for certificate coverage and frp conflicts.

## [v0.5.4](/en/news/v0.5.4)

October 21, 2026 · Online upgrade for Linux binary installs, so PVE LXC containers no longer need to be recreated; new OpenWrt packages (.ipk / .apk).

## [v0.5.3](/en/news/v0.5.3)

October 21, 2026 · Native fnOS app that plugs into the fnOS unified gateway and signs fnOS administrators in with their fnOS session; feature modules can be turned on or off.

## [v0.5.2](/en/news/v0.5.2)

October 20, 2026 · SNI routing lets several services share port 443, Web Services can replace nginx, shared users add TOTP and single sign-on, and a performance comparison with haproxy + nginx.

## [v0.5.0](/en/news/v0.5.0)

October 19, 2026 · Audit log, config history and rollback, Passkey, read-only accounts, ACME HTTP-01, HTTP/3, PROXY protocol, global Prometheus metrics, network self-check, and signed releases.
