# Configuration & Runtime Options

## Command-line flags

| Flag | Description |
| --- | --- |
| `-c` | Config directory; can also be set with the `GATELY_CONFIG_DIR` environment variable |
| `-listen` | Override the admin UI listen address, e.g. `:16680` |
| `-apply` | Declarative config directory; can also be set with the `GATELY_APPLY_DIR` environment variable. See below |
| `-version` | Print version information |

| Environment variable | Description |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | Initial administrator password on first start (at least 8 characters); defaults to `gately.cn` when unset; has no effect once the account exists |

## Config directory

All configuration lives in the config directory, with one plain-text `<name>.json` file per module, written atomically with file mode 0600. This means you can:

- Inspect it with any text editor;
- Put the directory in git, audit every change with diffs and roll back at any time;
- Back up by simply copying the directory.

::: warning Note
`ssl.json` contains certificate private keys. Before committing it to version control, make sure the repository is private, or add the file to your ignore list.
:::

## Declarative configuration

With `-apply <dir>`, every start copies the `<name>.json` files in that directory over the config directory; files with identical content are not rewritten. This suits distributing configuration from a git repository, a Kubernetes ConfigMap or a deployment script.

- Each file is validated as JSON before writing; if any file is invalid, Gately refuses to start;
- `account.json` and `tokens.json` cannot be overwritten; accounts and tokens can only be managed in the admin UI;
- The overwritten content is saved in the [config history](/en/docs/guide/audit) and can be rolled back at any time.

## External program directory

The Tailscale, frp and cloudflared modules supervise the official external programs. Gately looks for each executable in this order:

1. The path set in the module settings;
2. `PATH`;
3. The `bin/` subdirectory of the config directory.

For Docker deployments, put the binaries in the mounted config directory, for example `./gately-data/bin/frpc`. Tailscale needs both `tailscale` and `tailscaled`. See [Tailscale / frp / cloudflared](/en/docs/modules/tunnels) for details.

## Runtime logs

System and module runtime logs are written to `logs/` in the configuration directory by default, so they are still available on the System logs page and each module page after a restart or upgrade:

- One JSON Lines file per source: `logs/<module ID>.log`, with system logs in `logs/gately.log`. On startup the most recent entries, up to "Log retention lines", are restored from the file;
- New entries are written in batches at most 1 second later. A file is rotated to `.1` once it exceeds the limit (2 MB by default) and only that one previous file is kept, so each source uses at most twice the limit;
- Under Settings → Network & Runtime → Runtime parameters you can turn off "Persist logs" or change "Log file limit", effective immediately. When off, logs are kept in memory only and cleared on restart;
- Clearing a module's logs in the UI also deletes its files;
- Configuration backups do not include `logs/` or the audit log `audit.log`.

## Backup and restore

"Settings" provides configuration backup and restore, and is also the entry point for importing Lucky configurations.
