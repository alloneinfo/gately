# FAQ

## Does it need to connect to the project's servers?

No. Gately does not connect to any project servers by default and does not report device information. Outbound addresses such as public IP lookup use third-party public services and can be changed in the corresponding tasks.

## Where is the initial password?

It is printed to the log on first start, only once. For Docker deployments, run `docker logs gately 2>&1 | grep -i "initial password"`.

## What if I forget the administrator password?

Stop Gately, delete `account.json` from the config directory, and start it again. The program recreates the `admin` account and prints a new initial password to the log. After signing in, review security settings such as two-factor authentication.

## Does it support fnOS?

Yes. Every release ships offline fnOS packages (`.fpk`) that you install manually from the App Center. Gately plugs into the fnOS unified gateway, so it opens from the desktop icon and also works over your remote fnOS domain. Turn on "Sign in with fnOS" under "Settings → Access control" and fnOS administrators who are signed in go straight in, without a password or two-step code. See [fnOS native app](/en/docs/guide/install#fnos).

## Why is `--network host` recommended?

Port forwarding, STUN and web services listen directly on host ports. With bridge networking you have to publish every port in use yourself.

## Can it run alongside Lucky?

Yes. The admin port defaults to 16680, which does not conflict with Lucky's 16601. Imported rules are disabled by default so they do not compete with a still-running Lucky for ports and router UPnP mappings.

## Why do I have to supply the Tailscale, frp and cloudflared binaries myself?

These three modules supervise the official external programs. Gately does not bundle them, so you can always use the latest upstream version. See [Configuration & Runtime Options](/en/docs/guide/configuration) for where to put them.

## My STUN rule is enabled but I still cannot connect from outside

If your router's WAN interface sits behind carrier-grade NAT (100.64.0.0/10), outside hosts cannot reach the traversed port. This is a limit of the network itself. Use [Tailscale / frp / Cloudflare Tunnel](/en/docs/modules/tunnels) instead.

## How do I report issues or contribute?

Open an issue on [GitHub Issues](https://github.com/wangwanjie/gately/issues). Development conventions are in [Contributing](/en/docs/guide/contributing).
