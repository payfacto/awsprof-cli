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

---

## Session - 2026-07-11 09:08 (Picker UX fixes for Git Bash)

### What shipped

Fixed the interactive picker on Windows Git Bash (MINGW64/mintty), reported
live by the user. Two commits on top of `a393e37`, TDD'd, all checks clean
(gofmt, `go vet`, 36 tests across 8 packages), verified live in Git Bash.

- **`4e7fea8`** - bounded the select `Height` + footer cleanup + dep note.
- **`648956d`** - stop starting in filter mode + always-visible cancel hint.

All changes are in [internal/picker/picker.go](internal/picker/picker.go) and
its test, plus `go.mod` and [TECHSTACK.md](TECHSTACK.md).

### Root causes / durable facts (huh v1.0.0 internals)

- **Runaway scroll:** a `huh.Select` with static `Options(...)` and no `Height`
  sizes its viewport to the *full* option count. huh only clamps it to the
  terminal on a `tea.WindowSizeMsg`, which Git Bash/mintty does not reliably
  send - so the frame overflowed and the terminal scrolled, pushing the
  selection cursor off-screen. Fix: set an explicit `Height` from
  `term.GetSize(os.Stderr.Fd())` (new pure `selectHeight` helper, unit-tested),
  with a safe fallback when the size is unknown. `github.com/charmbracelet/x/term`
  promoted from indirect to a direct dep.
- **`Select.Filtering(true)` is a misnomer:** it does not *enable* filtering, it
  *starts* the field in filter mode (`filtering=true` + focuses the filter box).
  The `/` filter is available by default regardless. That one call caused the
  auto-open filter box, hid the title (huh renders the filter *instead of* the
  title while filtering), and short-circuited huh's per-position key setup
  (`WithPosition` returns early when `filtering`), which left the prev/next
  bindings enabled -> the duplicate `enter select`/`enter submit` footer.
  Fix: removed `Filtering(true)`.
- **Footer help = field bindings only** (`group.go:391` builds it from
  `field.KeyBinds()`). The form-level `Quit` (ctrl+c) is never shown there, so a
  cancel hint cannot live in the footer. Put it in the field **Description**
  (rendered in every state, incl. filtering) - the Title is unusable for this
  because it is swapped out for the filter box while filtering.
- **Esc cannot be a cancel key:** it is reserved for filter set/clear, and the
  form matches `Quit` *before* the field sees the key (`form.go:558`), so binding
  Esc to quit would abort mid-filter and break filtering. **Ctrl+C is the cancel
  key** and already aborts cleanly ([cmd/root.go](../cmd/root.go) handles
  `picker.ErrAborted` -> return nil, no export, no SSO).
- `pickerKeyMap()` disables the meaningless prev/next bindings on the
  single-field picker and relabels Submit's Enter as "select".

Net UX now: opens on the list (no auto-filter), title + `ctrl+c to cancel`
visible, `/` filters, footer reads `up . down . / filter . enter select`.

### Running state

- On `main` @ `648956d`, clean tree. Nothing pushed (still no remote).
- Rebuilt `./awsprof` in the repo root (git-ignored). No background processes.

### Inferred next steps

- Backlog unchanged and now the headline item: **cut v0.1.0** on user go-ahead
  (add remote, push `main`, tag `v0.1.0`; `HOMEBREW_TAP_TOKEN` repo secret for
  the Homebrew step).
- Prior minor follow-ups still stand: `shell.ExportLine` escaping, friendlier
  SSO-region error, small test-coverage gaps.
- Live picker render is now exercised on Git Bash (nav, filter, cancel all
  confirmed); native bash/zsh/fish *hooks* still only unit-tested.

### Suggested skills for next session

- `go-release` runbook when cutting v0.1.0.
- `clean-code:go` for further Go work.

---

## Session - 2026-07-11 10:19 (Environment coloring feature)

### What shipped

