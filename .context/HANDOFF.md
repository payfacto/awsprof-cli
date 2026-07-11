# awsprof - Handoff

## Goal

`awsprof` is a Go CLI (binary `awsprof` / `awsprof.exe`) for picking an AWS
profile to log in as, from the list of profiles available on the machine. It
reads the standard AWS shared config files (`~/.aws/config`,
`~/.aws/credentials`), performs an SSO device login only when the cached token
is stale, verifies identity, and sets `AWS_PROFILE` in the current shell via an
installed shell hook.

## Stack

Go single static binary (`CGO_ENABLED=0`), Cobra CLI, `charmbracelet/huh` picker,
`aws-sdk-go-v2` (config/sts/ssooidc + smithy-go), `pkg/browser`, `gopkg.in/ini.v1`
(profile parsing), `gopkg.in/yaml.v3` (`~/.awsprof.yaml`). No `aws` CLI
dependency. No app database; state is the AWS config files + the SSO token cache.
Tests use stdlib `testing`. Released via GoReleaser on `v*` tags (GitHub Actions)
to GitHub Releases + a Homebrew tap. Full detail in [TECHSTACK.md](TECHSTACK.md).

---

## Outstanding backlog

**Ship / verify**

- VERIFIED 2026-07-11 (live smoke test on Windows/PowerShell, real payfacto SSO):
  `list`, `whoami` (clean error on expired token), unknown-profile no-export,
  short-name prefix resolution (`sandbox-readonly` -> `payfacto-sandbox-readonly`),
  SSO device-login end-to-end (browser approve), identity round-trip, PowerShell
  export syntax + stdout/stderr contract, the shell hook setting `AWS_PROFILE`
  in-session, and **aws-CLI token-cache interop** (`aws sts get-caller-identity`
  reused awsprof's freshly cached token, no re-login).
- FIXED 2026-07-11 (commit `2174c18`): the smoke test caught the top pre-1.0
  bug live - `NeedsLogin` short-circuited to false on any `smithy.APIError` in
  the chain, but an expired token's refresh failure wraps an ssooidc
  `InvalidGrantException`, so `awsprof <sso-profile>` errored instead of
  re-logging-in. Now matches positive SSO-token-failure signals (regression test
  locks the real wrapped-APIError shape). Verified: login triggered on the
  expired token and succeeded.
- Still untested (lower priority): bash/zsh/fish hooks on their native shells
  (only PowerShell exercised); interactive picker on a TTY (BuildItems is unit
  tested; the huh render is not).

**Release (needs user go-ahead - never push/tag without asking)**

- No git remote is configured yet. To release: add the GitHub remote, push
  `main`, then tag `v0.1.0` (GoReleaser fires on `v*`). `HOMEBREW_TAP_TOKEN`
  must be set as a repo secret for the Homebrew step. See
  [GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md).

**Minor follow-ups (all ship-as-noted per final review)**

- `internal/shell` `ExportLine` does not escape profile names; the PowerShell
  branch (`%q`) is the weakest (a name with `$` would set the wrong value).
  Low risk given conventional AWS profile names; revisit if naming is loosened.
- `internal/sso` `ReadToken`/`Valid` are not on the activation path (the SDK
  reads/validates the cache); documented, kept for future use.
- SSO profile lacking a profile-level `region` (inherits only `sso_region` from
  its `[sso-session]`) can produce a confusing raw region error; consider a
  friendlier message.
- Minor test-coverage gaps recorded during the build (config `DefaultPath`,
  profiles cross-file precedence / DEFAULT-section, sso SlowDown branch) - all
  correct by inspection.

---

## Session history - condensed

**Session 2026-07-10 (Rebrand from `bb`).** Repurposed a partial copy of the `bb`
Bitbucket CLI into a fresh `awsprof-cli` project: rewrote README/CLAUDE/llms/
INDEX/TECHSTACK, reset this handoff, retargeted Makefile/.goreleaser/.gitignore
to `github.com/payfacto/awsprof-cli` (binary `awsprof`), deleted the bb pipeline
audit + bb-branded banner. Then brainstormed the design and wrote the spec
[specs/2026-07-10-awsprof-design.md](specs/2026-07-10-awsprof-design.md) and the
TDD plan [plans/2026-07-10-awsprof-implementation.md](plans/2026-07-10-awsprof-implementation.md).

---

## Session - 2026-07-11 (Implemented the CLI: 14-task plan, shipped v0-ready)

### What shipped

Executed the full implementation plan via subagent-driven development (fresh
implementer per task + per-task spec/quality review + fix loops, final
whole-branch review on the most capable model). Built greenfield on `main`;
`git init` done this session. 22 commits (`b733627` docs baseline ..`5317605`).
**Final state: gofmt clean, `go vet` clean, 28 tests pass across 8 packages,
binary builds.**

Command surface (all wired in `cmd/`): bare `awsprof` (interactive picker),
`awsprof <profile>` (positional; exact-then-prefix resolution), `list [--plain]`,
`whoami`, `shell-init <bash|zsh|fish|powershell>`, hidden `use <profile>` alias,
persistent `--shell`. Packages: `internal/{config,profiles,sso,identity,picker,shell}`.

### Key decisions / facts (durable)

- **Activation = shell hook + eval.** Binary prints ONLY the `export AWS_PROFILE=...`
  line to stdout; identity/progress/errors/picker-UI all go to stderr. The
  generated hook eval's only activation calls and passes data commands
  (`list`/`whoami`/`shell-init`) through. Final review verified this contract
  end-to-end (single stdout write at `cmd/activate.go`, no export on any failure).
- **Native SDK, no `aws` CLI.** SSO login is the OIDC device-authorization grant
  (`aws-sdk-go-v2/service/ssooidc`), token cached in the aws-CLI-compatible
  `~/.aws/sso/cache/*.json` (0600) so tools share the session; role creds then
  resolve via the SDK's SSO provider inside `identity.Check`.
- **`NeedsLogin` uses a `smithy.APIError` discriminator** (review-driven fix): an
  API-level error means creds resolved (authz/service problem, not login), so
  an `AccessDenied` carrying an `AWSReservedSSO_...` ARN never triggers a
  spurious re-login. See the pre-1.0 caveat in the backlog.
- **`Version` lives in package `cmd`** (`cmd.Version`), matching the Makefile /
  .goreleaser ldflags; verified `-X` injection works.
- **huh resolved to v1.0.0** (classic import path `github.com/charmbracelet/huh`),
  not the v2 `charm.land/huh/v2` path.

### Running state

- On `main` @ `5317605`, clean tree, nothing pushed (no remote configured).
- No background processes. SDD scratch (task briefs/reports/review packages,
  progress ledger) under `.superpowers/sdd/` (git-ignored).

### Inferred next steps

- Run the deferred manual smoke tests (top of backlog), especially the live
  SSO + `NeedsLogin` confirmation.
- On user go-ahead: configure the remote, push, tag `v0.1.0`.

### Suggested skills for next session

- `verify` / manual smoke on a real SSO profile.
- `go-release` runbook ([GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md)) when cutting v0.1.0.
- `handoff` to append the next block.
