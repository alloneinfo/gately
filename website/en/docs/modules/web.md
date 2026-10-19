# Web Services

Serve reverse proxy, redirects, static files and fixed text by domain. This is Gately's unified web entry point. Menu location: sidebar "Web → Web Services".

## Structure: listeners, sites and the default site

The configuration has three layers:

1. Listener: one listening port. Fields include network (`tcp` dual-stack, `tcp4`, `tcp6`), listen address (empty for all addresses), port, HTTPS switch, minimum TLS version (1.0 / 1.1 / 1.2 / 1.3), and whether to accept the PROXY protocol.
2. Site: a site under a listener, matched by domain. Domains can be an exact domain, `*.example.com` (matches subdomains at any depth) or `*` (matches everything). Matching priority is exact domain, wildcard, then `*`.
3. Default site: used when no site matches. When it is not enabled, a 404 is returned.

## Site types

| Type | Description |
| --- | --- |
| Reverse proxy | Forwards to one or more backends. Backend addresses use the form `http(s)://host:port`, and WebSocket is supported |
| Redirect | Redirects to a full URL with status code 301 / 302 / 307 / 308 (default 302), optionally keeping the original path and query string |
| Static files | The directory must be an absolute path. With "Directory listing" off, a directory request returns a response only when `index.html` exists |
| Fixed text | Custom content, `Content-Type` (default `text/plain; charset=utf-8`) and status code (100-599, default 200) |

Other options for reverse proxy sites:

- Use backend Host: the Host header is changed to the backend address; when off, the Host from the client request is kept;
- Skip backend certificate verification, disable Keep-Alive;
- Send the PROXY protocol to the backend (v2 header, passes the real client IP, requires backend support);
- Add `X-Forwarded-For` / `X-Forwarded-Proto` / `X-Forwarded-Host`.

Sites of every type can set an access log, cross-origin (CORS), custom request and response headers (one `Key: Value` per line; an empty Value removes that header), and can reference a [WAF Firewall](/en/docs/modules/waf) instance (if the instance does not exist or is disabled, requests pass through).

## Transport and protocols

- HTTPS: uses a certificate from the [SSL Certificates](/en/docs/modules/ssl) module that covers the domain;
- HTTP/3 (QUIC): turn it on for a listener with HTTPS enabled. HTTP/3 is then also served over UDP on the same port number and advertised to browsers with the `Alt-Svc` response header; allow that UDP port in your firewall and port mapping;
- WebSocket: supported automatically by reverse proxy;
- PROXY protocol: a listener can accept PROXY headers sent by upstream components (such as haproxy and frp). It trusts only the IPs or subnets listed in "Global settings → Trusted PROXY sources"; leaving it empty trusts no source. Reverse proxy can send PROXY headers to the backend.

## Authentication

Configure it on the "Security" page of the site editor:

- Web authentication: the browser shows a login page;
- Basic authentication: HTTP Basic;
- Users: one `username:password` per line, and the password can be a bcrypt hash. Basic authentication needs at least one user when enabled; web authentication needs at least one user or third-party login identity when enabled;
- Third-party login: GitHub, Google, QQ and OIDC are supported, using your own OAuth apps. Allowed identities go one per line in the form `provider:account`, for example `github:octocat` and `google:me@gmail.com`. A `#` at the end of a line adds a comment, and `github:*` means any account of that provider.

