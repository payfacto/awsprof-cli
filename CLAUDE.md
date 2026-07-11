@.context/INDEX.md

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What this is

`awsprof` is a Go CLI (binary `awsprof` / `awsprof.exe`) for picking an AWS
profile to log in as, from the list of profiles available on the machine. It
reads the standard AWS shared config files (`~/.aws/config`,
`~/.aws/credentials`) and lets the user select/activate one.

The repository was seeded from the `bb` Bitbucket CLI as a convenient Go +
Cobra + release-pipeline template, but it is a **brand-new project with no ties
to `bb`**. Ignore any residual Bitbucket concepts.

## Current state

**Docs and scaffolding only - there is no Go source yet.** Present: the
`.context/` knowledge base, `README.md`, `llms.txt`, the release pipeline
(`.goreleaser.yaml`, `.github/workflows/release.yml`, `Makefile`), and editor/
lint config. The concrete command surface and architecture are being defined in
a brainstorming/design pass; capture the outcome in `.context/specs/` before
writing code. See [`.context/HANDOFF.md`](.context/HANDOFF.md) to pick up where
the last session left off.

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

- `cmd/root.go` will define `var Version = "dev"` and wire it into
  `rootCmd.Version` (enables `awsprof --version`).
- `Makefile` derives the version from `git describe --tags --always --dirty`.
- `.goreleaser.yaml` injects `v{{.Version}}` at release time; the
  `.github/workflows/release.yml` workflow runs GoReleaser on any pushed tag
  matching `v*`.

When renaming/moving the `Version` variable, update both `Makefile` and
`.goreleaser.yaml` ldflags targets.

## Intended architecture (to be confirmed in design)

Expect the standard Cobra layout once code lands:

- **`cmd/`** - Cobra command definitions (thin: flag parsing, calling into the
  profile logic, printing output).
- **`internal/`** - profile discovery and AWS shared-config parsing (reading
  `~/.aws/config` / `~/.aws/credentials`, honoring `AWS_CONFIG_FILE` /
  `AWS_SHARED_CREDENTIALS_FILE`), and the selection/activation logic.
- Tests alongside the packages they cover, using stdlib `testing`.

An interactive picker (for `awsprof` with no subcommand) is a likely fit given
the "choose from a list" use case, but the TUI/prompt choice is deferred to the
design pass - do not assume a dependency until it is decided.

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
