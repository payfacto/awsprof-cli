# awsprof - Design

- Date: 2026-07-10
- Status: Approved (brainstorm complete; ready for implementation planning)
- Author: brainstormed with Claude Code

## Problem

Switching between AWS named profiles by hand is fiddly: you have to remember
profile names, `export AWS_PROFILE`, and run `aws sso login` when the SSO token
has expired. A shell function (`awsp`, see references) solves this on bash but
does not work in PowerShell, cannot be shared as a binary, and hardcodes
site-specific assumptions.

`awsprof` is a self-contained Go CLI that lets you pick an AWS profile to log in
as, from the list of profiles available on the machine, and activates it in your
current shell. It performs an SSO login only when the cached token is stale, then
confirms who you are.

## Goals

- Pick a profile from an interactive list (the core ask), or name it directly.
- Activate the chosen profile in the user's current shell (`AWS_PROFILE`).
- Log in via AWS IAM Identity Center (SSO) only when the cached token is
  missing or expired.
- Confirm identity after activation (Account + ARN).
- Work cross-shell, including Windows PowerShell.
- Ship as a single binary with no dependency on the `aws` CLI.

## Non-goals

- Not a general AWS CLI. It does not wrap arbitrary AWS operations.
- Does not manage/edit `~/.aws/config` or credentials (read-only over them).
- Does not store long-lived credentials in its own config.
- Does not export `AWS_REGION` (setting `AWS_PROFILE` lets AWS tools read the
  profile's own `region`; exporting a region can wrongly override it).
- Does not implement `aws sso login` for non-SSO profile types (static keys,
  assume-role, credential_process are set and verified, never SSO-logged-in).

## Core decisions

| Decision | Choice | Why |
| --- | --- | --- |
| Activation model | Shell hook + eval | Sets `AWS_PROFILE` in the current shell; works cross-shell |
| AWS backend | Native `aws-sdk-go-v2` | Self-contained binary; no `aws` CLI dependency |
| Profile scope | SSO-first, tolerate the rest | Broadest; only SSO profiles get the login flow |
| Name resolution | Exact, then configurable prefix (default `payfacto-`) | Generalizes the `awsp` behavior without hardcoding |
| Primary invocation | `awsprof <profile>` (positional) | Matches the `awsp <profile>` habit; drops a `use` verb |
| Interactive picker | `charmbracelet/huh` filterable select | Lightest path to a nice, filterable picker; Windows-friendly |

## Command surface

```text
awsprof                       # interactive picker -> login if needed -> activate
awsprof <profile>             # resolve (exact/prefix) -> login if needed -> activate
awsprof list [--plain]        # list profiles; default marks the active one; --plain = bare names
awsprof whoami                # current identity (Account, ARN, active profile), no switch
awsprof shell-init <shell>    # emit the shell hook (bash|zsh|fish|powershell)
awsprof use <profile>         # HIDDEN alias of `awsprof <profile>` (scripts / name collisions)
awsprof --version / --help
```

- Root is `awsprof [profile]` with `Args: MaximumNArgs(1)`. A token matching a
  registered subcommand runs that subcommand; any other token falls through to
  root's `RunE` as the profile name; no args launches the picker.
- Reserved names: a profile named exactly `list`, `whoami`, `shell-init`,
  `use`, `completion`, or `help` is shadowed by the subcommand. The picker always
  reaches it, and the hidden `use <profile>` gives an unambiguous path.
- Activation output contract: the `export` line goes to **stdout** (for the
  shell wrapper to eval); all human output (identity, "logging in...", errors,
  the picker UI) goes to **stderr**. `list --plain` prints bare names to stdout
  for scripting/completion.

## Configuration - `~/.awsprof.yaml`

Small and optional; sensible defaults when absent.

```yaml
prefixes: ["payfacto-"]   # tried in order for short-name resolution
```

## Architecture

Package layout:

```text
main.go                      -> cmd.Execute()
cmd/                         Cobra wiring (thin): root (positional profile + picker),
                             list, whoami, shell_init, use (hidden alias)
internal/config              load ~/.awsprof.yaml (prefixes) + defaults
internal/profiles            parse ~/.aws/config + credentials; enumerate names;
                             classify type (sso-session / legacy-sso / static /
                             assume-role / credential_process); resolve name (exact->prefix)
internal/sso                 SSO OIDC device-auth flow + token cache (~/.aws/sso/cache)
internal/identity            STS GetCallerIdentity; token-validity check
internal/shell               shell detection + hook generation + export-line syntax per shell
internal/picker              huh filterable select
internal/version             version var (injected via ldflags)
```

Boundaries: `config`, `profiles`, and `shell` are pure (files and strings) and
directly unit-testable. `sso` and `identity` sit behind small interfaces so the
network is mockable.

## Activation data flow (`awsprof <name>` and the picker)

1. Load config (prefixes); discover profiles from `~/.aws`.
2. Choose the target: resolve the typed name (exact, then prefix), or let the
   user select in the huh picker.
3. Build an SDK config for that profile and call `sts.GetCallerIdentity`:
   - Succeeds: credentials already valid; nothing to log in.
   - Fails and the profile is SSO: run the device-login flow (below), then retry.
   - Fails and the profile is non-SSO: surface the error (cannot auto-login).
4. Emit `export AWS_PROFILE=<name>` in shell-correct syntax to stdout.
5. Print identity (Account, ARN, profile) to stderr.

An unknown name (no exact match and no configured prefix yields an existing
profile) prints the profile list to stderr and exits non-zero without emitting
an export. Prefixes are tried in configured order; the first that resolves to an
existing profile wins, so resolution is deterministic.

## SSO device-login flow (`internal/sso`)

Runs only when the cached token is missing or expired.

1. Resolve the profile's `sso_session` (or legacy inline `sso_start_url` /
   `sso_region`).
