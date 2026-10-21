# WAF Firewall

A web application firewall based on [Coraza](https://coraza.io/) and OWASP CRS v4, which can be attached to sites in [Web Services](/en/docs/modules/web).

## Capabilities

- Built-in OWASP CRS v4 rule set, with rule files that can be enabled individually;
- Custom SecLang rules;
- Blocking mode and detection-only mode;
- Multiple instances, referenced by sites as needed.

## Recommendations

1. For a new site, start with "Detect only" and watch the logs for a while to confirm there are no false positives;
2. Adjust rule files or add custom exceptions for any false positives;
3. Then switch to "Block".

## Testing

The WAF is tested with real attack payloads, covering blocking of path traversal and similar attacks.
