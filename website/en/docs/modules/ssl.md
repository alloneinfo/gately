# SSL Certificates

Issue and renew certificates automatically for use by [Web Services](/en/docs/modules/web), or export them to other software.

## Capabilities

- **ACME DNS-01** automatic issuance and renewal, reusing the DNS providers from [Dynamic DNS](/en/docs/modules/ddns), with no need to open ports 80 / 443;
- **ACME HTTP-01**: needs no DNS provider, but port 80 of each domain must be reachable from the internet; wildcards are not supported;
- Wildcard certificates and EAB (for example ZeroSSL);
- Upload an existing certificate, or read it from files;
- Map to a directory: after a certificate changes, the files are written to the specified directory and a script runs, so other services can reload.

## HTTP-01 validation

Choose HTTP-01 as the certificate's "Method". During issuance, the CA requests `http://<domain>/.well-known/acme-challenge/<token>`:

- If [Web Services](/en/docs/modules/web) already listens on port 80, it answers directly without affecting existing sites;
- If port 80 is free, Gately listens on it temporarily during issuance and closes it right after;
- If your router forwards public port 80 to another local port, enter that port in "Temporary port".

Many residential ISPs (for example in mainland China) block inbound port 80; use DNS-01 in that case.

## Certificate mapping

Once a certificate is enabled, Gately writes `.crt`, `.key`, `.pem` and `_issuerCertificate.crt` into the mapping directory and runs the change script.

## Security note

Private keys are stored in `ssl.json` in the configuration directory. Keep that file safe, and make sure the repository is private if you put it under version control.

## Migrating from Lucky

Certificates that are already issued are imported as they are and are not requested again before they expire. ZeroSSL registers a new ACME account using the EAB credentials from Lucky. See the [Migration Guide](/en/docs/guide/migration) for details.
