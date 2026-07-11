# awsprof - Handoff

## Goal

`awsprof` is a Go CLI (binary `awsprof` / `awsprof.exe`) for picking an AWS
profile to log in as, from the list of profiles available on the machine. It
reads the standard AWS shared config files (`~/.aws/config`,
`~/.aws/credentials`) and lets the user select/activate one.

## Stack

Go single static binary (`CGO_ENABLED=0`), Cobra for commands. AWS shared-config
parsing library and interactive-picker library are still to be decided (see
[TECHSTACK.md](TECHSTACK.md)). No app database; state is the AWS config files on
disk. Tests use stdlib `testing`. Released via GoReleaser on `v*` tags (GitHub
Actions) to GitHub Releases + a Homebrew tap.

---

## Outstanding backlog

**Design (do first)**

- **Brainstorm the command surface** - what `awsprof` does concretely: bare
  interactive picker vs. `list`/`use` subcommands; how a profile is "activated"
  (export `AWS_PROFILE`, run `aws sso login --profile`, print credentials);
  output formats. Capture the outcome in `.context/specs/`. (Carried since
  2026-07-10.)
- **Decide the AWS-config parsing approach** - `aws-sdk-go-v2/config`, an ini
  parser, or hand-rolled; depends on how much SSO/assume-role resolution awsprof
  does itself. (Carried since 2026-07-10.)
- **Decide the interactive-picker library** - Bubble Tea, huh, promptui, or
  survey. (Carried since 2026-07-10.)

**Build out**

- **Initialize the Go module** (`go mod init github.com/payfacto/awsprof-cli`)
  and scaffold `main.go` + `cmd/root.go` with the version wiring described in
  [CLAUDE.md](../CLAUDE.md). (Carried since 2026-07-10.)
- **Initialize git** as a fresh `awsprof-cli` repo (no history carried from the
  `bb` template). (Carried since 2026-07-10.)

---

## Session history - condensed

_(populated as older sessions are compressed)_

---

## Session - 2026-07-10 (Rebrand template from `bb` to `awsprof`)

### Purpose

The repo is a partial copy of the `bb` Bitbucket CLI, repurposed as a brand-new
`awsprof-cli` project (binary `awsprof`) with no ties to `bb` - just a
convenient Go + Cobra + release-pipeline template. First task: update all docs
to reflect the rename, before brainstorming what the CLI will actually do.

### What was done

- Confirmed there is **no `.git`** and **no Go source** - this is a docs/config
  skeleton only, so "no ties to `bb`" is clean by construction.
- Rewrote the identity/overview docs for `awsprof`: `README.md`, `CLAUDE.md`,
  `llms.txt`, `.context/INDEX.md`, `.context/TECHSTACK.md`, and reset this
  `HANDOFF.md`. Kept them at overview level (functionality is TBD pending the
  brainstorm) rather than inventing a command surface.
- Retargeted the build/release files to `awsprof` and module path
  `github.com/payfacto/awsprof-cli`: `Makefile`, `.goreleaser.yaml` (also fixed
  `license: MIT` -> `Apache-2.0` to match `LICENSE`), `.gitignore` (binary name;
  dropped the `~/.bbcloud.yaml` entry).
- Deleted the `bb`-specific pipeline/deploy enhancement audit under
  `.context/reference/`.
- Left generic/reusable files as-is: `GO-RELEASE-PATTERNS.md`,
  `claude-context-pattern.md`, `.github/workflows/release.yml`, `.claudeignore`,
  `.markdownlint.json`, `LICENSE` (Copyright 2026 PayFacto), `.vscode/settings.json`.

### Decisions

- **Module path `github.com/payfacto/awsprof-cli`** (matches repo name), binary
  `awsprof`. (User-confirmed.)
- **Wipe & reset `bb` history** rather than archive it - truest to "no ties to
  bb". (User-confirmed.)
- **Keep & retarget the release pipeline** (GoReleaser + payfacto Homebrew tap +
  GitHub release-on-`v*`). (User-confirmed.)
- **Docs kept at identity/overview level** - the concrete command surface is the
  output of the next brainstorming pass, not something to fabricate now.

### Running state

- No git repo, no Go source. Docs/config skeleton only.
- No background processes.

### Inferred next steps

- **Brainstorm the command surface** (the reason for this rebrand) - see the
  Design backlog above. Then spec it in `.context/specs/`, then plan, then code.
- Initialize the Go module and git repo when ready to build.

### Suggested skills for next session

- `superpowers:brainstorming` - to define what `awsprof` does.
- `clean-code:go` / `use-modern-go` - once Go code lands.
- `handoff` - to append the next session block here.
