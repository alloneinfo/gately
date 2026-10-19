# Installation

## Docker

Image: `ghcr.io/wangwanjie/gately`, available for amd64, arm64 and arm/v7.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # Port forwarding, STUN traversal and web services all listen on host ports directly
    network_mode: host
    volumes:
      # Config directory (plain JSON, safe to keep in git; ssl.json contains certificate private keys)
      - ./data:/etc/gately
      # Docker management (remove if not needed)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

The image ships with `sh`, `bash` and `curl`, so typical notification scripts run out of the box.

## Binary

1. Get the archive for your platform from the [download page](/en/download): Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows or FreeBSD.
2. Extract it and run:

```bash
./gately -c /etc/gately
```

### systemd

The repository ships a complete service file, [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), which already grants the capability needed to listen on low ports:

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Use the procd init script [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init):

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Initial sign-in

The admin UI listens on port `16680` by default. The username is `admin`, and the initial password is written to the log on first start. For automated deployments, set the initial password with the `GATELY_INITIAL_PASSWORD` environment variable; no random password is generated in that case.

The first sign-in shows an onboarding guide that walks you through changing the password, enabling two-factor authentication or a Passkey, running the network self-check, importing from Lucky and setting up notifications.

## Upgrading

Gately has no built-in online self-update; upgrades are handled by your deployment method:

- Docker: `docker pull` the new image and recreate the container (keep the config directory mounted);
- Binary: replace the executable and restart the service.

Before upgrading, export a configuration backup or keep the config directory in git. Downloaded archives and images can be checked against their signatures as described in [Verifying Releases](/en/docs/guide/verify).
