# Security Model & Privacy Promise

Gately runs on the network edge and handles DNS provider keys, certificate private keys and other credentials, so security and transparency are its primary design goals.

## Privacy promise

- **Zero outbound connections by default**: no connections to project servers, no device reporting, no remote configuration push.
- **Visible, editable outbound addresses**: public IP lookup, STUN servers and similar addresses are shown in the UI and can be changed or turned off.
- **Self-managed third-party sign-in**: GitHub, Google, QQ and OIDC sign-in use OAuth apps you register yourself, with no relay in between.
- **Self-managed upgrades**: the program never replaces its own binary in the background; you decide when to upgrade.

## Data protection

- Configuration is plain JSON with file mode 0600, so it can be audited and rolled back.
- Sensitive fields are masked in API output.
- The cloudflared token is passed through an environment variable and does not appear in the process list; frp and Tailscale tokens and keys are stored with 0600 permissions.

## Authentication and access control

- bcrypt password hashing and HMAC-signed session tokens;
- Two-factor authentication (TOTP, verified against the RFC 6238 test vectors);
- Login rate limiting;
- Secure entrance path, LAN-only access and admin UI HTTPS;
- Multi-token API: each token has read/write access only to the modules selected at creation, can have an expiry time, and records the source IP of its most recent use. Account, two-factor authentication, token management, and the backup, restore, restart and system settings endpoints accept login sessions only. See [API Tokens & OpenAPI](/en/docs/reference/api);
- Site access control: automatic bans on authentication failures, GeoIP country filtering and path rules. See [Web Services](/en/docs/modules/web);
- Configuration backups support encryption (scrypt + AES-GCM). See [Configuration Backup](/en/docs/modules/backup).

## Defaults for high-risk features

The local shell terminal is disabled by default. The SSH terminal uses a trust-on-first-use policy to record host key fingerprints.

## Reporting security issues

Please report vulnerabilities privately through the repository's Security page and avoid disclosing details in public issues: <https://github.com/wangwanjie/gately/security>

For more practical recommendations, see the [Hardening Guide](/en/docs/guide/hardening).
