# Port Forwarding

Forward a port that Gately listens on to another device or service.

## Capabilities

- TCP and UDP;
- IPv4 and IPv6, with forwarding between v4 and v6;
- Port ranges and multiple ports;
- Multiple targets with failover;
- Connection limits;
- Traffic statistics and runtime logs.

## Typical scenarios

- Expose NAS service ports through a single entry point;
- Bridge IPv6 and IPv4 traffic;
- Configure a primary and a standby backend for one service.

## Deployment tip

Port forwarding listens on host ports directly. For Docker deployments, `--network host` is recommended.

## Migrating from Lucky

Rules can be imported from `lucky_portforward.lkcf`. Options such as TLS / stream encryption and rate limiting are not supported yet, and a notice is shown during import.