Brainstormed and implemented environment-based coloring of profile names.
Two commits on top of `648956d`: `7b7c817` (design spec) and `a54bd56` (the
feature). gofmt/`go vet` clean, **61 tests pass across 9 packages**, binary
builds. Delivered TDD (envcolor package test-first) with live verification.

- **New `internal/envcolor`** (pure): `Detect(name) (Env, idx)` matches the env
  keyword against whole hyphen-segments (case-insensitive, first match wins);
  `Render(name, *lipgloss.Renderer)` colors **only** the matched segment
  ("Style D") and returns unrecognized names unchanged.
- **Wired into** `list` ([cmd/list.go](../cmd/list.go)), the picker
  ([internal/picker/picker.go](../internal/picker/picker.go), colored at the
  huh-option boundary), `whoami` ([cmd/whoami.go](../cmd/whoami.go)), and the
  unknown-profile fallback list (`printProfiles` in
  [cmd/activate.go](../cmd/activate.go)).
- **`whoami` reframe:** when `AWS_PROFILE` is unset it now prints the effective
  `default` with a dim `(unset)` hint (extracted testable `whoamiLine`), instead
  of the old `(unset -> default)` literal.
- **lipgloss** promoted from indirect to a direct dependency (no `go.sum`
  change; already present). Docs synced: the
  [spec](specs/2026-07-11-env-color-design.md) and INDEX, README, llms.txt,
  CLAUDE.md, TECHSTACK.md.

### Key decisions / facts (durable)

- **Palette (hardcoded, not configurable):** prod=**bold** red, staging=orange,
  uat=purple, qa=yellow, dev=green, sandbox=blue; no-env names stay plain.
  Aliases: prod|production, staging|stage|stg, uat, qa, dev|development,
  sandbox|test|sbx. `test` maps to **sandbox** (per user), not dev.
- **Style D** (color the env segment only, name otherwise normal) was chosen
  over whole-name / badge / dimmed-remainder variants. Selected via live sideshow
  mockups after terminal ANSI and AskUserQuestion previews failed to render color
  for the user - **sideshow is the way to show color/visual options to this
  user** (session "Profile environment coloring").
- **Color safety is delegated to lipgloss:** a `*lipgloss.Renderer` bound to the
  target stream (stdout for list/whoami, stderr for picker/fallback) auto-detects
  color depth and honors `NO_COLOR`, non-TTY, and Windows VT. `list --plain`
  bypasses coloring entirely. All four paths verified byte-clean on the real
  binary; lipgloss also suppresses the faint attribute when color is off (so no
  guard needed).
