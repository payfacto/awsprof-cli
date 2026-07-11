# TECHSTACK.md

Technology stack for `awsprof` - a Go CLI for picking an AWS profile to log in
as. The project was seeded from a Go + Cobra CLI template (`bb`) and reuses its
build/release machinery; the application libraries are being chosen in design.

Entries are split into the **confirmed base** (carried from the template and
release pipeline) and **to be decided** (settled during the command-surface
design pass). Once `go.mod` exists it becomes the authoritative source and this
file should be refreshed from it.

## Language and Runtime (confirmed)

- Go (targeting a current stable release, ~1.26; `go.mod` will pin the exact
  version once created).
- Single-binary CLI; entrypoint `main.go` -> `cmd.Execute()`.
- Cross-compiled for `linux`, `darwin`, `windows` x `amd64`, `arm64` with
  `CGO_ENABLED=0` (per `.goreleaser.yaml`).

## Core Frameworks and Libraries

Confirmed:

- CLI framework: `github.com/spf13/cobra` (intended; standard for the template).

To be decided in design:

- **AWS shared-config parsing** - how to read `~/.aws/config` /
  `~/.aws/credentials` and resolve profiles/SSO sessions. Candidates:
  `github.com/aws/aws-sdk-go-v2/config` (+ `aws/config`), a dedicated ini parser
  (`gopkg.in/ini.v1`), or a small hand-rolled reader. Decide based on how much
  SSO/assume-role resolution awsprof needs to do itself vs. delegate to the AWS
  CLI.
- **Interactive picker** - for `awsprof` with no subcommand. Candidates:
  `charmbracelet/bubbletea` + `bubbles`, `charmbracelet/huh`, `manifoldco/promptui`,
  or `AlecAivazis/survey`. Pick the lightest option that fits the UX.
- **Activation mechanism** - whether awsprof shells out to the `aws` CLI
  (`aws sso login`), sets/prints `AWS_PROFILE`, or emits exportable credentials.

## Data and Persistence

- No application database.
- Reads AWS shared config/credentials from `~/.aws/` (overridable via
  `AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE`).
- Whether awsprof keeps any state of its own (e.g. a "last used profile") is TBD.

## Security and Secrets

- Reads existing AWS credentials; must not copy long-lived secrets into a config
  of its own or log them.
- SSO/short-lived-credential handling is delegated to AWS tooling where possible.
- Exact secret-handling rules to be defined alongside the activation mechanism.

## Build and Dependency Management (confirmed)

- Go modules (`go.mod` / `go.sum`) - to be initialized.
- `Makefile` targets: `build`, `install`, `test`, `clean`.
- Version stamped via `-ldflags -X 'github.com/payfacto/awsprof-cli/cmd.Version=...'`,
  derived from `git describe --tags --always --dirty`.

## Testing Stack (confirmed)

- Go stdlib `testing` only - no third-party test framework planned.
- Tests live alongside the packages they cover.

## CI/CD and Delivery (confirmed)

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
