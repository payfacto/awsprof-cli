# awsprof - AWS profile switcher CLI

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/payfacto/awsprof-cli)](https://goreportcard.com/report/github.com/payfacto/awsprof-cli)

A small Go CLI for picking an AWS profile to log in as, from a list of the
profiles available on your machine. Built for humans at the terminal and for
scripting/agents.

## What it does

AWS credentials and named profiles live in the shared config files
(`~/.aws/config` and `~/.aws/credentials`). Switching between them by hand -
remembering names, exporting `AWS_PROFILE`, running `aws sso login --profile X` -
is fiddly. `awsprof` reads the available profiles, lets you pick or name one,
logs in via SSO device authorization only when the cached token is stale, and
sets `AWS_PROFILE` in your current shell - so "which account am I in?" becomes
a single command instead of a grep through dotfiles.

`awsprof` never shells out to the `aws` CLI; SSO login and identity checks go
directly through `aws-sdk-go-v2`. Its SSO token cache is aws-CLI-compatible, so
a subsequent `aws sts get-caller-identity` (or any other AWS CLI/SDK call)
reuses the same cached token.

## Install

### Homebrew (macOS / Linux)

```bash
brew tap payfacto/tap
brew install payfacto/tap/awsprof
```

### Pre-built binaries

Download an archive for your OS and architecture from the
[Releases page](https://github.com/payfacto/awsprof-cli/releases), extract it,
and place the `awsprof` binary somewhere on your `PATH`.

### From source

```bash
go install github.com/payfacto/awsprof-cli@latest

# Or build locally
go build -o awsprof .
```

## Setup: install the shell hook

`awsprof` activates a profile by printing an `export AWS_PROFILE=...` (or
shell-appropriate equivalent) line to stdout - that line has to be `eval`'d by
your shell to actually take effect. A one-time shell hook does that for you.
Add the line for your shell to its startup file:

- **bash** - add to `~/.bashrc`: `eval "$(awsprof shell-init bash)"`
- **zsh** - add to `~/.zshrc`: `eval "$(awsprof shell-init zsh)"`
- **fish** - add to `~/.config/fish/config.fish`: `awsprof shell-init fish | source`
- **PowerShell** - add to `$PROFILE`: `awsprof shell-init powershell | Out-String | Invoke-Expression`

Open a new shell (or re-source the file) and `awsprof` is ready to use.

## Usage

```bash
awsprof                                        # interactive picker, then login if needed, then activate
awsprof <profile>                              # resolve profile (exact name, then configured prefix), login if needed, activate
awsprof list [--plain]                         # list profiles; marks the active one; --plain prints bare names
awsprof whoami                                 # show current identity (AWS_PROFILE, account, ARN); does not switch
awsprof shell-init <bash|zsh|fish|powershell>  # print the shell hook (see Setup above)
awsprof use <profile>                          # hidden alias of `awsprof <profile>`
awsprof --version                              # print version
awsprof --help                                 # help
```

A persistent `--shell` flag (default `bash`) selects the export syntax; the
installed shell hook sets it for you automatically, so you normally never pass
it by hand.

### Activation model: stdout vs. stderr

`awsprof` writes exactly one thing to stdout on a successful activation: the
`export AWS_PROFILE=...` line (or the fish/PowerShell equivalent). Everything
else - the interactive picker, login progress, the resolved account/ARN
confirmation, and all errors - goes to stderr. This keeps stdout safe to `eval`
and keeps `awsprof list --plain` and other data commands script-friendly
(their normal output passes through stdout unevaluated). Without the shell
hook installed, running the raw `awsprof` binary only prints the export line;
it cannot modify your current shell's environment on its own.

### Profile discovery and short names

Profiles are discovered from the standard AWS locations:

- `~/.aws/config` (sections like `[profile dev]`, plus `[sso-session ...]`)
- `~/.aws/credentials` (sections like `[default]`, `[work]`)
- Overridable via the `AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE`
  environment variables.

`awsprof` reads these files; it does not write your long-lived credentials into
its own config. For SSO profiles, browser-based device login only runs when
the cached SSO token is stale; a valid cached token is reused.

`awsprof <name>` resolves `name` to a profile by trying an exact match first,
then each prefix in the `prefixes` list from `~/.awsprof.yaml` (default:
`payfacto-`), so `awsprof dev` can match a profile literally named `dev`, or
(if no exact match exists) `payfacto-dev`.

### Environment coloring

`list`, the interactive picker, and `whoami` color the environment segment of a
profile name so you can tell prod from dev at a glance. The environment is
detected from a hyphen-delimited segment of the name (case-insensitive,
whole-segment match, first match wins):

| Environment | Color | Recognized segments |
| --- | --- | --- |
| prod | bold red | `prod`, `production` |
| staging | orange | `staging`, `stage`, `stg` |
| uat | purple | `uat` |
| qa | yellow | `qa` |
| dev | green | `dev`, `development` |
| sandbox | blue | `sandbox`, `test`, `sbx` |

Only the matched segment is colored; names with no recognized environment (e.g.
`payfacto-synapse`) are left plain. Color is emitted only when the output is an
interactive terminal - it is disabled when `NO_COLOR` is set, when output is
piped or redirected, and always for `awsprof list --plain` (which stays
byte-clean for scripting).

## Development

```bash
go build -o awsprof .     # build (version reports as "dev")
make build                # build with a git-derived version stamp
go test ./...             # run all tests
```

Standard `go fmt` and `go vet` apply.

## Releasing

Releases are automated by GitHub Actions ([`.github/workflows/release.yml`](.github/workflows/release.yml))
plus [GoReleaser](https://goreleaser.com/) ([`.goreleaser.yaml`](.goreleaser.yaml)):
pushing a tag that starts with `v` builds Linux/macOS/Windows (amd64 + arm64),
publishes a GitHub Release, and bumps the
[payfacto Homebrew tap](https://github.com/payfacto/homebrew-tap). The full
runbook is in [`.context/GO-RELEASE-PATTERNS.md`](.context/GO-RELEASE-PATTERNS.md).

## License

[Apache License 2.0](LICENSE).
