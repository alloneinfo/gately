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
- Passkey (WebAuthn) sign-in: only the public key is stored, user verification (fingerprint, face or PIN) is required, and the signature counter guards against cloned authenticators; requires access through a domain name over HTTPS (or localhost);
- Two-factor authentication (TOTP, verified against the RFC 6238 test vectors);
- Read-only accounts: can view status and configuration, but cannot make changes, download backups, read secrets or open terminals; sessions are invalidated immediately when the password is reset or the account is deleted;
- Login rate limiting;
- Local reverse proxies: when the admin UI sits behind nginx, haproxy or Gately Web Services on the same host, `X-Real-IP` / `X-Forwarded-For` are trusted only when sent from addresses of this host. LAN-only access, login rate limiting and the audit log use them to get the real source. You can turn this off or add extra trusted proxies under "Settings → Access Control";
- Secure entrance path, LAN-only access and admin UI HTTPS;
- fnOS unified gateway: when the fnOS package is accessed through the fnOS unified gateway, fnOS controls access and only fnOS administrators can open it. "Sign in with fnOS" is off by default; once enabled, fnOS administrators who are signed in enter as the Gately administrator (skipping two-step verification), while regular fnOS users still sign in. See [fnOS native app](/en/docs/guide/install#fnos);
- Module switches: modules turned off under "Settings → Modules" start no services and serve no admin API, which shrinks the attack surface;
- Multi-token API: each token has read/write access only to the modules selected at creation, can have an expiry time, and records the source IP of its most recent use. Account, two-factor authentication, token management, and the backup, restore, restart and system settings endpoints accept login sessions only. See [API Tokens & OpenAPI](/en/docs/reference/api);
- Site access control: automatic bans on authentication failures, GeoIP country filtering and path rules. See [Web Services](/en/docs/modules/web);
- Configuration backups support encryption (scrypt + AES-GCM). See [Configuration Backup](/en/docs/modules/backup).

## Audit and traceability

The audit log records sign-ins, configuration changes and sensitive access. Config history keeps the version of each configuration file before it was overwritten and supports one-click rollback. See [Audit Log & Config History](/en/docs/guide/audit). Release artifacts come with cosign signatures, build provenance attestations and an SBOM; see [Verifying Releases](/en/docs/guide/verify) for how to check them.

## Defaults for high-risk features

The local shell terminal is disabled by default. The SSH terminal uses a trust-on-first-use policy to record host key fingerprints.

## Reporting security issues

Please report vulnerabilities privately through the repository's Security page and avoid disclosing details in public issues: <https://github.com/wangwanjie/gately/security>

For more practical recommendations, see the [Hardening Guide](/en/docs/guide/hardening).
