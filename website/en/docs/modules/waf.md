# WAF Firewall

A web application firewall based on [Coraza](https://coraza.io/) and OWASP CRS v4, which can be attached to sites in [Web Services](/en/docs/modules/web).

## Capabilities

- Built-in OWASP CRS v4 rule set, with rule files that can be enabled individually;
- Custom SecLang rules;
- Blocking mode and detection-only mode;
- Adjustable request body inspection limit: over the limit, reject the request or inspect only the first part; request body inspection can be skipped by path;
- WebDAV compatibility mode: allows the methods, headers and any upload content type used by cloud drives and WebDAV;
- Multiple instances, referenced by sites as needed.

## Recommendations

1. For a new site, start with "Detect only" and watch the logs for a while to confirm there are no false positives;
2. Adjust rule files or add custom exceptions for any false positives;
3. Then switch to "Block".

## Sites that take large uploads

By default the WAF inspects request bodies of up to 12.5 MB and rejects larger requests (413), so cloud drives, WebDAV and file managers cannot upload big files. Adjust this in the instance's "Request body inspection":

- **Inspection limit**: 0 means the default 12.5 MB, up to 1024 MB;
- **Over the limit**: "Reject the request" keeps the previous behavior; "Inspect the first part only" inspects only the part within the limit and passes the rest to the backend unchanged, which suits sites that take big uploads;
- **WebDAV and upload compatible**: by default OWASP CRS only allows GET, HEAD, POST and OPTIONS, and blocks the `Lock-Token`, `If` and `Content-Range` headers and content types outside its allow list, so WebDAV clients cannot list folders, upload or lock files. When enabled, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK and similar methods and these headers are allowed, and upload content types are no longer restricted; request bodies of PUT and `application/octet-stream` are treated as file content and no longer inspected as form parameters. Checks for injection, path traversal and other attacks still apply;
- **Paths without body inspection**: one path prefix per line (such as `/api/upload` or `/remote.php/dav/`). Matching requests skip body inspection; headers, URL and query parameters are still inspected.

We recommend a separate instance for cloud drive and WebDAV sites: turn on "WebDAV and upload compatible" and choose "Inspect the first part only", add the upload endpoints to the skipped paths if needed, and let other sites keep the defaults.

curl, rclone and similar clients send an `Expect: 100-continue` header when uploading larger content, which CRS 4 blocks by default. All Gately instances allow this header (Gately handles 100-continue itself), so no extra setting is needed.

## Testing

The WAF is tested with real attack payloads, covering blocking of path traversal and similar attacks.
