# mmrun

[![CI](https://github.com/isdmx/mmrun/actions/workflows/ci.yml/badge.svg)](https://github.com/isdmx/mmrun/actions/workflows/ci.yml)
[![Release](https://github.com/isdmx/mmrun/actions/workflows/release.yml/badge.svg)](https://github.com/isdmx/mmrun/actions/workflows/release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/isdmx/mmrun.svg)](https://pkg.go.dev/github.com/isdmx/mmrun)

A scriptable, git-style command-line client for [Mattermost](https://mattermost.com).
Authenticate once (password + 2FA, personal access token, or SSO), then read,
post, tail, and search from your terminal — with human, AI-friendly, or JSON output.

## Features

- **Persistent auth** — password + MFA/2FA, personal access token, or SSO/OAuth; session stored under XDG with `0600` permissions.
- **Read & post** — channels and direct messages, threaded replies, file attachments.
- **Live tail** — stream new messages over WebSocket (`Ctrl-C` to stop).
- **Search** — server-side message search with Mattermost modifiers, plus channel and user search.
- **Followed threads** — list the threads you follow with reply/unread counts.
- **Actionable output** — results carry channel, author, `post_id`, thread root, and a clickable permalink.
- **Three output modes** — colored human, plain AI-friendly, and JSON (auto-detected by TTY).

## Install

With Go:

```sh
go install github.com/isdmx/mmrun@latest
```

Or download a prebuilt binary from the [releases page](https://github.com/isdmx/mmrun/releases).

## Quick start

```sh
# Log in (prompts for server, credentials, and MFA if required)
mmrun auth login --server https://mattermost.example.com

# Who am I?
mmrun me

# List teams and channels
mmrun team
mmrun channel

# Read and post
mmrun read town-square --limit 20
mmrun post town-square "Deploy finished :rocket:"

# Direct message a user
mmrun user search alice
mmrun post @alice "ping"

# Search and follow along
mmrun search "from:bob deploy failed"
mmrun tail incidents
```

## Commands

| Command | Description |
|---|---|
| `auth login` | Log in via password+MFA, `--token <PAT>`, or `--sso <provider>` |
| `auth logout` | Revoke the session server-side and delete local credentials |
| `auth status` | Show the current session (server, user, expiry) |
| `me [--profile]` | Show your account, status, timezone, and custom status |
| `team` / `team list` | List teams you belong to |
| `channel` / `channel list` | List channels (`--type public\|private\|dm\|group\|all`) |
| `channel search <term>` | Find channels by name, including ones you have not joined |
| `user search <term>` | Find users by name/username |
| `read <channel>` | Fetch messages (`--limit`, `--since 24h`, `--thread <id>`, `--full`) |
| `post <channel> <msg>` | Post a message (`--reply-to <id>`, `--file <path>`) |
| `tail <channel>` | Stream new messages live until interrupted |
| `search <query>` | Server-side message search (`--team`, `--full`) |
| `thread` / `thread list` | List followed threads (`--unread`, `--limit`) |
| `file download <id>` | Download a post's attachments or a single file (`--out <dir>`) |
| `file upload <channel> <path>` | Upload a file with an optional `--message` |
| `version` / `--version` | Print version, commit, and build date |

**Channel references** accept several forms: a bare name (`python`, resolved
against your team), `team/channel`, `@username` for a DM, or a raw channel ID.
Channel-taking commands also accept `--team` to qualify a bare name.

## Output modes

Use the global `-o` / `--output` flag: `auto` (default), `human`, `ai`, or `json`.

- `auto` → `human` (colored) on a terminal, `ai` (plain) when piped.
- `ai` → tab-separated `key=value` fields, no ANSI — friendly for tools/LLMs.
- `json` → structured output for scripting (`mmrun -o json search foo | jq`).

## Configuration

Files follow the XDG Base Directory specification:

| File | Location | Notes |
|---|---|---|
| Preferences | `$XDG_CONFIG_HOME/mmrun/config.toml` | `server_url`, `default_team`, `output_mode` |
| Session | `$XDG_STATE_HOME/mmrun/session.json` | token + user; `0600`, managed by `auth` |
| Downloads | `$XDG_DOWNLOAD_DIR` (or `~/Downloads`) | default target for `file download` |

`config.toml` is safe to edit by hand; `session.json` holds your secret token and is not meant for manual editing.

## Development

```sh
make help        # list all targets
make ci          # tidy-check + fmt-check + vet + lint + test-race
make build       # build ./bin/mmrun with version metadata
make test-race   # tests with the race detector + coverage
make lint        # golangci-lint (strict config in .golangci.yml)
make hooks       # install git hooks via prek (.pre-commit-config.yaml)
```

Requires Go (see `go.mod`), [golangci-lint](https://golangci-lint.run) v2, and
[gofumpt](https://github.com/mvdan/gofumpt); `make tools` installs them.
Pre-commit hooks (format, vet, lint, mod-tidy on commit; tests on push) run via
[`prek`](https://github.com/j178/prek) or `pre-commit`.

## Releasing

Releases are automated with [GoReleaser](https://goreleaser.com): pushing a
`v*` tag runs the release workflow, which cross-compiles binaries, builds
archives and checksums, generates a changelog, and publishes a GitHub Release.

```sh
git tag v0.1.0
git push origin v0.1.0
```

Build metadata (`version`, `commit`, `date`) is injected via `-ldflags` and
shown by `mmrun --version`.

## License

Not yet specified. (Add a `LICENSE` file to set the project's license.)
