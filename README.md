# coupler-io-cli

Command-line interface for [Coupler.io](https://www.coupler.io): manage dataflows, sources and destinations, and query the data they produce, from your terminal, scripts and CI.

It wraps the Coupler.io public API (`https://api.coupler.io`, [documentation](https://docs.coupler.io/api)).

## Install

From source (needs Go 1.24+):

```bash
go install github.com/coupler-io/coupler-io-cli@latest   # installs a binary named coupler-io-cli
# or
git clone git@github.com:coupler-io/coupler-io-cli.git && cd coupler-io-cli && make build   # ./coupler
```

Tagged releases publish `coupler` binaries for Linux, macOS and Windows (amd64/arm64) on the GitHub releases page.

Shell completion: `coupler completion zsh|bash|fish|powershell`.

## Authenticate

1. Copy your personal API token from the [API page](https://app.coupler.io/app/api) in Coupler.io. Your organization's plan must include the public API.
2. Find your workspace id:

```bash
COUPLER_API_TOKEN="$COUPLER_TOKEN" coupler workspaces list
```

   If your token has a single workspace you can skip this step and the `--workspace-id` flag below: `configure` picks it automatically and prints its id.
3. Run:

```bash
echo "$COUPLER_TOKEN" | coupler configure --api-token-stdin --workspace-id 42
```

This validates the token, picks a workspace (the one from `--workspace-id`, or an interactive choice when the token has several), and saves both to `~/.coupler.yaml` with `0600` permissions.

`--api-token-stdin` keeps the token out of your shell history and the process list. Because it consumes stdin, pass `--workspace-id` together with it, otherwise the workspace prompt has nothing to read and the command asks you to pass the id. `--api-token <token>` also works, but the value is visible in `ps` and shell history; avoid it on shared machines. The two flags are mutually exclusive.

Settings are resolved in this order: flag, environment variable, config file.

| Flag | Env var | Config key | Purpose |
|------|---------|------------|---------|
| `--api-token` | `COUPLER_API_TOKEN` | `api-token` | Personal API token |
| `--api-token-stdin` | | | Read the token from standard input |
| `-w`, `--workspace-id` | `COUPLER_WORKSPACE_ID` | `workspace-id` | Workspace used by workspace-scoped commands |
| `--api-url` | `COUPLER_API_URL` | `api-url` | API host (default `https://api.coupler.io`) |
| `-o`, `--output` | `COUPLER_OUTPUT` | | `table` (default), `json`, `text`, `csv` |

In CI, skip `configure` and set `COUPLER_API_TOKEN` and `COUPLER_WORKSPACE_ID`, or pipe the token with `--api-token-stdin`.

### Config file

Only `~/.coupler.yaml` is read. A `.coupler.yaml` in the current directory is deliberately ignored, so a cloned repository cannot redirect `api-url` and capture your token.

A missing file is fine. A file that exists but cannot be read or parsed makes commands fail with an error instead of silently dropping the saved token. `configure` and `completion` still run in that case, so `coupler configure` is how you overwrite a broken file.

## Quick start

```bash
coupler workspaces list                       # find workspace ids
coupler dataflows list
coupler datasets list --dataflow-id <id>
coupler datasets get <dataset-id>             # columns and their DB names
coupler datasets query <dataset-id> -s 'select country, sum(revenue) from data group by 1' -o csv
coupler dataflows run <id>                    # queue a full run
```

## Commands

| Command | What it does |
|---------|--------------|
| `configure` | Validate a token and save token + workspace |
| `workspaces list` | List workspaces you can access |
| `dataflows list \| get \| setup \| create \| update \| run` | Manage dataflows. `update` renames and sets the schedule; `run --mode full\|preview` queues a run |
| `sources create \| update` | Add or change a dataflow source (params, credential, enabled, paused) |
| `destinations create \| update` | Add or change a dataflow destination |
| `datasets list \| get \| create \| update \| query` | Browse datasets, add custom SQL datasets, run DuckDB SQL on the latest data |
| `datasets columns update` | Set column labels and descriptions |
| `datasets snapshots list \| get` | Datasets with data from the latest successful run |
| `integrations sources\|destinations list \| get` | Discover connectors and the params they accept |
| `integrations sources field-options` | Resolve dynamic param options (accounts, sheets, ...) |
| `credentials list` | Connected accounts usable by sources/destinations |
| `templates list` | Dataflow templates (use with `dataflows create --template-id`) |
| `skills list \| prompt` | Agent skills |

Run `coupler <command> --help` for flags and examples.

### Building a dataflow from scratch

```bash
coupler integrations sources list
coupler integrations sources get csv                      # which params does it take?
coupler credentials list --type destination

coupler dataflows create --name "Sales report"            # prints the new id
coupler sources create --dataflow-id <id> --integration csv --param url=https://example.com/data.csv
coupler destinations create --dataflow-id <id> --integration google_spreadsheet \
    --credential-id 15 --param sheet_name=Sales
coupler dataflows update <id> --interval daily --at 08:00 --timezone Europe/Berlin
coupler dataflows setup <id>                              # anything still missing?
coupler dataflows run <id>
```

### Passing params

- `--param key=value` is repeatable and always sends a string.
- `--params '{"limit": 10, "tags": ["a"]}'` sends typed JSON. Use `@file.json` to read a file or `-` for stdin.
- When both are given, `--param` wins for the same key.

### Output formats

- `table` / `text` are for humans. Wide values are shortened in tables.
- `json` is the untouched API response, including fields the table hides, totals and metadata. Use it for scripting with `jq`.
- `csv` prints the rows only, which suits `datasets query` and `datasets list`.

### Errors and exit codes

Errors go to stderr (`Error: ...`); results go to stdout.

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Runtime error: API error, network failure, unreadable or invalid config |
| `2` | Usage error: unknown command or flag, wrong arguments, missing required flag, invalid `--output`, bad `--param`/`--params` |
| `128+N` | Interrupted by signal N (`130` for Ctrl-C, `143` for SIGTERM) |

An unknown subcommand of a group (for example `coupler dataflows nope`) is a usage error with a suggestion, not a silent help screen. Running a group with no arguments prints its help and exits `0`.

API rate limit: 100 requests per minute per organization (HTTP 429).

## Development

```bash
make build     # ./coupler
make test      # go test -race ./...
make lint      # gofmt + go vet
```

Project layout:

```
main.go                      entry point, version injected by GoReleaser
cmd/root.go                  root command, global flags, command registration
internal/client              HTTP client and APIError
internal/config              flags/env/~/.coupler.yaml resolution
internal/cmdutil             Factory, path building, param parsing, shared printing
internal/output              table, text, json, csv formatters
internal/commands/<name>     one package per command group
internal/cmdtest             fake API server helpers for tests
```

To add a command: create a package under `internal/commands/`, build it from `*cmdutil.Factory`, register it in `cmd/root.go`, and test it with `cmdtest.Setup`.

Release: push a tag like `v0.1.0`; GitHub Actions runs GoReleaser.

## License

MIT
