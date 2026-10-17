# Dynamic DNS (DDNS)

The public IP of a home broadband connection changes over time. DDNS updates your DNS records automatically after an IP change, so your domain always points to your current address.

## Supported providers

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, and a custom callback (for providers that are not built in, or for self-hosted endpoints).

## How the IP is obtained

- **URL**: request a public-IP lookup address (replace it with any service you like, or host your own);
- **Network interface**: read the address of a specific network interface directly, suited to devices that hold a public IP;
- **Command**: run a command and use its output, suited to unusual network setups.

Both IPv4 and IPv6 are supported.

## Notifications

A Webhook can be called when the IP changes or a sync fails.

## Recommendations

- Create a sub-account key limited to DNS record editing for your DNS provider. Avoid using the main account key;
- After creating a task, click "Force sync" to confirm that the record has been updated;
- The public-IP lookup address no longer depends on any service operated by the project, and you can change it in a task at any time.

## Migrating from Lucky

DDNS tasks can be imported from `lucky_ddns.lkcf`, covering 11 providers. Lucky's official IP lookup addresses are removed during import. See the [Migration Guide](/en/docs/guide/migration) for details.
