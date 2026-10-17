# Configuration & Runtime Options

## Command-line flags

| Flag | Description |
| --- | --- |
| `-c` | Config directory; can also be set with the `GATELY_CONFIG_DIR` environment variable |
| `-listen` | Override the admin UI listen address, e.g. `:16680` |
| `-version` | Print version information |

## Config directory

All configuration lives in the config directory, with one plain-text `<name>.json` file per module, written atomically with file mode 0600. This means you can:

- Inspect it with any text editor;
- Put the directory in git, audit every change with diffs and roll back at any time;
- Back up by simply copying the directory.

::: warning Note
`ssl.json` contains certificate private keys. Before committing it to version control, make sure the repository is private, or add the file to your ignore list.
:::

## External program directory

The Tailscale, frp and cloudflared modules supervise the official external programs. Gately looks for each executable in this order:

1. The path set in the module settings;
2. `PATH`;
3. The `bin/` subdirectory of the config directory.

For Docker deployments, put the binaries in the mounted config directory, for example `./gately-data/bin/frpc`. Tailscale needs both `tailscale` and `tailscaled`. See [Tailscale / frp / cloudflared](/en/docs/modules/tunnels) for details.

## Backup and restore

"Settings" provides configuration backup and restore, and is also the entry point for importing Lucky configurations.
