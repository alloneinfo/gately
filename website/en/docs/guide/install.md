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

### systemd example

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Initial sign-in

The admin UI listens on port `16680` by default. The username is `admin`, and the initial password is written to the log on first start. After signing in for the first time, change the password and enable two-factor authentication.

## Upgrading

Gately has no built-in online self-update; upgrades are handled by your deployment method:

- Docker: `docker pull` the new image and recreate the container (keep the config directory mounted);
- Binary: replace the executable and restart the service.

Before upgrading, export a configuration backup or keep the config directory in git.
