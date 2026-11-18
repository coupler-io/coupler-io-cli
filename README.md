# coupler-io-cli

Command-line interface for [Coupler.io](https://www.coupler.io): manage dataflows, sources and destinations, and query the data they produce, from your terminal, scripts and CI.

It wraps the Coupler.io public API (`https://api.coupler.io`, [reference](https://api.coupler.io/api-docs)).

## Install

From source (needs Go 1.24+):

```bash
go install github.com/coupler-io/coupler-io-cli@latest   # note: Go names this binary coupler-io-cli, rename or alias it to coupler-io
# or
git clone git@github.com:coupler-io/coupler-io-cli.git && cd coupler-io-cli && make build   # ./coupler-io
```

Tagged releases publish `coupler-io` binaries for Linux, macOS and Windows (amd64/arm64) on the GitHub releases page.

Shell completion: `coupler-io completion zsh|bash|fish|powershell`.

## Authenticate

1. Copy your personal API token from the **API access** section of your [Coupler.io profile](https://app.coupler.io/app/profile). Your organization's plan must include the public API.
2. Run:

```bash
echo "$COUPLER_IO_TOKEN" | coupler-io configure --api-token-stdin --workspace-id 42
```

This validates the token, picks a workspace (the one from `--workspace-id`, or an interactive choice when the token has several), and saves both to `~/.coupler-io.yaml` with `0600` permissions.

`--api-token-stdin` keeps the token out of your shell history and the process list. Because it consumes stdin, pass `--workspace-id` together with it, otherwise the workspace prompt has nothing to read and the command asks you to pass the id. `--api-token <token>` also works, but the value is visible in `ps` and shell history; avoid it on shared machines. The two flags are mutually exclusive.

Settings are resolved in this order: flag, environment variable, config file.

| Flag | Env var | Config key | Purpose |
|------|---------|------------|---------|
| `--api-token` | `COUPLER_IO_API_TOKEN` | `api-token` | Personal API token |
| `--api-token-stdin` | | | Read the token from standard input |
| `-w`, `--workspace-id` | `COUPLER_IO_WORKSPACE_ID` | `workspace-id` | Workspace used by workspace-scoped commands |
| `--api-url` | `COUPLER_IO_API_URL` | `api-url` | API host (default `https://api.coupler.io`) |
| `-o`, `--output` | `COUPLER_IO_OUTPUT` | | `table` (default), `json`, `text`, `csv` |

In CI, skip `configure` and set `COUPLER_IO_API_TOKEN` and `COUPLER_IO_WORKSPACE_ID`, or pipe the token with `--api-token-stdin`.

### Config file

Only `~/.coupler-io.yaml` is read. A `.coupler-io.yaml` in the current directory is deliberately ignored, so a cloned repository cannot redirect `api-url` and capture your token.

A missing file is fine. A file that exists but cannot be read or parsed makes commands fail with an error instead of silently dropping the saved token. `configure` and `completion` still run in that case, so `coupler-io configure` is how you overwrite a broken file.

## Quick start

```bash
coupler-io workspaces list                       # find workspace ids
coupler-io dataflows list
coupler-io datasets list --dataflow-id <id>
coupler-io datasets get <dataset-id>             # columns and their DB names
coupler-io datasets query <dataset-id> -s 'select country, sum(revenue) from data group by 1' -o csv
coupler-io dataflows run <id>                    # queue a full run
```

## Commands

| Command | What it does |
|---------|--------------|
| `configure` | Validate a token and save token + workspace |
| `workspaces list` | List workspaces you can access |
| `dataflows list \| get \| setup \| create \| update \| run` | Manage dataflows. `update` renames and sets the schedule; `run --mode full\|preview` queues a run |
| `sources create \| update` | Add or change a dataflow source (params, credential, `--enabled`, `--lifecycle-state`) |
| `destinations create \| update` | Add or change a dataflow destination (params, credential, `--lifecycle-state`; there is no `--enabled`, use `--lifecycle-state disabled`) |
| `datasets list \| get \| create \| update \| query` | Browse datasets, add custom SQL datasets, run DuckDB SQL on the latest data |
| `datasets columns update` | Set column labels and descriptions |
| `datasets snapshots list \| get` | Datasets with data from the latest successful run |
| `integrations sources\|destinations list \| get` | Discover connectors and the params they accept |
| `integrations sources field-options` | Resolve dynamic param options (accounts, sheets, ...) |
| `credentials list` | Connected accounts usable by sources/destinations |
| `templates list` | Dataflow templates (use with `dataflows create --template-id`) |
| `skills list \| prompt` | Agent skills |

Run `coupler-io <command> --help` for flags and examples.

### Building a dataflow from scratch

```bash
coupler-io integrations sources list
coupler-io integrations sources get csv                      # which params does it take?
coupler-io credentials list --type destination

coupler-io dataflows create --name "Sales report"            # prints the new id
coupler-io sources create --dataflow-id <id> --integration csv --param url=https://example.com/data.csv
coupler-io destinations create --dataflow-id <id> --integration google_spreadsheet \
    --credential-id 15 --param sheet_name=Sales
coupler-io dataflows update <id> --interval daily --at 08:00 --timezone Europe/Berlin
coupler-io dataflows setup <id>                              # anything still missing?
coupler-io dataflows run <id>
```

### Passing params

- `--param key=value` is repeatable and always sends a string.
- `--params '{"limit": 10, "tags": ["a"]}'` sends typed JSON. Use `@file.json` to read a file or `-` for stdin.
- When both are given, `--param` wins for the same key.

### Reading values from files and stdin

Flags documented as "`@file` or `-`" (`--params`, `--statement`, `--query`, `--ai-context`, `--updates`) read the value from a file (`@path`) or from standard input (`-`).

Standard input can be read **once** per run. `--api-token-stdin` counts as a read, so it cannot be combined with a `-` value, and two `-` values in one command cannot both work. Both cases fail with a usage error (exit code `2`) that says so. Pass the other value as `@file` instead:

```bash
echo "$COUPLER_IO_TOKEN" | coupler-io datasets query <id> -s @report.sql --api-token-stdin
```

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
| `2` | Usage error, see [Validation](#validation): unknown command or flag, wrong arguments, missing required flag, flags used together wrongly, nothing to update, invalid `--output`, malformed `--param`/`--params`, stdin used twice |
| `128+N` | Interrupted by signal N (`130` for Ctrl-C, `143` for SIGTERM) |

An unknown subcommand of a group (for example `coupler-io dataflows nope`) is a usage error with a suggestion, not a silent help screen. Running a group with no arguments prints its help and exits `0`.

API rate limit: 100 requests per minute per organization (HTTP 429).

### Validation

Two places can reject a command, and each has one job. The rule: **the CLI checks how a command is used; the API checks the values.**

| | Checked by the CLI (exit `2`, no request is sent) | Checked by the API (exit `1`, the API's message is printed as is) |
|---|---|---|
| Examples | A required flag or argument is missing. Flags that must be used together or not together (`--active-hours-start` with `--active-hours-end`, `--updates` with `--key`). An update with no fields to change. Syntax the CLI has to parse itself: `--param key=value`, `--params` JSON, `@file`, `--output` format. Standard input used twice. | Allowed values and ranges (`--interval`, `--week-day`, `--month-day`, `--lifecycle-state`, `--mode`, `--type`), time and timezone formats, empty values, whether an id exists, credential access, plan limits (for example pausing), and anything that depends on the integration. |

Why this split:

- The API is the single source of truth. Enum lists and ranges copied into the CLI go stale when the API changes, and then the CLI rejects values the API accepts.
- Usage checks need no network round trip and do not depend on server state, so they are cheap, stable, and the CLI has to do them anyway to build a well-formed request. Some of them are the only protection: the API treats an update with nothing to change as a successful no-op.

When adding a command, follow the rule: send values through untouched and let the API answer. Do not add enum or range checks to the CLI. If a flag's allowed values help users, list them in the flag's help text.

Because `--enabled` is not read by the destinations API, it exists only on `sources update`.

## Development

```bash
make build     # ./coupler-io
make test      # go test -race ./... (CI and the release workflow run exactly this)
make lint      # gofmt + go vet
```

Layout follows [mailtrap-cli](https://github.com/mailtrap/mailtrap-cli):

```
main.go                      entry point, version injected by GoReleaser
cmd/root.go                  root command, global flags, command registration
internal/client              HTTP client and APIError
internal/config              flags/env/~/.coupler-io.yaml resolution
internal/cmdutil             Factory, path building, param parsing, shared printing
internal/output              table, text, json, csv formatters
internal/commands/<name>     one package per command group
internal/commands/dataflowparts   shared create/update commands behind sources and destinations
internal/cmdtest             fake API server helpers for tests
```

To add a command: create a package under `internal/commands/`, build it from `*cmdutil.Factory`, register it in `cmd/root.go`, and test it with `cmdtest.Setup`. Every command package has its own `_test.go`; put behaviour that only makes sense across commands (global flags, exit codes) in `cmd/`. Report usage mistakes with `cmdutil.UsageErrorf` (exit code 2), never with a plain `fmt.Errorf`.

Release: push a tag like `v0.1.0`; GitHub Actions runs GoReleaser.

## License

MIT
