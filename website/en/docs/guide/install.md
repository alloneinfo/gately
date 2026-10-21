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

## fnOS native app {#fnos}

Every release ships offline packages for fnOS: `gately_<version>_fnos_x86.fpk` (Intel / AMD) and `gately_<version>_fnos_arm.fpk` (ARM). They are not listed in the official App Center, so install them manually:

1. Get the `.fpk` for your device's architecture from the [download page](/en/download#fnos);
2. In fnOS, open "App Center → Manual install" (the button at the bottom-left), pick the file and finish the wizard;
3. Click the Gately icon on the fnOS desktop to open the admin UI.

### Works with the fnOS unified gateway

The desktop icon opens the admin UI through the fnOS unified gateway at `/app/gately/` on the address you use for fnOS, sharing the same entry point. Both the LAN address and your remote fnOS domain work, with no need to open or forward port 16680.

Access through the gateway is controlled by fnOS, and only fnOS administrators can open it. The secure entrance path and "LAN-only access" apply only to direct access on port 16680. On the LAN, `http://<NAS address>:16680` still works.

### Sign in with your fnOS administrator session

Turn on "Sign in with fnOS" under "Settings → Access control" and administrators who are signed in to fnOS enter Gately straight from the desktop icon, without a password or two-step code:

- Only fnOS administrators are signed in, as the Gately administrator; regular fnOS users still enter a username and password;
- It applies only to access through the unified gateway; direct access on port 16680 still requires a normal sign-in;
- After you sign out, Gately does not sign you back in silently; the login page shows a "Sign in as fnOS user &lt;name&gt;" button instead;
- The setting is off by default and only shown in the fnOS package.

### Notes

- The app runs as `root`, so it can listen on 80/443, manage Docker through `docker.sock` and open a real Web terminal. Install it only on a NAS you control.
- On the first start the initial `admin` password is written to `/var/apps/gately/var/gately.log`: `grep -i "initial password" /var/apps/gately/var/gately.log`.
- Config lives in the app's `var` directory and is kept on uninstall unless you choose "delete all" in the uninstall wizard.
- The package also adds a `/usr/local/bin/gately` command for `gately ctl` and `gately mcp`.
- Port 16680 is shared with the Docker image. Stop the `gately` container before starting the native app.

## Initial sign-in

The admin UI listens on port `16680` by default. The username is `admin`, and the initial password is written to the log on first start. For automated deployments, set the initial password with the `GATELY_INITIAL_PASSWORD` environment variable; no random password is generated in that case.

The first sign-in shows an onboarding guide that walks you through changing the password, enabling two-factor authentication or a Passkey, running the network self-check, importing from Lucky and setting up notifications.

## Upgrading

### Online upgrade {#online-upgrade}

Linux binary installs (systemd, PVE LXC containers, OpenWrt and so on) can upgrade online from the About page of the admin UI: click "Check for Updates", then "Upgrade now" when a new version is found. This also works for LXC containers created in PVE from the OCI image, so there is no need to delete and recreate them. Gately will:

1. Download `gately_<version>_linux_<arch>.tar.gz` for this platform from GitHub;
2. Verify its SHA-256 against the `checksums.txt` of the same release;
3. Test-run the new binary with `-version` to make sure it runs on this machine;
4. Replace the executable and restart with the same arguments, keeping the previous version as `gately.old` in the same directory.

Everything happens in a dialog: it first shows the release notes, then the current step, download progress and speed once the upgrade starts. Until the binary is replaced (fetching, downloading and verifying) you can click "Cancel upgrade" at any time, which leaves the current binary untouched.

If GitHub is hard to reach, set an **update proxy** on the About page or under Settings → Network & Runtime. It supports `http://`, `https://` and `socks5://`, and both the update check and the download go through it. When empty, Gately connects directly and still honors the `HTTPS_PROXY` environment variable.

Without internet access, get the tar.gz from the [download page](/en/download) and click "Upload upgrade package" on the About page. Uploads accept any version, so you can also roll back to an older release.

Online upgrades only run when an admin starts them; Gately never downloads or installs a version on its own. If the new version fails to start, roll back manually with the kept copy (systemd example):

```bash
mv /usr/local/bin/gately.old /usr/local/bin/gately
systemctl restart gately
```

If you granted low-port access to the binary with `setcap`, run `setcap` again after the upgrade.

### Other deployments

- Docker: `docker pull` the new image and recreate the container (keep the config directory mounted);
- fnOS: manually install the newer `.fpk` from the App Center; your config is kept;
- Windows, macOS, FreeBSD: replace the executable and restart the service.

Before upgrading, export a configuration backup or keep the config directory in git. Downloaded archives and images can be checked against their signatures as described in [Verifying Releases](/en/docs/guide/verify).
