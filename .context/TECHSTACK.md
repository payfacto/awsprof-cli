# TECHSTACK.md

Technology stack for `awsprof` - a Go CLI for picking an AWS profile to log in
as. The project was seeded from a Go + Cobra CLI template (`bb`) and reuses its
build/release machinery. All entries below are derived from `go.mod`, the
`internal/`/`cmd/` source tree, `Makefile`, `.goreleaser.yaml`, and
`.github/workflows/release.yml`; `go.mod` is the authoritative source for
versions.

## Language and Runtime

- Go `1.26.4` (per `go.mod`).
- Single-binary CLI; entrypoint `main.go` -> `cmd.Execute()`.
- Cross-compiled for `linux`, `darwin`, `windows` x `amd64`, `arm64` with
  `CGO_ENABLED=0` (per `.goreleaser.yaml`).

## Core Frameworks and Libraries

Direct dependencies (`go.mod` `require` block):

- CLI framework: `github.com/spf13/cobra` v1.10.2.
- Interactive picker: `github.com/charmbracelet/huh` v1.0.0 - filterable
  single-select for the bare-command picker (`internal/picker`), rendered to
  stderr so stdout stays reserved for the export line.
- AWS SDK: `github.com/aws/aws-sdk-go-v2` v1.42.1, with
  `github.com/aws/aws-sdk-go-v2/config` v1.32.29 (shared-config/credential
  loading in `internal/identity`), `github.com/aws/aws-sdk-go-v2/service/ssooidc`
  v1.37.0 (SSO device-authorization flow in `internal/sso`), and
  `github.com/aws/aws-sdk-go-v2/service/sts` v1.44.0 (`GetCallerIdentity` in
  `internal/identity`). `github.com/aws/aws-sdk-go-v2/service/sso` is pulled in
  transitively (via `aws-sdk-go-v2/config`), not imported directly.
- `github.com/aws/smithy-go` v1.27.3 - used to distinguish an AWS API error
  (`smithy.APIError`, meaning credentials resolved fine) from a
  credential-resolution failure in `identity.NeedsLogin`.
- `github.com/pkg/browser` - opens the SSO device-authorization URL in the
  user's browser during login (`cmd.ssoLogin`).
- AWS shared-config parsing: `gopkg.in/ini.v1` v1.67.3 - reads
  `~/.aws/config` / `~/.aws/credentials` and classifies profiles
  (`internal/profiles`).
- Settings file parsing: `gopkg.in/yaml.v3` v3.0.1 - reads the optional
  `~/.awsprof.yaml` (`internal/config`).

Activation mechanism: `awsprof` never shells out to the `aws` CLI.
It resolves credentials and calls `sts.GetCallerIdentity` directly through
`aws-sdk-go-v2`; for a stale SSO token it runs the OIDC device-authorization
grant itself, then prints an `export AWS_PROFILE=...` (or shell-equivalent)
line to stdout for the installed shell hook to `eval`.

## Data and Persistence

- No application database.
- Reads AWS shared config/credentials from `~/.aws/` (overridable via
  `AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE`); never writes to those
  files.
- State awsprof does keep: an aws-CLI-compatible SSO token cache file at
  `~/.aws/sso/cache/<sha1-of-session-or-start-url>.json` (`internal/sso`,
  mode `0600`), and the optional `~/.awsprof.yaml` settings file (currently
  just the `prefixes` list for short-name resolution; default `payfacto-`).

## Security and Secrets

- Reads existing AWS credentials; never copies long-lived secrets into a
  config of its own.
- SSO login uses the OIDC device-authorization grant (`aws-sdk-go-v2/service/ssooidc`);
  the resulting access/refresh token is cached in the same on-disk format and
  location the `aws` CLI v2 uses (`~/.aws/sso/cache/*.json`, `0600`), so tools
  interoperate on the same cached token.
- `identity.NeedsLogin` never retriggers a login for an authorization/service
  error (e.g. `AccessDenied`) - only for a credential-resolution failure that
  looks like a stale/missing SSO token - so invalid permissions never prompt a
  spurious re-login.

## Build and Dependency Management

- Go modules (`go.mod` / `go.sum`).
- `Makefile` targets: `build`, `install`, `test`, `clean`.
- Version stamped via `-ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'`,
  derived from `git describe --tags --always --dirty`.

## Testing Stack

- Go stdlib `testing` only - no third-party test framework.
- Tests live alongside the packages they cover (`cmd/*_test.go`,
  `internal/*/*_test.go`).

## CI/CD and Delivery

- GitHub Actions workflow `.github/workflows/release.yml`:
  - Trigger: pushed tags matching `v*`.
  - `actions/checkout@v4`, `actions/setup-go@v5` (Go version from `go.mod`).
  - Runs `go test ./...` before release.
  - `goreleaser/goreleaser-action@v7.0.0` (`~> v2`) builds and publishes.
  - Uses `HOMEBREW_TAP_TOKEN` for the Homebrew tap update.
- Release builds via GoReleaser v2 (`.goreleaser.yaml`): tar.gz (Linux/macOS),
  zip (Windows), plus `checksums.txt`.
- Distribution: GitHub Releases + Homebrew tap `payfacto/homebrew-tap`
  (`Formula/`). Full runbook in [GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md).

## Infrastructure and Deployment

- No server-side infrastructure. `awsprof` ships as standalone binaries.
- No Dockerfile / Kubernetes / Terraform in the repo.
