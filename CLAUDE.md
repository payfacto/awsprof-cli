@.context/INDEX.md

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What this is

`awsprof` is a Go CLI (binary `awsprof` / `awsprof.exe`) for picking an AWS
profile to log in as, from the list of profiles available on the machine. It
reads the standard AWS shared config files (`~/.aws/config`,
`~/.aws/credentials`), resolves short names via a configured prefix list, logs
in via SSO device authorization only when the cached token is stale, verifies
the resulting identity, and sets `AWS_PROFILE` in the current shell through an
installed shell hook.

The repository was seeded from the `bb` Bitbucket CLI as a convenient Go +
Cobra + release-pipeline template, but it is a **brand-new project with no ties
to `bb`**. Ignore any residual Bitbucket concepts.

## Current state

**Code present - the CLI is fully implemented.** Command surface: bare
interactive picker, `<profile>` positional resolution, `list [--plain]`,
`whoami`, `shell-init <bash|zsh|fish|powershell>`, and a hidden `use <profile>`
alias, all wired in `cmd/`. Supporting packages live under `internal/` (see
"Package layout" below). Tests exist alongside every package. See
[`.context/HANDOFF.md`](.context/HANDOFF.md) for session history and any
remaining backlog items.

## Commands

```bash
go build -o awsprof .      # build the CLI binary (version = "dev")
make build                 # build with a git-derived version stamp
make test                  # go test ./...
go test ./...              # run all tests
```

Standard `go fmt` and `go vet` apply. No linter is configured.

## Versioning

Version is injected via `-ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'`.

- `cmd/root.go` defines `var Version = "dev"` and wires it into
  `rootCmd.Version` (enables `awsprof --version`).
- `Makefile` derives the version from `git describe --tags --always --dirty`.
- `.goreleaser.yaml` injects `v{{.Version}}` at release time; the
  `.github/workflows/release.yml` workflow runs GoReleaser on any pushed tag
  matching `v*`.

When renaming/moving the `Version` variable, update both `Makefile` and
`.goreleaser.yaml` ldflags targets.

## Package layout

`main.go` calls `cmd.Execute()`. Cobra command definitions are thin: flag
parsing, calling into `internal/`, printing output.

- **`cmd/`** - `root.go` (root command: bare picker or positional `<profile>`,
  persistent `--shell` flag), `activate.go` (shared resolve -> login-if-needed
  -> verify -> print-export flow used by root and `use`), `list.go`,
  `whoami.go`, `shell_init.go`, `use.go` (hidden alias).
- **`internal/config`** - loads the optional `~/.awsprof.yaml` (currently just
  the `prefixes` list used for short-name resolution; default `payfacto-`).
- **`internal/profiles`** - discovers and classifies profiles from
  `~/.aws/config` / `~/.aws/credentials` (honoring `AWS_CONFIG_FILE` /
  `AWS_SHARED_CREDENTIALS_FILE`) via `gopkg.in/ini.v1`; resolves a short name to
  a profile (exact match, then each configured prefix in order).
- **`internal/sso`** - the SSO device-authorization login flow
  (`ssooidc` RegisterClient / StartDeviceAuthorization / CreateToken polling)
  and an aws-CLI-compatible token cache at `~/.aws/sso/cache/<sha1>.json`.
- **`internal/identity`** - `sts.GetCallerIdentity` check for a profile, plus
  `NeedsLogin(err)` to classify a credential-resolution failure (stale SSO
  token) vs. an authorization/service error (never retriggers login).
- **`internal/picker`** - the `huh`-based filterable single-select picker;
  renders to stderr so stdout stays reserved for the export line.
- **`internal/shell`** - per-shell (`bash`/`zsh`/`fish`/`powershell`) export
  statement rendering and the `shell-init` hook text.
- Tests live alongside every package (`*_test.go`), using stdlib `testing`.

Do NOT re-introduce `bb`'s Bitbucket client, TUI, or `~/.bbcloud.yaml` config.

## Language and clean code

Go. Always use the `clean-code:go` skill when writing or reviewing Go code, and
prefer modern Go idioms (`use-modern-go`). Write tests for new code and bug
fixes (`superpowers:test-driven-development`).

## Documentation sync - REQUIRED on every command change

Whenever a command or flag is **added, removed, renamed, or its signature
changes**, update all three in the same change:

1. **`README.md`** - the usage/commands section and any narrative examples.
2. **`llms.txt`** - the condensed, agent-facing command reference.
3. **`CLAUDE.md`** (this file) - any command list or architecture note affected.

## Design reference

Design specs and TDD implementation plans live in `.context/specs/` and
`.context/plans/` (indexed by `.context/INDEX.md`). The `.context/` convention
itself is documented in [`.context/claude-context-pattern.md`](.context/claude-context-pattern.md).