Third-party login providers are configured in "Global settings": enter your own Client ID / Secret and enable the provider, and also set a callback address (it must be served by one of Gately's listeners, for example `https://auth.example.com`). The callback URL to register with GitHub / Google / QQ is:

```text
https://auth.example.com/__gately_auth/oauth/callback
```

Third-party login for the management UI itself is configured separately under "Settings → Account & Security".

## Access control

### IP allowlist and blocklist

"IP access control" on the site's "Security" page can be off, allowlist or blocklist. The list takes one IP or subnet per line (such as `192.168.1.0/24`).

### Path rules

In "Path rules" on the "Security" page, override the site's authentication and IP settings by path prefix:

| Field | Values |
| --- | --- |
| `prefix` | A path prefix starting with `/`, such as `/admin` |
| `auth` | Inherit from site (empty), `off` (no authentication), `on` (authentication required) |
| `ipMode` / `ipList` | Inherit from site (empty), `whitelist` or `blacklist`, with a list of IPs or subnets |

Matching rules:

- The longest prefix wins;
- Matching follows path segment boundaries: `/admin` matches `/admin` and `/admin/x`, but not `/administrator`;
- The request path is normalized first (`.`, `..` and duplicate slashes are removed) and then matched, so `/public/../admin` is treated as `/admin`;
- Only one rule can be configured for the same prefix.

The "Authentication scope" option (`authOptional`): when ticked, authentication applies only to paths marked "Required" in the rules, and all other paths are public. This option needs Basic or web authentication to be enabled first.

### Temporary share links

After Basic or web authentication is enabled, you can create share links for a site so that visitors can access it without an account:

| Field | Description |
| --- | --- |
| Name | A label, such as "For Alice" |
| Expiry | Leave empty for no expiry |
| Uses `maxUses` | Maximum number of times the link can be redeemed; 0 means unlimited. The count resets after a restart |
| Path limit `pathPrefix` | When not empty, the link can only access paths under this prefix; it must start with `/` |

A visitor opens `https://domain/any-path?_gs=token` to redeem a session. After a successful redemption, the server sets a login-free cookie and redirects to the address without the `_gs` parameter. The session lasts at most 24 hours and never beyond the link's expiry. An invalid token is handled as a normal visit and does not reveal the reason for the failure. Sessions already issued become invalid immediately after the link is deleted. The token is generated after saving (at least 16 characters), and the full link can be copied in the UI.

### Authentication failure rate limiting and automatic ban

- After 10 failed authentications from the same source IP within 15 minutes, further attempts are rejected for a while;
- Under "Global settings → Auto ban → Ban on authentication failure", enter a number of minutes (0 to 43200); the source IP is banned when it reaches 10 failures. 0 means no ban, rate limiting only;
- An `AuthBanned` event is published when an automatic ban triggers, and it can be pushed through the [Notification Center](/en/docs/modules/notify);
- IPv6 addresses are counted and banned per /64 subnet.

### Ban list

The ban list in "Global settings" shows the currently banned IPs, the reason (repeated authentication failures or manual ban) and the expiry time, and each entry can be lifted with "Unban". You can also ban manually: enter an IP and a number of hours (1 to 720, which is at most 30 days); the API defaults to 24 hours.

### GeoIP country filter

A site's "Country filter" has three modes: off, allow only, and deny. Enter two-letter ISO country codes separated by commas, such as `CN, HK, US`. At least one country is required when enabled.

- Filtering depends on a GeoIP database, which you must first import under "Global settings → GeoIP database": upload a `.mmdb` file, or enter a download URL. MaxMind GeoLite2, DB-IP and other country or city databases are supported; the upload limit is 129 MB;
- When no database is loaded, filtering has no effect (all visitors pass) and the UI shows a notice;
- Private network addresses are always allowed;
- In "Deny" mode, addresses whose country cannot be found in the database are allowed.

## Reverse proxy enhancements

These options are on the "Traffic" page of the site editor.

### Backends, weight and backup

The backend list takes one address per line, optionally followed by parameters:

```text
http://10.0.0.2:8080 weight=3
http://10.0.0.3:8080 backup
```

- `weight=N`: weight, an integer from 1 to 100, default 1;
- `backup`: a standby backend that is used only when all non-backup backends are unavailable. When every backend is unavailable, all non-backup backends are tried again.

### Load balancing policy

| UI option | `lbPolicy` | Description |
| --- | --- | --- |
| Round robin (weighted) | Empty | Default |
| Random (weighted) | `random` | |
| IP hash (sticky sessions) | `ip_hash` | The same client IP always goes to the same backend |
| Least connections | `least_conn` | Picks the smallest value of connections / weight |

### Health checks

The active check switch is off by default. Fields and defaults once enabled:

| Field | Default | Range |
| --- | --- | --- |
| Mode `mode` | `http` | `http` (requests the check path; a status code below 400 counts as healthy) or `tcp` (only tests the connection) |
| Path `path` | `/` | Starts with `/`, http mode only |
| Interval `interval` | 10 seconds | 1 to 3600 |
| Timeout `timeout` | 3 seconds | 1 to 60 |
| Consecutive failures `fails` | 3 failures mark it unavailable | 1 to 20 |
| Consecutive successes `passes` | 2 successes recover it | 1 to 20 |

When sending the PROXY protocol to the backend is enabled, HTTP checks switch to TCP automatically.

Passive checks are always in effect: among multiple backends, a backend with 3 consecutive connection failures is paused for 30 seconds before being probed again (a single backend with no active check is never removed). Once active checks are enabled, recovery is decided entirely by the check results. When a backend's state changes, upstream-down and upstream-recovered events are published, and they can be pushed through the [Notification Center](/en/docs/modules/notify).

### Compression

Compression can be enabled for reverse proxy, static file and fixed text sites (not for redirect sites). Based on `Accept-Encoding`, `br` is preferred, then `gzip`.

- Only text-like content is compressed: `text/*`, JSON, JavaScript, XML, XHTML, RSS / Atom, WebAssembly and SVG; `text/event-stream` is excluded;
- Responses whose declared `Content-Length` is below 256 bytes are left as they are;
- Responses that already have a `Content-Encoding` are left as they are; WebSocket upgrade requests pass straight through;
- After compression, strong ETags are converted to weak ETags.

### Per-IP rate limiting

Token-bucket rate limiting per client IP. Fields:

| Field | Description |
| --- | --- |
| `rate` | Requests allowed per second |
| `burst` | Burst capacity. When `rate` is set and `burst` is 0, it defaults to `rate` rounded down plus 1 |
| `maxConns` | Maximum concurrent requests per IP; 0 means unlimited |

When rate limiting is enabled, at least one of `rate` and `maxConns` must be set.

### In-memory response cache

Applies to reverse proxy sites only. The cache lives in memory and uses LRU eviction.

| Field | Default | Range |
| --- | --- | --- |
| `ttl` default duration | 0 | 0 to 30 days (in seconds). When the upstream declares no cache directive, 200 responses are cached for this long; 0 means only responses the upstream explicitly allows are cached |
| `maxMb` cache capacity | 64 | 1 to 4096 MB |

Caching rules:

- Only GET requests are cached, and the request must not contain `Authorization`, `Cookie` or `Range` headers, nor `Cache-Control: no-cache`;
- Only responses with status 200 are cached; a response is skipped if it contains `Set-Cookie`, `Vary: *`, or a `Cache-Control` of `no-store` / `private` / `no-cache`;
- `s-maxage` is honored first, then `max-age`;
- A single entry is limited to 4 MB and also to at most a quarter of the total capacity;
- A hit returns the `X-Cache: HIT` and `Age` headers, and a miss returns `X-Cache: MISS`;
- The cache key includes Host, the full request URI and `Accept-Encoding`.

### Path rewrite and redirect

Rules are matched in order, and matching stops at the first hit.

| Field | Description |
| --- | --- |
| `pattern` | A regular expression matched against the request path, such as `^/old/(.*)$` |
| `to` | The replacement result; use `$1` to reference groups |
| `redirect` | Off for an internal rewrite (the rewritten path is sent to the backend, and the target must start with `/`); on for a redirect |
| `code` | Redirect status code 301 / 302 / 307 / 308, default 302 |

Redirects keep the original query parameters.

## Access statistics and monitoring

### Site statistics

For a running listener, click "Stats" on a site row to open the statistics window, where you can choose the last 1 hour, 6 hours or 24 hours:

- Minute-level access statistics: one point per minute, kept for the last 24 hours, including a request count curve and 4xx / 5xx error curves;
- Status code distribution (2xx / 3xx / 4xx / 5xx) and cumulative response traffic;
- Top 10 source IPs and top 10 countries (country statistics need an imported GeoIP database);
- Reverse proxy sites show the health status, weight, backup flag and current connection count of each backend.

Statistics are kept in memory and start counting again after a restart or after the listener is modified. The rankings track at most 5000 distinct sources.

### Prometheus metrics

The global metrics endpoint `GET /api/metrics` aggregates metrics from all modules; see [Monitoring](/en/docs/reference/monitoring). The module endpoint below covers Web Services metrics only.

The metrics endpoint is `GET /api/webservice/metrics` in Prometheus text format, and it requires login credentials or an [API token](/en/docs/reference/api-reference).

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/webservice/metrics
```

| Metric | Type | Description |
| --- | --- | --- |
| `gately_site_requests_total` | counter | Cumulative requests of a site |
| `gately_site_errors_total` | counter | Backend errors of a site |
| `gately_site_active_requests` | gauge | Requests in progress |
| `gately_site_response_bytes_total` | counter | Cumulative response bytes |
| `gately_site_responses_total` | counter | Responses counted by status class (label `class`: `2xx` to `5xx`) |
| `gately_upstream_up` | gauge | Whether a backend is healthy: 1 healthy, 0 unavailable (with label `upstream`) |

All metrics carry the labels `listener` (listener name#listener key) and `site` (site name, or the first domain when unnamed).

## Deployment tip

Web Services need to listen on host ports such as 80 / 443. For Docker deployments, `--network host` is recommended.
