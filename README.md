# awsprof - AWS profile switcher CLI

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/payfacto/awsprof-cli)](https://goreportcard.com/report/github.com/payfacto/awsprof-cli)

A small Go CLI for picking an AWS profile to log in as, from a list of the
profiles available on your machine. Built for humans at the terminal and for
scripting/agents.

> **Status: early / in design.** The project scaffolding, docs, and release
> pipeline are in place. The concrete command surface is being defined - see
> "What it does" below for the intent and [`.context/HANDOFF.md`](.context/HANDOFF.md)
> for current state. Expect the usage examples here to change as the design
> settles.

## What it does

AWS credentials and named profiles live in the shared config files
(`~/.aws/config` and `~/.aws/credentials`). Switching between them by hand -
remembering names, exporting `AWS_PROFILE`, running `aws sso login --profile X` -
is fiddly. `awsprof` reads the available profiles and lets you pick one to
activate, so "which account am I in?" becomes a single command instead of a
grep through dotfiles.

The exact behavior (interactive picker vs. flag-driven selection, how a profile
is "activated" - `AWS_PROFILE` export, SSO login, credential printing - and
which formats it emits) is the subject of the next design pass.

## Install

> Distribution is wired up but no release has been cut yet; these will work once
> the first `v*` tag is published.

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

## Usage (in design)

The intended shape, subject to the upcoming design pass:

```bash
awsprof                 # interactive: pick a profile from the list
awsprof list            # print the available profiles
awsprof use PROFILE     # select/activate a specific profile
```

Profiles are discovered from the standard AWS locations:

- `~/.aws/config` (sections like `[profile dev]`, plus `[sso-session ...]`)
- `~/.aws/credentials` (sections like `[default]`, `[work]`)
- Overridable via the `AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE`
  environment variables.

`awsprof` reads these files; it does not write your long-lived credentials into
its own config.

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
