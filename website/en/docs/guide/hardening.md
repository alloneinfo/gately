# Hardening Guide

The admin UI controls DNS, certificates and ports. Work through the following checklist.

## Admin UI

- [ ] Change the administrator password after the first sign-in
- [ ] Enable two-factor authentication (TOTP)
- [ ] Set a secure entrance path so the admin UI is reachable only through a custom path
- [ ] Enable "LAN-only access", and use a VPN (such as [Tailscale](/en/docs/modules/tunnels)) when you need remote administration
- [ ] Enable admin UI HTTPS when accessing it over untrusted networks
- [ ] Give automation scripts and the [CLI / MCP](/en/docs/guide/cli-mcp) least-privilege [API tokens](/en/docs/reference/api) with an expiry time, and avoid sharing the administrator password
- [ ] Set an encryption passphrase for [configuration backups](/en/docs/modules/backup) and enable scheduled backups
- [ ] Configure push notifications in the [Notification Center](/en/docs/modules/notify) for events such as sign-ins, certificates and backup failures

## Public-facing services

- When exposing LAN services through [Web Services](/en/docs/modules/web), enable HTTPS and combine it with basic auth, web auth or third-party sign-in;
- Enable the [WAF](/en/docs/modules/waf) on public sites: start in "detect only" mode to watch for false positives, then switch to "block";
- Restrict sources with IP allow/block lists, GeoIP country filtering and automatic bans on authentication failures.

## Docker

Mounting `docker.sock` is equivalent to granting root-level access to the host. Remove the mount when you do not need [Docker Management](/en/docs/modules/docker).

## Web terminal

The local shell is disabled by default. Enable it only when truly needed, and make sure two-factor authentication is on for the admin UI.

## Configuration backup

Keep the config directory in a private git repository or back it up regularly. `ssl.json` contains certificate private keys, so store it carefully.
