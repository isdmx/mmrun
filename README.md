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
| `read <channel>` | Fetch messages (`--limit`, `--since 24h`, `--thread <id>`, `--full`, `--columns`, `--mark-read, --style chat|tree|table, --time-format relative`) |
| `post <channel> <msg>` | Post a message; use `-` to read from stdin (`--reply-to <id>`, repeatable `--file <path>`, `--dry-run`) |
| `tail <channel>` | Stream new messages live (--mentions-only, --from <user>) |
| `search <query>` | Server-side message search (`--team`, `--full`, `--columns`) |
| `thread` / `thread list` | List followed threads (`--unread`, `--limit`, `--columns`) |
| `thread read <id> --mark-read` | Read a thread, optionally mark it read |
| `react add <post-id> <emoji>` | Add a reaction |
| `react remove <post-id> <emoji> --yes` | Remove your reaction (requires `--yes`) |
| `pin add <post-id>` / `pin remove <post-id> --yes` | Pin or unpin a post |
| `pinned <channel>` | List pinned posts in a channel |
| `stats <channel>` | Show channel member/pinned counts |
| `flagged [--team] [--limit]` | List posts you flagged |
| `flag add <post-id>` / `flag remove <post-id> --yes` | Flag or unflag a post |
| `copy <post-id>` | Copy post permalink to clipboard |
| `edit edit <post-id> <msg>` | Edit a post's text |
| `edit delete <post-id> --yes` | Delete a post (requires `--yes`) |
| `mentions [--team] [--limit]` | Search posts that mention you |
| `reply <post-id> <msg>` | Reply to a post in its channel (`--file`, `--dry-run`) |
| `mark-read <id> [--type]` | Mark a channel or thread as read |
| `file download <id>` | Download a post's attachments or a single file (`--out <dir>`) |
| `file upload <channel> <path>...` | Upload one or more files (`--message`, `--dry-run`) |
| `open <id>` | Open a post or channel in the browser |
| `read <channel> --tail` | Fetch messages then enter live-stream mode |
| `post <channel> [--editor] <msg>` | Post a message; `--editor` opens `$EDITOR` |
| `context list/add/use/remove` | Manage session contexts (multi-account) |
| `config` | View/edit configuration (`path`, `list`, `get`, `set`, `generate`) |
| `version` / `--version` | Print version, commit, and build date |

**Color themes** — `config set theme dark|light|minimal` (or `--color auto|always|never`).
Dark is the default. Themes drive username colors, timestamps, channel names,
code-block syntax highlighting (via chroma), and more. Human-mode only.

**Message styles** — `--style table|chat|tree` controls layout. `chat` shows user+time header + message body + metadata footer. `tree` adds ●/↳ thread markers. Config: `config set style chat`.

**Docker** — `docker pull ghcr.io/isdmx/mmrun`.


**Output formats** — `read`, `search`, `thread read`, and `mentions` accept
`--format table|tree` and `--threads-only`. Tree mode shows replies indented
under root posts with `●`/`↳` markers. Set the default via
`config set format tree`.

**Shell completion** — `mmrun read <tab>` completes channel names and
`@usernames`; `mmrun post <tab>` does the same; post-ID commands complete
thread IDs. Flag completion for `--team`. Enable with:
```sh
source <(mmrun completion bash)   # or zsh, fish
```

### CI / automation
Set `MMRUN_URL` and `MMRUN_TOKEN` to skip `auth login` entirely:
```sh
export MMRUN_URL=https://mattermost.example.com
export MMRUN_TOKEN=<your-token>
mmrun post town-square "deploy finished"
```
Env auth is ephemeral — no session file is written. Re-login on 401 is
automatically suppressed when using env auth.

**Channel references** accept several forms: `~channel` (matches across teams),
`team/channel`, `@username`, a bare email (opens DM), a bare channel name
(resolved against your team; falls back to DM by username if not found as a
channel), or a raw 26-char channel ID (with user-ID fallback).

**Columns** — `read`, `search`, and `thread` accept `--columns` to choose output
columns: a full list (`--columns time,user,message`) or add/remove from the
default (`--columns -permalink,-root_id`). Unknown names error with the valid
set.

**Dry run** — `post` and `file upload` accept `--dry-run` to resolve the target
and preview what would be sent without posting or uploading.

## Output modes

Use the global `-o` / `--output` flag: `auto` (default), `human`, `ai`, or `json`.

- `auto` → `human` (colored) on a terminal, `ai` (plain) when piped.
- `ai` → tab-separated `key=value` fields, no ANSI — friendly for tools/LLMs.
- `json` → structured output for scripting (`mmrun -o json search foo | jq`).

## Configuration

Files follow the XDG Base Directory specification:

| File | Location | Notes |
|---|---|---|
| Preferences | `$XDG_CONFIG_HOME/mmrun/config.toml` | see keys below |
| Session | `$XDG_STATE_HOME/mmrun/session.json` | token + user; `0600`, managed by `auth` |
| Downloads | `$XDG_DOWNLOAD_DIR` (or `~/Downloads`) | default target for `file download` |

Manage `config.toml` with the `config` command instead of editing by hand:

```sh
mmrun config generate          # write a commented default file
mmrun config list              # show all settings and effective values
mmrun config set default_team sberdevices
mmrun config get color
mmrun config path
```

Keys: `server_url`, `default_team`, `output_mode` (auto|human|ai|json),
`default_limit` (message page size), `preview_len` (message preview length),
`color` (auto|always|never; `NO_COLOR` is also honored), `download_dir`,
`columns` (default columns for `read`/`search`). Precedence is
**flag > config > default**. `session.json` holds your secret token and is not
meant for manual editing.

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
