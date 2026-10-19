# Port Forwarding

Forward a port that Gately listens on to another device or service.

## Capabilities

- TCP and UDP;
- IPv4 and IPv6, with forwarding between v4 and v6;
- Port ranges and multiple ports;
- Multiple targets with failover;
- Connection limits;
- Traffic statistics and runtime logs;
- Sending the PROXY protocol (v1 / v2) to targets, passing the real client address downstream.

## PROXY protocol

Port forwarding only relays TCP, so by default the downstream sees only Gately's own address. Set "PROXY protocol" to v1 or v2 in a rule, and Gately sends a PROXY header right after connecting to the target. The downstream then gets the real client address for logging, rate limiting and access control.

The target must enable the matching receive option, otherwise the connection is rejected. For example:

- haproxy: `bind 192.168.1.8:4434 accept-proxy`;
- nginx: `listen 4435 ssl proxy_protocol;`;
- Gately Web Services: enable "Accept PROXY protocol" on the listener, and add the source to the trusted PROXY sources.

This option applies to TCP only; UDP forwarding is unaffected.

## Typical scenarios

- Expose NAS service ports through a single entry point;
- Bridge IPv6 and IPv4 traffic;
- Configure a primary and a standby backend for one service.

## Deployment tip

Port forwarding listens on host ports directly. For Docker deployments, `--network host` is recommended.

## Migrating from Lucky

Rules can be imported from `lucky_portforward.lkcf`. Options such as TLS / stream encryption and rate limiting are not supported yet, and a notice is shown during import.
