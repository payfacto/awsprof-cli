# awsprof - Context Index

Navigation hub for the curated `.context/` knowledge base. `CLAUDE.md` `@`-imports
this file every session; the entries below are read on demand. `@`-prefixed files
are also auto-imported by `CLAUDE.md`.

## Root Files

- Session handoff (outstanding backlog, condensed session history, decisions,
  open questions) now lives out-of-repo at `~/.claude/handoffs/` (see the
  `handoff` skill), not in this directory. Use `/handoff resume` to pick up
  where the last session left off.
- [@TECHSTACK.md](TECHSTACK.md) - Tech stack reference. Split into the confirmed
  base (Go, Cobra, GoReleaser) and the parts still to be decided in design.
- [GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md) - Release runbook: GoReleaser
  tag-and-push flow, Homebrew tap automation, troubleshooting. Reusable template.
- [claude-context-pattern.md](claude-context-pattern.md) - How this `.context/`
  knowledge convention works (language-agnostic meta reference).

## Subfolders

### `specs/`

- [specs/2026-07-10-awsprof-design.md](specs/2026-07-10-awsprof-design.md) -
  The awsprof design: shell-hook activation, native AWS SDK (SSO device-login
  flow, no `aws` CLI), `awsprof <profile>` primary + picker + list/whoami,
  exact-then-prefix name resolution, cross-shell hooks. Approved 2026-07-10.
- [specs/2026-07-11-env-color-design.md](specs/2026-07-11-env-color-design.md) -
  Environment coloring: detect the env segment in a profile name
  (`payfacto-<app>-<env>-<role>`) and color just that segment ("Style D") in
  `list`, the picker, and `whoami`. Hardcoded env->color map (prod=bold red,
  staging=orange, uat=purple, qa=yellow, dev=green, sandbox=blue) via lipgloss;
  respects NO_COLOR / non-TTY / `--plain`. Approved 2026-07-11.

### `plans/`

- [plans/2026-07-10-awsprof-implementation.md](plans/2026-07-10-awsprof-implementation.md) -
  TDD plan (14 tasks) implementing the awsprof design: module bootstrap,
  config/profiles/shell pure packages, `list` milestone, SSO cache + device
  login, identity check, huh picker, activation wiring, whoami, shell-init,
  docs sync. Read the design spec first.

### `reference/`

- [reference/awsp-awswho-snippet.txt](reference/awsp-awswho-snippet.txt) - The
  shell-function prototype awsprof is based on: `awsp <profile>` (switch, short
  names resolve a `payfacto-` prefix, SSO login only when the token is
  missing/expired, exports `AWS_PROFILE`, prints Account+ARN) and `awswho`
  (print current identity). Prior art / behavior reference for the design.

### `tools/`

- _(empty - add diagnostic helpers, gated on env vars; annotate each with how
  to invoke.)_
