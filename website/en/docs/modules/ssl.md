# SSL Certificates

Issue and renew certificates automatically for use by [Web Services](/en/docs/modules/web), or export them to other software.

## Capabilities

- **ACME DNS-01** automatic issuance and renewal, reusing the DNS providers from [Dynamic DNS](/en/docs/modules/ddns), with no need to open ports 80 / 443;
- Wildcard certificates and EAB (for example ZeroSSL);
- Upload an existing certificate, or read it from files;
- Map to a directory: after a certificate changes, the files are written to the specified directory and a script runs, so other services can reload.

## Certificate mapping

Once a certificate is enabled, Gately writes `.crt`, `.key`, `.pem` and `_issuerCertificate.crt` into the mapping directory and runs the change script.

## Security note

Private keys are stored in `ssl.json` in the configuration directory. Keep that file safe, and make sure the repository is private if you put it under version control.

## Migrating from Lucky

Certificates that are already issued are imported as they are and are not requested again before they expire. ZeroSSL registers a new ACME account using the EAB credentials from Lucky. See the [Migration Guide](/en/docs/guide/migration) for details.