2. Check `~/.aws/sso/cache/<key>.json`; if the access token is present and
   unexpired, reuse it.
3. Otherwise: `RegisterClient` -> `StartDeviceAuthorization` -> open the browser
   (`pkg/browser`) to the verification URL (also print URL + user code to stderr
   as a fallback) -> poll `CreateToken` at the returned interval, handling
   `AuthorizationPendingException` and `SlowDownException`, until authorized or a
   timeout.
4. Write the token + expiry to `~/.aws/sso/cache/<key>.json` in the
   aws-CLI-compatible format and location, so the `aws` CLI and other SDKs reuse
   the same session. With a valid token cached, the SDK's SSO provider resolves
   role credentials itself.

## Cross-shell activation

`shell-init <shell>` emits the wrapper function plus the correct export syntax;
the binary's `--shell` flag (injected by the wrapper) selects the syntax.

The generated wrapper only eval's activating calls, so data commands keep their
stdout:

```bash
awsprof() {
  case "$1" in
    list|whoami|shell-init|completion|help|-h|--help|-v|--version)
      command awsprof "$@" ;;                       # passthrough: stdout flows normally
    *)                                              # bare (picker) or a profile name
      local out; out="$(command awsprof --shell bash "$@")" || return
      [ -n "$out" ] && eval "$out" ;;               # only the export line is eval'd
  esac
}
```

| Shell | rc line | export syntax |
| --- | --- | --- |
| bash | `eval "$(awsprof shell-init bash)"` in `~/.bashrc` | `export AWS_PROFILE=x` |
| zsh | `eval "$(awsprof shell-init zsh)"` in `~/.zshrc` | `export AWS_PROFILE=x` |
| fish | `awsprof shell-init fish \| source` in `config.fish` | `set -gx AWS_PROFILE x` |
| powershell | `awsprof shell-init powershell \| Out-String \| Invoke-Expression` in `$PROFILE` | `$env:AWS_PROFILE = "x"` |

## Error handling and exit codes

- `0` on success.
- Non-zero on: unknown/ambiguous profile (list printed to stderr), login timeout
  or failure, cancelled picker (Esc/Ctrl-C), not-authenticated (`whoami`).
- No `export` is ever emitted on failure, so a failed command never half-changes
  the shell.
- stdout carries only eval-able export lines or `--plain` data; every diagnostic
  goes to stderr.

## Dependencies

- `github.com/spf13/cobra` - CLI framework.
- `github.com/charmbracelet/huh` - interactive picker. Its form output
  defaults to stderr in interactive mode (stdout only in accessible mode), so
  the picker UI does not break the stdout export contract. huh v2 uses import
  path `charm.land/huh/v2` with the same `NewSelect` API; if `go get` pulls v2,
  update the import path only.
- `github.com/aws/aws-sdk-go-v2` with `config`, `credentials`, `service/sts`,
  `service/sso`, `service/ssooidc` - profile loading, identity, SSO.
- `github.com/pkg/browser` - open the SSO verification URL cross-platform.
- `gopkg.in/yaml.v3` - parse `~/.awsprof.yaml`.
- `gopkg.in/ini.v1` - enumerate profile names (the SDK has no public
  list-all-profiles API).

Windows is a first-class target: PowerShell hook, `os.UserHomeDir()`, and
`pkg/browser` are all cross-platform.

## Testing strategy

- Pure, table-driven (stdlib `testing`): name resolution (exact -> prefix,
  ambiguity, unknown), profile parsing/classification (sample `~/.aws` in temp
  dirs), SSO cache-key hashing + token expiry, shell-hook generation and
  export-line formatting per shell.
- `sso` and `identity` behind interfaces: unit-test request-building, cache
  read/write, and poll/backoff classification.
- The live SSO device flow is a documented manual smoke test (needs real SSO
  access); it is not exercised in automated tests.
- No third-party test framework.

## Alternatives considered

- Activation via subshell spawn or manual `eval "$(awsprof use x)"`: rejected in
  favor of a shell hook, which persists the profile in the current shell without
  a nested shell and without the user remembering to eval.
- Shelling out to the `aws` CLI (or a hybrid): rejected in favor of the native
  SDK for a self-contained binary. Cost: reimplementing the SSO device flow.
- `bubbles`/Bubble Tea or `promptui` for the picker: rejected in favor of `huh`
  for the lightest path to a filterable select. Revisit if per-row account/region
  badges become worth the extra code.
- Fuzzy/substring or exact-only name resolution: rejected in favor of
  exact-then-prefix for predictability in scripts (the picker still filters).

## Open questions / to verify during implementation

- Confirm the SSO cache-key hashing (SHA1 of the `sso_session` name for modern
  config vs. SHA1 of the start URL for legacy inline config) against a real
  `aws` CLI cache before relying on interop.
- Confirm the exact aws-CLI-compatible token cache JSON schema fields
  (accessToken, expiresAt, region, startUrl, clientId, clientSecret, etc.).
- Decide the SSO login poll timeout default (e.g. the device-auth
  `expiresIn`, capped).
- Resolved: huh form output defaults to stderr in interactive mode, so the
  stdout export contract holds. The picker will still set `WithOutput(os.Stderr)`
  explicitly as belt-and-braces.

## References

- `.context/reference/awsp-awswho-snippet.txt` - the `awsp`/`awswho` shell
  functions this tool is based on.
- `.context/TECHSTACK.md`, `CLAUDE.md` - project conventions and versioning.
- AWS SDK for Go v2: `service/ssooidc`, `service/sso`, `config` (shared config
  and SSO credential provider).