- **Color-depth behavior:** at 256-color (user's `xterm-256color`) all six envs
  resolve to distinct codes (prod 203, staging 215, uat 176, qa 221, dev 71,
  sandbox 75). Only pure **16-color** terminals collapse orange->bright-red,
  colliding staging with prod - mitigated by prod's bold. Documented as a v1
  non-goal (adaptive/AdaptiveColor deferred).
- Scope intentionally excludes: YAML-configurable colors, `--color` flag,
  light-vs-dark adaptive hues (all v1 non-goals in the spec).

### Running state

- On `main` @ `a54bd56`, nothing pushed (still no git remote).
- `.context/HANDOFF.md` was carrying an uncommitted 09:08 block from a prior
  session; this block is appended on top and both are being committed together
  now.
- `./awsprof` rebuilt in repo root (git-ignored). No background processes.

### Verified vs. residual

- Verified live: `list` (user screenshot - Style D correct), the **picker**
  (user screenshot - env colors intact on all rows, huh not clobbering),
  `whoami` unset framing + forced-color + piped-clean (via built binary),
  and the NO_COLOR / non-TTY / `--plain` off-paths.
- **Residual (low):** the user's picker screenshot had the cursor on an
  *uncolored* row (`default`), so a *highlighted colored* row is not explicitly
  confirmed. Non-selected colored rows are fine. If huh's selected-row style ever
  visibly clobbers the env color, fallback is a huh theme tweak (don't override
  the option foreground) or a colored leading marker.

### Inferred next steps

- Headline backlog item is unchanged: **cut v0.1.0** on user go-ahead (add
  remote, push `main`, tag `v0.1.0`; `HOMEBREW_TAP_TOKEN` repo secret needed).
- Prior minor follow-ups still stand: `shell.ExportLine` escaping, friendlier
  SSO-region error, small test-coverage gaps.
- Optional coloring follow-ups if ever wanted: confirm the highlighted colored
  picker row; add YAML-configurable env colors; `AdaptiveColor` for light
  terminals.

### Suggested skills for next session

- `go-release` runbook ([GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md)) when
  cutting v0.1.0.
- `clean-code:go` for further Go work; `sideshow` to show the user any
  color/visual options.

---

## Session - 2026-07-11 14:16 (Shipped v0.1.0 + v0.1.1: release pipeline, go-install fix, review + clean-code, CI/README)

Long session. Cut the first two releases and hardened the code. All work is on
`main`, pushed to Bitbucket, mirrored to GitHub. Commits since the previous
block: `41bac9d`..`226aae9` (see `git log 946ea15..HEAD`).

### Release pipeline is LIVE (Payfacto-standard, proven end-to-end)

- **Bitbucket is source of truth**, GitHub hosts CI + releases:
  - `origin` = `https://jmadore@bitbucket.org/payfactopay/awsprof-cli.git`
  - `github` = `https://github.com/payfacto/awsprof-cli.git` (repo is now PUBLIC)
  - Local `gh` is authed as **jmadore-payfacto** with ADMIN on the payfacto repos.
- **Flow:** work on `main` -> `git push origin main` (Bitbucket) -> the
  [bitbucket-pipelines.yml](bitbucket-pipelines.yml) mirror force-pushes main+tags
  to GitHub -> `.github/workflows/release.yml` runs GoReleaser on any `v*` tag.
- **To cut a release:** `git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z`.
  That is the ONLY release action needed. Newest tag = **v0.1.1**.
- **One-time manual setup is DONE (won't repeat):** Bitbucket Pipelines enabled +
  SSH keypair + `github.com` known host; a write-enabled GitHub deploy key
  (`bitbucket-mirror`); and the `HOMEBREW_TAP_TOKEN` Actions secret (a PAT with
  Contents:write on both `payfacto/awsprof-cli` and `payfacto/homebrew-tap` -
  GoReleaser uses it as GITHUB_TOKEN for the release AND the tap bump). Do not
  record the token value.
- **Homebrew:** `payfacto/homebrew-tap` (public), `Formula/awsprof.rb` bumped by
  GoReleaser each release. `brew install payfacto/tap/awsprof`.
- `.goreleaser.yaml` pins `release.github.owner/name` to payfacto/awsprof-cli so
  the release + formula URLs are correct regardless of which remote GoReleaser
  infers locally (origin is Bitbucket, which produced wrong URLs in snapshots).

### go install fix (`98710fa`) - the entrypoint moved

- `main.go` relocated to **`cmd/awsprof/main.go`** so
  `go install github.com/payfacto/awsprof-cli/cmd/awsprof@latest` yields a binary
  named **`awsprof`** (Go names it after the package dir; the module basename was
  giving `awsprof-cli`). Build paths updated: `.goreleaser` gets
  `main: ./cmd/awsprof` and `binary: awsprof`; Makefile builds `./cmd/awsprof`;
  plus README/llms/CLAUDE/TECHSTACK.
- **Version fallback** in `cmd/root.go`: `effectiveVersion` + `mainModuleVersion`
  read `runtime/debug` build info when ldflags aren't applied, so `go install` /
  `go build` report the real module version instead of `dev` (release binaries
  still use the ldflags value). Unit-tested (`cmd/version_test.go`).
- **`.gitignore` gotcha (fixed in same commit):** a bare `awsprof` line was
  ignoring the whole `cmd/awsprof/` dir; anchored to `/awsprof` (root binary only).
- Verified: `go install .../cmd/awsprof@v0.1.1` -> `awsprof` reporting `v0.1.1`.

### Full-repo review + hardening (`f4db1c9`, via code-review-expert)

- **Security (was P2):** `internal/shell` `ExportLine` now quotes the profile
  name per shell (POSIX `'\''`, PowerShell single-quote doubling, fish backslash)
  so a crafted `~/.aws/config` profile name cannot break out and execute when the
  hook eval's the line. Also fixes the PowerShell `$`/`$(...)` interpolation bug
  (was `%q`). Covered by injection test cases.
- `internal/sso/cache.go`: expanded the SHA1 comment (interop filename derivation,
  not crypto - explicit SAST false-positive note per user request); removed
  production-dead `ReadToken` / `Token.Valid` / `expirySkew`; **`WriteToken` is
  now atomic** (temp file + `os.Rename`).
- `cmd/activate.go`: removed the `resolveTargetForTest` seam (tests call
  `resolveTarget` directly); added a friendly "SSO profile has no region" message
  via a tested `regionHint` helper + an `ssoLogin` guard.

### clean-code:go pass (`da5f492`)

- `renderList` split into `renderPlainList` + `renderList` (drops the bool flag
  arg F3 + 4th param F1); `profiles.classify` returns `(Type, *SSOConfig)` instead
  of mutating a `*Profile` (F2/F1, now a pure query); `login.go` named the magic
  5s poll durations (G25). 79 tests pass incl. `-race` (CGO/gcc present here).
- **Left as conscious trade-offs (documented in code):** `sso.Login` 5 params
  (ctx exempt + 3 DI seams; struct-bundling is net-neutral), and
  `identity.NeedsLogin` error-string matching (the AWS SDK exposes no clean typed
  signal at that boundary; locked by regression + guard tests).

### CI + README polish (`31a693d`, `226aae9`)

- Bumped `actions/checkout@v4->@v7` and `actions/setup-go@v5->@v6` (Node 24;
  clears the Node 20 deprecation warning seen on the v0.1.1 run). Takes effect on
  the next tagged release.
- Added `assets/banner.png` (top of README) and `assets/example.png` (after the
  "What it does" section, shows the color-coded picker + whoami). `assets/` now
  tracked. The two `<p align="center">` blocks trip MD033 (inline HTML) - expected
  and fine for centering images.

### Running state

- On `main` @ `226aae9`, clean except an **untracked `assets/icon.png`** the user
  added (left uncommitted intentionally - not part of any task).
- No background processes. `goreleaser` v2.17.0 installed at `$(go env GOPATH)/bin`.
- **User's shell env:** `~/go/bin` is on PATH; `~/.bashrc` line 19 runs the hook
  `eval "$(awsprof shell-init bash)"`. `~/go/bin/awsprof.exe` = the v0.1.0 release
  binary placed manually (works); `~/go/bin/awsprof-cli.exe` = leftover from their
  earlier `go install` (safe to `rm`). They can move to v0.1.1 via
  `go install github.com/payfacto/awsprof-cli/cmd/awsprof@latest`. The old
  `awsp`/`awswho` shell functions still live in `~/.bashrc` (superseded, harmless).

### Inferred next steps / backlog (all non-urgent)

- **Homebrew cask migration:** GoReleaser deprecated `brews` (still works, warns);
  the replacement `homebrew_casks` is macOS-only + hits Gatekeeper on unsigned
  binaries. Defer until a macOS code-signing/notarization story - do them together.
- **`identity.Check` test seam:** non-blocking architecture suggestion from the
  review (introduce an STS-client interface so the login-retry path is unit
  testable) - not done.
- Optional coloring follow-ups: YAML-configurable env colors; `AdaptiveColor` for
  light terminals; confirm the huh *selected* (highlighted) colored picker row
  (only non-selected rows were screenshot-verified).

### Suggested skills for next session

- `go-release` runbook ([GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md)) for the
  next `vX.Y.Z`.
- `clean-code:go` / `code-review-expert` for further Go work.
