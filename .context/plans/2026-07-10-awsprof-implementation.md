# awsprof Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `awsprof`, a self-contained Go CLI that lists AWS profiles, lets you pick/name one, logs in via SSO only when the token is stale, verifies identity, and activates the profile in your current shell.

**Architecture:** Cobra CLI with a thin `cmd/` layer over focused `internal/` packages (`config`, `profiles`, `shell`, `sso`, `identity`, `picker`). Pure logic (parsing, resolution, shell snippets, cache/expiry, error classification) is unit-tested with the stdlib; network glue (AWS SDK v2) sits behind small interfaces so the pure parts are testable and the live SSO flow is a manual smoke test.

**Tech Stack:** Go, `spf13/cobra`, `charmbracelet/huh`, `aws-sdk-go-v2` (`config`, `credentials`, `service/sts`, `service/sso`, `service/ssooidc`), `pkg/browser`, `gopkg.in/yaml.v3`, `gopkg.in/ini.v1`.

## Global Constraints

- Module path: `github.com/payfacto/awsprof-cli`; binary name: `awsprof`.
- Go 1.24 or newer; `go.mod` is authoritative (release CI reads `go-version-file: go.mod`).
- Version variable: `github.com/payfacto/awsprof-cli/cmd.Version` (default `"dev"`, injected via ldflags).
- No dependency on the `aws` CLI at runtime.
- Output contract: activation prints ONLY the shell `export` line to stdout; all human output (identity, progress, errors, picker UI) goes to stderr. No `export` is emitted on any failure.
- Tests: Go stdlib `testing` only; no third-party assertion or mock framework.
- Formatting rules for all generated content (code comments, docs, commits): plain ASCII punctuation only. No em-dashes; use a hyphen. No smart quotes; use straight quotes and `...`.
- Run `gofmt` and `go vet ./...` clean before each commit.

---

### Task 1: Bootstrap module, version, and CLI skeleton

**Files:**

- Create: `go.mod` (via `go mod init`)
- Create: `main.go`
- Create: `cmd/root.go`

**Interfaces:**

- Consumes: nothing (first task).
- Produces: `cmd.Execute() error`; `cmd.Version string` (default `"dev"`, wired into `rootCmd.Version`; this is the exact symbol the Makefile and .goreleaser.yaml inject via ldflags); `rootCmd` (a `*cobra.Command` with `Use: "awsprof [profile]"`, `Args: cobra.MaximumNArgs(1)`). Later tasks attach subcommands via `rootCmd.AddCommand(...)` and fill `rootCmd.RunE`.

- [ ] **Step 1: Initialize the repo, module, and dependencies**

```bash
git init
go mod init github.com/payfacto/awsprof-cli
go get github.com/spf13/cobra@latest
```

Expected: `go.mod` created with module `github.com/payfacto/awsprof-cli` and a `require github.com/spf13/cobra`.

- [ ] **Step 2: Write the root command, version var, and Execute**

Create `cmd/root.go`. `Version` lives in package `cmd` so that
`-X 'github.com/payfacto/awsprof-cli/cmd.Version=...'` (Makefile / .goreleaser.yaml)
targets a real symbol:

```go
// Package cmd wires the awsprof command tree.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the CLI version. It defaults to "dev" for plain `go build` and is
// overridden at release time via -ldflags -X on
// github.com/payfacto/awsprof-cli/cmd.Version.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "awsprof [profile]",
	Short:         "Pick an AWS profile to log in as",
	Args:          cobra.MaximumNArgs(1),
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Filled in Task 11 (picker when no args; activate when one arg).
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 3: Write main.go**

Create `main.go`. It prints any error to stderr before exiting non-zero
(rootCmd sets `SilenceErrors: true`, so main is responsible for the message):

```go
package main

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Build and verify version/help**

```bash
go build -o awsprof .
./awsprof --version
./awsprof --help
```

Expected: builds with no error; `--version` prints `awsprof version dev`; `--help` shows usage with `awsprof [profile]`.

- [ ] **Step 5: Commit**

```bash
gofmt -w . && go vet ./...
git add go.mod go.sum main.go cmd/root.go
git commit -m "feat: bootstrap awsprof module, version, and CLI skeleton"
```

---

### Task 2: Config loader (`~/.awsprof.yaml`)

**Files:**

- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**

- Consumes: nothing.
- Produces: `config.Config{ Prefixes []string }`; `config.Load(path string) (Config, error)` (missing file yields defaults, never an error); `config.DefaultPath() string` (`~/.awsprof.yaml`). Default `Prefixes` is `["payfacto-"]` when none are specified.

- [ ] **Step 1: Add the YAML dependency**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write failing tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"payfacto-"}) {
		t.Fatalf("got %v, want [payfacto-]", cfg.Prefixes)
	}
}

func TestLoad_ReadsPrefixes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "awsprof.yaml")
	if err := os.WriteFile(p, []byte("prefixes: [\"acme-\", \"corp-\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"acme-", "corp-"}) {
		t.Fatalf("got %v", cfg.Prefixes)
	}
}

func TestLoad_EmptyFileReturnsDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "awsprof.yaml")
	if err := os.WriteFile(p, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"payfacto-"}) {
		t.Fatalf("got %v", cfg.Prefixes)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/config/`
Expected: FAIL (package/functions undefined).

- [ ] **Step 4: Write the implementation**

Create `internal/config/config.go`:

```go
// Package config loads the optional ~/.awsprof.yaml settings file.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds awsprof settings. All fields are optional.
type Config struct {
	// Prefixes are tried in order when resolving a short profile name.
	Prefixes []string `yaml:"prefixes"`
}

func defaults() Config {
	return Config{Prefixes: []string{"payfacto-"}}
}

// DefaultPath returns the default config path (~/.awsprof.yaml).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".awsprof.yaml"
	}
	return filepath.Join(home, ".awsprof.yaml")
}

// Load reads the config file at path. A missing file yields defaults and no
// error. A present file with no prefixes also falls back to the default prefix.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Prefixes) == 0 {
		cfg.Prefixes = defaults().Prefixes
	}
	return cfg, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/config/ go.mod go.sum
git commit -m "feat: add ~/.awsprof.yaml config loader with default prefixes"
```

---

### Task 3: Profile discovery, classification, and SSO parsing

**Files:**

- Create: `internal/profiles/profiles.go`
- Test: `internal/profiles/profiles_test.go`

**Interfaces:**

- Consumes: nothing.
- Produces:
  - `profiles.Type` (`TypeUnknown, TypeSSO, TypeStatic, TypeAssumeRole, TypeProcess`).
  - `profiles.SSOConfig{ SessionName, StartURL, Region, AccountID, RoleName string }`.
  - `profiles.Profile{ Name string; Type Type; SSO *SSOConfig }` (SSO non-nil only when Type==TypeSSO).
  - `profiles.ListFrom(configPath, credentialsPath string) ([]Profile, error)` (sorted by Name; missing files tolerated).
  - `profiles.List() ([]Profile, error)` (uses `AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE`, else `~/.aws/config` and `~/.aws/credentials`).

- [ ] **Step 1: Add the ini dependency**

```bash
go get gopkg.in/ini.v1
```

- [ ] **Step 2: Write failing tests**

Create `internal/profiles/profiles_test.go`:

```go
package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func find(ps []Profile, name string) *Profile {
	for i := range ps {
		if ps[i].Name == name {
			return &ps[i]
		}
	}
	return nil
}

func TestListFrom_ClassifiesAndParses(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config", `
[default]
region = us-east-1

[sso-session payfacto]
sso_start_url = https://payfacto.awsapps.com/start
sso_region = us-east-1

[profile payfacto-synapse-admin]
sso_session = payfacto
sso_account_id = 111122223333
sso_role_name = Admin
region = us-east-1

[profile legacy-sso]
sso_start_url = https://old.awsapps.com/start
sso_region = us-west-2
sso_account_id = 444455556666
sso_role_name = ReadOnly

[profile role-prof]
role_arn = arn:aws:iam::123:role/thing
source_profile = default

[profile proc-prof]
credential_process = /usr/bin/cred-helper
`)
	cred := writeFile(t, dir, "credentials", `
[static-prof]
aws_access_key_id = AKIA...
aws_secret_access_key = secret
`)

	ps, err := ListFrom(cfg, cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sso := find(ps, "payfacto-synapse-admin")
	if sso == nil || sso.Type != TypeSSO || sso.SSO == nil {
		t.Fatalf("payfacto-synapse-admin not classified as SSO: %+v", sso)
	}
	if sso.SSO.StartURL != "https://payfacto.awsapps.com/start" || sso.SSO.Region != "us-east-1" {
		t.Fatalf("sso-session not resolved: %+v", sso.SSO)
	}
	if sso.SSO.AccountID != "111122223333" || sso.SSO.RoleName != "Admin" {
		t.Fatalf("sso account/role wrong: %+v", sso.SSO)
	}

	legacy := find(ps, "legacy-sso")
	if legacy == nil || legacy.Type != TypeSSO || legacy.SSO.StartURL != "https://old.awsapps.com/start" {
		t.Fatalf("legacy sso wrong: %+v", legacy)
	}

	if p := find(ps, "role-prof"); p == nil || p.Type != TypeAssumeRole {
		t.Fatalf("role-prof wrong: %+v", p)
	}
	if p := find(ps, "proc-prof"); p == nil || p.Type != TypeProcess {
		t.Fatalf("proc-prof wrong: %+v", p)
	}
	if p := find(ps, "static-prof"); p == nil || p.Type != TypeStatic {
		t.Fatalf("static-prof wrong: %+v", p)
	}
	if p := find(ps, "default"); p == nil {
		t.Fatalf("default profile missing")
	}
}

func TestListFrom_MissingFilesAreTolerated(t *testing.T) {
	ps, err := ListFrom(filepath.Join(t.TempDir(), "none"), filepath.Join(t.TempDir(), "none"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps) != 0 {
		t.Fatalf("expected no profiles, got %v", ps)
	}
}

func TestListFrom_SortedByName(t *testing.T) {
	dir := t.TempDir()
	cfg := writeFile(t, dir, "config", "[profile zeta]\n[profile alpha]\n")
	ps, err := ListFrom(cfg, filepath.Join(dir, "none"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "alpha" || ps[1].Name != "zeta" {
		t.Fatalf("not sorted: %v", ps)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/profiles/`
Expected: FAIL (undefined symbols).

- [ ] **Step 4: Write the implementation**

Create `internal/profiles/profiles.go`:

```go
// Package profiles discovers and classifies AWS named profiles from the shared
// config and credentials files.
package profiles

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// Type is the kind of an AWS profile.
type Type int

const (
	TypeUnknown Type = iota
	TypeSSO
	TypeStatic
	TypeAssumeRole
	TypeProcess
)

// SSOConfig holds the SSO settings resolved for an SSO profile.
type SSOConfig struct {
	SessionName string
	StartURL    string
	Region      string
	AccountID   string
	RoleName    string
}

// Profile is a named AWS profile.
type Profile struct {
	Name string
	Type Type
	SSO  *SSOConfig
}

func configPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

func credentialsPath() string {
	if p := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

// List discovers profiles from the standard AWS file locations.
func List() ([]Profile, error) {
	return ListFrom(configPath(), credentialsPath())
}

// ListFrom discovers profiles from explicit file paths. Missing files are
// treated as empty. Results are sorted by name.
func ListFrom(cfgPath, credPath string) ([]Profile, error) {
	cfg, err := loadINI(cfgPath)
	if err != nil {
		return nil, err
	}
	cred, err := loadINI(credPath)
	if err != nil {
		return nil, err
	}

	sessions := map[string]SSOConfig{}
	for _, s := range cfg.Sections() {
		if name, ok := strings.CutPrefix(s.Name(), "sso-session "); ok {
			sessions[name] = SSOConfig{
				StartURL: s.Key("sso_start_url").String(),
				Region:   s.Key("sso_region").String(),
			}
		}
	}

	cfgSecs := map[string]*ini.Section{}
	names := map[string]bool{}
	for _, s := range cfg.Sections() {
		switch {
		case s.Name() == ini.DefaultSection || strings.HasPrefix(s.Name(), "sso-session "):
			continue
		case s.Name() == "default":
			names["default"] = true
			cfgSecs["default"] = s
		default:
			if pn, ok := strings.CutPrefix(s.Name(), "profile "); ok {
				names[pn] = true
				cfgSecs[pn] = s
			}
		}
	}

	credSecs := map[string]*ini.Section{}
	for _, s := range cred.Sections() {
		if s.Name() == ini.DefaultSection {
			continue
		}
		names[s.Name()] = true
		credSecs[s.Name()] = s
	}

	out := make([]Profile, 0, len(names))
	for name := range names {
		p := Profile{Name: name}
		classify(&p, cfgSecs[name], credSecs[name], sessions)
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func classify(p *Profile, cfgSec, credSec *ini.Section, sessions map[string]SSOConfig) {
	get := func(key string) string {
		if cfgSec != nil {
			if v := cfgSec.Key(key).String(); v != "" {
				return v
			}
		}
		if credSec != nil {
			if v := credSec.Key(key).String(); v != "" {
				return v
			}
		}
		return ""
	}

	if session := get("sso_session"); session != "" {
		sc := SSOConfig{SessionName: session, AccountID: get("sso_account_id"), RoleName: get("sso_role_name")}
		if s, ok := sessions[session]; ok {
			sc.StartURL, sc.Region = s.StartURL, s.Region
		}
		p.Type, p.SSO = TypeSSO, &sc
		return
	}
	if url := get("sso_start_url"); url != "" {
		p.Type = TypeSSO
		p.SSO = &SSOConfig{
			StartURL:  url,
			Region:    get("sso_region"),
			AccountID: get("sso_account_id"),
			RoleName:  get("sso_role_name"),
		}
		return
	}
	switch {
	case get("credential_process") != "":
		p.Type = TypeProcess
	case get("role_arn") != "":
		p.Type = TypeAssumeRole
	case get("aws_access_key_id") != "":
		p.Type = TypeStatic
	default:
		p.Type = TypeUnknown
	}
}

func loadINI(path string) (*ini.File, error) {
	if _, err := os.Stat(path); err != nil {
		return ini.Empty(), nil
	}
	return ini.Load(path)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/profiles/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/profiles/ go.mod go.sum
git commit -m "feat: discover and classify AWS profiles (SSO, static, assume-role, process)"
```

---

### Task 4: Short-name resolution

**Files:**

- Create: `internal/profiles/resolve.go`
- Test: `internal/profiles/resolve_test.go`

**Interfaces:**

- Consumes: nothing from other packages.
- Produces: `profiles.Resolve(input string, prefixes []string, names []string) (string, error)`. Exact match wins; else the first prefix (in order) that yields an existing name wins; else an error whose message is `unknown profile "<input>"`.

- [ ] **Step 1: Write failing tests**

Create `internal/profiles/resolve_test.go`:

```go
package profiles

import "testing"

func TestResolve(t *testing.T) {
	names := []string{"payfacto-synapse-admin", "corp-data", "default"}
	cases := []struct {
		in       string
		prefixes []string
		want     string
		wantErr  bool
	}{
		{"payfacto-synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false},
		{"synapse-admin", []string{"payfacto-"}, "payfacto-synapse-admin", false},
		{"data", []string{"payfacto-", "corp-"}, "corp-data", false},
		{"nope", []string{"payfacto-"}, "", true},
		{"default", nil, "default", false},
	}
	for _, c := range cases {
		got, err := Resolve(c.in, c.prefixes, names)
		if c.wantErr {
			if err == nil {
				t.Errorf("Resolve(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("Resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/profiles/ -run TestResolve`
Expected: FAIL (Resolve undefined).

- [ ] **Step 3: Write the implementation**

Create `internal/profiles/resolve.go`:

```go
package profiles

import "fmt"

// Resolve maps a typed name to an existing profile name: exact match first,
// then each prefix in order. Returns an error if nothing matches.
func Resolve(input string, prefixes []string, names []string) (string, error) {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	if set[input] {
		return input, nil
	}
	for _, pre := range prefixes {
		if cand := pre + input; set[cand] {
			return cand, nil
		}
	}
	return "", fmt.Errorf("unknown profile %q", input)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/profiles/ -run TestResolve`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/profiles/resolve.go internal/profiles/resolve_test.go
git commit -m "feat: resolve short profile names via exact-then-prefix"
```

---

### Task 5: Shell integration (export syntax + hook)

**Files:**

- Create: `internal/shell/shell.go`
- Test: `internal/shell/shell_test.go`

**Interfaces:**

- Consumes: nothing.
- Produces:
  - `shell.Shell` (`Bash, Zsh, Fish, PowerShell`).
  - `shell.Parse(string) (Shell, error)` (case-insensitive; `powershell`/`pwsh` both map to PowerShell).
  - `(Shell).ExportLine(profile string) string`.
  - `(Shell).Hook() string` (the wrapper snippet emitted by `awsprof shell-init`).

- [ ] **Step 1: Write failing tests**

Create `internal/shell/shell_test.go`:

```go
package shell

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Shell{"bash": Bash, "ZSH": Zsh, "fish": Fish, "powershell": PowerShell, "pwsh": PowerShell}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := Parse("tcsh"); err == nil {
		t.Errorf("Parse(tcsh): expected error")
	}
}

func TestExportLine(t *testing.T) {
	cases := map[Shell]string{
		Bash:       "export AWS_PROFILE='dev'",
		Zsh:        "export AWS_PROFILE='dev'",
		Fish:       "set -gx AWS_PROFILE 'dev'",
		PowerShell: "$env:AWS_PROFILE = \"dev\"",
	}
	for sh, want := range cases {
		if got := sh.ExportLine("dev"); got != want {
			t.Errorf("ExportLine(%v) = %q, want %q", sh, got, want)
		}
	}
}

func TestHook_ContainsWrapperAndPassthrough(t *testing.T) {
	h := Bash.Hook()
	if !strings.Contains(h, "awsprof()") {
		t.Errorf("bash hook missing function definition")
	}
	if !strings.Contains(h, "command awsprof") {
		t.Errorf("bash hook missing passthrough to real binary")
	}
	if !strings.Contains(h, "eval") {
		t.Errorf("bash hook missing eval of activation output")
	}
	if !strings.Contains(Fish.Hook(), "function awsprof") {
		t.Errorf("fish hook missing function")
	}
	if !strings.Contains(PowerShell.Hook(), "function awsprof") {
		t.Errorf("powershell hook missing function")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/shell/`
Expected: FAIL (undefined).

- [ ] **Step 3: Write the implementation**

Create `internal/shell/shell.go`:

```go
// Package shell renders per-shell export statements and the shell-init hook.
package shell

import (
	"fmt"
	"strings"
)

// Shell is a supported target shell.
type Shell int

const (
	Bash Shell = iota
	Zsh
	Fish
	PowerShell
)

// Parse maps a shell name to a Shell.
func Parse(s string) (Shell, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bash":
		return Bash, nil
	case "zsh":
		return Zsh, nil
	case "fish":
		return Fish, nil
	case "powershell", "pwsh":
		return PowerShell, nil
	default:
		return 0, fmt.Errorf("unsupported shell %q (want bash|zsh|fish|powershell)", s)
	}
}

// ExportLine returns the statement that sets AWS_PROFILE for this shell.
func (sh Shell) ExportLine(profile string) string {
	switch sh {
	case Fish:
		return fmt.Sprintf("set -gx AWS_PROFILE '%s'", profile)
	case PowerShell:
		return fmt.Sprintf("$env:AWS_PROFILE = %q", profile)
	default:
		return fmt.Sprintf("export AWS_PROFILE='%s'", profile)
	}
}

// Hook returns the shell wrapper printed by `awsprof shell-init <shell>`.
// The wrapper eval's only activation output; data commands pass through so
// their stdout is preserved.
func (sh Shell) Hook() string {
	switch sh {
	case Fish:
		return `function awsprof
    switch $argv[1]
        case list whoami shell-init completion help -h --help -v --version
            command awsprof $argv
        case '*'
            set -l out (command awsprof --shell fish $argv)
            or return
            test -n "$out"; and eval "$out"
    end
end`
	case PowerShell:
		return `function awsprof {
    switch ($args[0]) {
        {$_ -in 'list','whoami','shell-init','completion','help','-h','--help','-v','--version'} {
            & (Get-Command -CommandType Application awsprof).Source @args
        }
        default {
            $out = & (Get-Command -CommandType Application awsprof).Source --shell powershell @args
            if ($LASTEXITCODE -eq 0 -and $out) { Invoke-Expression ($out -join "` + "`n" + `") }
        }
    }
}`
	default: // Bash and Zsh share POSIX syntax.
		return `awsprof() {
  case "$1" in
    list|whoami|shell-init|completion|help|-h|--help|-v|--version)
      command awsprof "$@" ;;
    *)
      local out
      out="$(command awsprof --shell ` + sh.name() + ` "$@")" || return
      [ -n "$out" ] && eval "$out" ;;
  esac
}`
	}
}

func (sh Shell) name() string {
	switch sh {
	case Zsh:
		return "zsh"
	case Fish:
		return "fish"
	case PowerShell:
		return "powershell"
	default:
		return "bash"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/shell/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/shell/
git commit -m "feat: per-shell export lines and shell-init hooks (bash/zsh/fish/powershell)"
```

---

### Task 6: `awsprof list` (read-only end-to-end milestone)

**Files:**

- Create: `cmd/list.go`
- Test: `cmd/list_test.go`

**Interfaces:**

- Consumes: `profiles.List() ([]Profile, error)`.
- Produces: `list` subcommand registered on `rootCmd`. `--plain` prints bare names to stdout; default marks the active profile (from `AWS_PROFILE`) with a trailing ` *`. A pure helper `renderList(ps []profiles.Profile, active string, plain bool) string` is used so it is testable.

- [ ] **Step 1: Write failing test**

Create `cmd/list_test.go`:

```go
package cmd

import (
	"strings"
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func TestRenderList(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}

	plain := renderList(ps, "beta", true)
	if plain != "alpha\nbeta\n" {
		t.Fatalf("plain = %q", plain)
	}

	human := renderList(ps, "beta", false)
	if !strings.Contains(human, "beta *") {
		t.Fatalf("active profile not marked: %q", human)
	}
	if strings.Contains(human, "alpha *") {
		t.Fatalf("non-active profile wrongly marked: %q", human)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestRenderList`
Expected: FAIL (renderList undefined).

- [ ] **Step 3: Write the implementation**

Create `cmd/list.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/spf13/cobra"
)

var listPlain bool

func init() {
	listCmd.Flags().BoolVar(&listPlain, "plain", false, "print bare profile names only")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AWS profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ps, err := profiles.List()
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, renderList(ps, os.Getenv("AWS_PROFILE"), listPlain))
		return nil
	},
}

func renderList(ps []profiles.Profile, active string, plain bool) string {
	var b strings.Builder
	for _, p := range ps {
		if plain {
			fmt.Fprintf(&b, "%s\n", p.Name)
			continue
		}
		mark := ""
		if p.Name == active {
			mark = " *"
		}
		fmt.Fprintf(&b, "%s%s\n", p.Name, mark)
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestRenderList`
Expected: PASS.

- [ ] **Step 5: Manual end-to-end check**

```bash
go build -o awsprof .
mkdir -p testdata
printf '[profile alpha]\n[profile beta]\n' > testdata/config
AWS_CONFIG_FILE=$PWD/testdata/config ./awsprof list
AWS_CONFIG_FILE=$PWD/testdata/config ./awsprof list --plain
```

Expected: `list` prints `alpha` and `beta`; `--plain` prints the bare names. (`testdata/` is throwaway; remove it after.)

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add cmd/list.go cmd/list_test.go
git commit -m "feat: add 'awsprof list' with --plain and active marker"
```

---

### Task 7: SSO token cache (key, read/write, expiry)

**Files:**

- Create: `internal/sso/cache.go`
- Test: `internal/sso/cache_test.go`

**Interfaces:**

- Consumes: nothing.
- Produces:
  - `sso.Token{ AccessToken string; ExpiresAt time.Time; StartURL, Region, ClientID, ClientSecret, RefreshToken string }`.
  - `sso.CacheKey(sessionOrStartURL string) string` (lowercase SHA1 hex).
  - `sso.CacheFilePath(session, startURL string) (string, error)` (`~/.aws/sso/cache/<key>.json`; uses session name when non-empty else start URL).
  - `sso.ReadToken(path string) (Token, error)`.
  - `sso.WriteToken(path string, tok Token) error` (aws-CLI-compatible JSON).
  - `(Token).Valid(now time.Time) bool` (unexpired with a 60s safety skew).

> VERIFY DURING IMPLEMENTATION: the aws CLI keys the cache on SHA1 of the `sso_session` name for modern config and SHA1 of the start URL for legacy config. Confirm against a real `~/.aws/sso/cache/*.json` before relying on interop. The JSON field names below match the aws CLI v2 format; confirm the set present in your environment.

- [ ] **Step 1: Write failing tests**

Create `internal/sso/cache_test.go`:

```go
package sso

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	in := "payfacto"
	sum := sha1.Sum([]byte(in))
	want := hex.EncodeToString(sum[:])
	if got := CacheKey(in); got != want {
		t.Fatalf("CacheKey(%q) = %q, want %q", in, got, want)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tok.json")
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	in := Token{AccessToken: "abc", ExpiresAt: exp, StartURL: "https://x/start", Region: "us-east-1"}
	if err := WriteToken(p, in); err != nil {
		t.Fatal(err)
	}
	got, err := ReadToken(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "abc" || !got.ExpiresAt.Equal(exp) || got.Region != "us-east-1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestTokenValid(t *testing.T) {
	now := time.Now()
	if (Token{ExpiresAt: now.Add(2 * time.Minute)}).Valid(now) != true {
		t.Errorf("token 2m out should be valid")
	}
	if (Token{ExpiresAt: now.Add(30 * time.Second)}).Valid(now) != false {
		t.Errorf("token inside 60s skew should be invalid")
	}
	if (Token{}).Valid(now) != false {
		t.Errorf("zero token should be invalid")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/sso/`
Expected: FAIL (undefined).

- [ ] **Step 3: Write the implementation**

Create `internal/sso/cache.go`:

```go
// Package sso implements the AWS IAM Identity Center (SSO) device-login flow
// and an aws-CLI-compatible token cache.
package sso

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const expirySkew = 60 * time.Second

// Token is a cached SSO access token plus the fields the aws CLI stores.
type Token struct {
	AccessToken  string
	ExpiresAt    time.Time
	StartURL     string
	Region       string
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// cacheJSON mirrors the aws CLI v2 sso/cache token file schema.
type cacheJSON struct {
	AccessToken  string `json:"accessToken"`
	ExpiresAt    string `json:"expiresAt"`
	StartURL     string `json:"startUrl,omitempty"`
	Region       string `json:"region,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

const expiresLayout = "2006-01-02T15:04:05Z"

// CacheKey returns the lowercase SHA1 hex of the given session name or URL.
func CacheKey(sessionOrStartURL string) string {
	sum := sha1.Sum([]byte(sessionOrStartURL))
	return hex.EncodeToString(sum[:])
}

// CacheFilePath returns ~/.aws/sso/cache/<key>.json. It keys on the session
// name when set, else the start URL.
func CacheFilePath(session, startURL string) (string, error) {
	key := session
	if key == "" {
		key = startURL
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "sso", "cache", CacheKey(key)+".json"), nil
}

// ReadToken loads a cached token from path.
func ReadToken(path string) (Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Token{}, err
	}
	var j cacheJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return Token{}, err
	}
	exp, _ := time.Parse(expiresLayout, j.ExpiresAt)
	return Token{
		AccessToken:  j.AccessToken,
		ExpiresAt:    exp,
		StartURL:     j.StartURL,
		Region:       j.Region,
		ClientID:     j.ClientID,
		ClientSecret: j.ClientSecret,
		RefreshToken: j.RefreshToken,
	}, nil
}

// WriteToken writes tok to path in aws-CLI-compatible JSON, creating parents.
func WriteToken(path string, tok Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	j := cacheJSON{
		AccessToken:  tok.AccessToken,
		ExpiresAt:    tok.ExpiresAt.UTC().Format(expiresLayout),
		StartURL:     tok.StartURL,
		Region:       tok.Region,
		ClientID:     tok.ClientID,
		ClientSecret: tok.ClientSecret,
		RefreshToken: tok.RefreshToken,
	}
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Valid reports whether the token is present and not within the expiry skew.
func (t Token) Valid(now time.Time) bool {
	if t.AccessToken == "" && t.ExpiresAt.IsZero() {
		return false
	}
	return t.ExpiresAt.After(now.Add(expirySkew))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sso/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/sso/cache.go internal/sso/cache_test.go
git commit -m "feat: aws-CLI-compatible SSO token cache (key, read/write, expiry)"
```

---

### Task 8: SSO device-authorization login flow

**Files:**

- Create: `internal/sso/login.go`
- Test: `internal/sso/login_test.go`

**Interfaces:**

- Consumes: `sso.Token` (Task 7); `profiles.SSOConfig` (Task 3).
- Produces:
  - `sso.OIDCClient` interface with `RegisterClient`, `StartDeviceAuthorization`, `CreateToken` matching the `ssooidc` client method signatures.
  - `sso.Opener func(url string) error`.
  - `sso.Login(ctx context.Context, c OIDCClient, open Opener, cfg profiles.SSOConfig, now func() time.Time) (Token, error)`: registers a client, starts device auth, opens the browser, and polls until authorized/denied/timeout.

> VERIFY DURING IMPLEMENTATION: confirm the exact `ssooidc` input/output field names and the error types (`types.AuthorizationPendingException`, `types.SlowDownException`, `types.ExpiredTokenException`, `types.AccessDeniedException`) against the installed SDK version. The poll loop below classifies via `errors.As` on those types.

- [ ] **Step 1: Add SDK dependencies**

```bash
go get github.com/aws/aws-sdk-go-v2/service/ssooidc
go get github.com/aws/aws-sdk-go-v2/aws
go get github.com/pkg/browser
```

- [ ] **Step 2: Write failing tests (fake OIDC client)**

Create `internal/sso/login_test.go`:

```go
package sso

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

type fakeOIDC struct {
	createCalls int
	failAfter   int
	denied      bool
}

func (f *fakeOIDC) RegisterClient(_ context.Context, _ *ssooidc.RegisterClientInput, _ ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error) {
	return &ssooidc.RegisterClientOutput{ClientId: aws.String("cid"), ClientSecret: aws.String("csec")}, nil
}

func (f *fakeOIDC) StartDeviceAuthorization(_ context.Context, _ *ssooidc.StartDeviceAuthorizationInput, _ ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	return &ssooidc.StartDeviceAuthorizationOutput{
		DeviceCode:              aws.String("dev"),
		UserCode:                aws.String("USER-CODE"),
		VerificationUriComplete: aws.String("https://device.sso/verify?code=USER-CODE"),
		Interval:                1,
		ExpiresIn:               600,
	}, nil
}

func (f *fakeOIDC) CreateToken(_ context.Context, _ *ssooidc.CreateTokenInput, _ ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	f.createCalls++
	if f.denied {
		return nil, &ssotypes.AccessDeniedException{}
	}
	if f.createCalls <= f.failAfter {
		return nil, &ssotypes.AuthorizationPendingException{}
	}
	return &ssooidc.CreateTokenOutput{AccessToken: aws.String("access-tok"), ExpiresIn: 3600}, nil
}

func TestLogin_SucceedsAfterPending(t *testing.T) {
	var opened string
	cfg := profiles.SSOConfig{SessionName: "payfacto", StartURL: "https://x/start", Region: "us-east-1"}
	tok, err := Login(context.Background(), &fakeOIDC{failAfter: 2}, func(u string) error { opened = u; return nil }, cfg, time.Now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "access-tok" {
		t.Fatalf("token = %+v", tok)
	}
	if opened == "" {
		t.Fatalf("browser opener not called")
	}
	if tok.StartURL != "https://x/start" || tok.Region != "us-east-1" {
		t.Fatalf("token missing session info: %+v", tok)
	}
}

func TestLogin_AccessDenied(t *testing.T) {
	cfg := profiles.SSOConfig{StartURL: "https://x/start", Region: "us-east-1"}
	_, err := Login(context.Background(), &fakeOIDC{denied: true}, func(string) error { return nil }, cfg, time.Now)
	if err == nil || !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("expected ErrLoginDenied, got %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/sso/ -run TestLogin`
Expected: FAIL (undefined).

- [ ] **Step 4: Write the implementation**

Create `internal/sso/login.go`:

```go
package sso

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

// ErrLoginDenied is returned when the user denies the device authorization.
var ErrLoginDenied = errors.New("sso login denied")

// ErrLoginTimeout is returned when the device code expires before authorization.
var ErrLoginTimeout = errors.New("sso login timed out")

// OIDCClient is the subset of the ssooidc client the login flow needs.
type OIDCClient interface {
	RegisterClient(context.Context, *ssooidc.RegisterClientInput, ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error)
	StartDeviceAuthorization(context.Context, *ssooidc.StartDeviceAuthorizationInput, ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error)
	CreateToken(context.Context, *ssooidc.CreateTokenInput, ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error)
}

// Opener opens a URL in the user's browser.
type Opener func(url string) error

// Login runs the device-authorization grant and returns an access token.
func Login(ctx context.Context, c OIDCClient, open Opener, cfg profiles.SSOConfig, now func() time.Time) (Token, error) {
	reg, err := c.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("awsprof"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso:account:access"},
	})
	if err != nil {
		return Token{}, fmt.Errorf("register client: %w", err)
	}

	auth, err := c.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		StartUrl:     aws.String(cfg.StartURL),
	})
	if err != nil {
		return Token{}, fmt.Errorf("start device authorization: %w", err)
	}

	url := aws.ToString(auth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "Opening browser to authorize SSO login...\nIf it does not open, visit: %s\nUser code: %s\n", url, aws.ToString(auth.UserCode))
	_ = open(url)

	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := now().Add(time.Duration(auth.ExpiresIn) * time.Second)

	for {
		if now().After(deadline) {
			return Token{}, ErrLoginTimeout
		}
		out, err := c.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     reg.ClientId,
			ClientSecret: reg.ClientSecret,
			DeviceCode:   auth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err == nil {
			return Token{
				AccessToken:  aws.ToString(out.AccessToken),
				ExpiresAt:    now().Add(time.Duration(out.ExpiresIn) * time.Second),
				StartURL:     cfg.StartURL,
				Region:       cfg.Region,
				ClientID:     aws.ToString(reg.ClientId),
				ClientSecret: aws.ToString(reg.ClientSecret),
				RefreshToken: aws.ToString(out.RefreshToken),
			}, nil
		}

		var pending *ssotypes.AuthorizationPendingException
		var slow *ssotypes.SlowDownException
		var denied *ssotypes.AccessDeniedException
		var expired *ssotypes.ExpiredTokenException
		switch {
		case errors.As(err, &pending):
			// keep polling
		case errors.As(err, &slow):
			interval += 5 * time.Second
		case errors.As(err, &denied):
			return Token{}, ErrLoginDenied
		case errors.As(err, &expired):
			return Token{}, ErrLoginTimeout
		default:
			return Token{}, fmt.Errorf("create token: %w", err)
		}

		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/sso/ -run TestLogin`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/sso/login.go internal/sso/login_test.go go.mod go.sum
git commit -m "feat: SSO device-authorization login flow with pollable OIDC client"
```

---

### Task 9: Identity check and SSO-login classification

**Files:**

- Create: `internal/identity/identity.go`
- Test: `internal/identity/identity_test.go`

**Interfaces:**

- Consumes: nothing from sibling packages (uses AWS SDK).
- Produces:
  - `identity.Identity{ Account, Arn string }`.
  - `identity.NeedsLogin(err error) bool` (pure classifier: true when the error indicates a missing/expired SSO token).
  - `identity.Check(ctx context.Context, profile string) (Identity, error)` (loads shared config for the profile and calls STS GetCallerIdentity).

> VERIFY DURING IMPLEMENTATION: confirm the exact error text/type the SSO credential provider returns for an expired token, and tighten `NeedsLogin` to match (the substring set below is the starting point).

- [ ] **Step 1: Add SDK dependencies**

```bash
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/sts
```

- [ ] **Step 2: Write failing tests (classifier is pure)**

Create `internal/identity/identity_test.go`:

```go
package identity

import (
	"errors"
	"testing"
)

func TestNeedsLogin(t *testing.T) {
	yes := []error{
		errors.New("failed to refresh cached SSO token, the SSO session has expired"),
		errors.New("the SSO session associated with this profile has expired or is otherwise invalid; to refresh this SSO session run aws sso login"),
	}
	for _, e := range yes {
		if !NeedsLogin(e) {
			t.Errorf("NeedsLogin should be true for: %v", e)
		}
	}
	no := []error{
		nil,
		errors.New("AccessDenied: not authorized to perform sts:GetCallerIdentity"),
		errors.New("no EC2 IMDS role found"),
	}
	for _, e := range no {
		if NeedsLogin(e) {
			t.Errorf("NeedsLogin should be false for: %v", e)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/identity/`
Expected: FAIL (undefined).

- [ ] **Step 4: Write the implementation**

Create `internal/identity/identity.go`:

```go
// Package identity resolves and verifies the caller identity for a profile.
package identity

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Identity is the resolved AWS caller identity.
type Identity struct {
	Account string
	Arn     string
}

// Check loads shared config for profile and calls STS GetCallerIdentity.
func Check(ctx context.Context, profile string) (Identity, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return Identity{}, err
	}
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return Identity{}, err
	}
	return Identity{Account: aws.ToString(out.Account), Arn: aws.ToString(out.Arn)}, nil
}

// NeedsLogin reports whether err indicates a missing or expired SSO token
// (as opposed to an authorization failure or a non-SSO credential problem).
func NeedsLogin(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "sso") {
		return false
	}
	return strings.Contains(msg, "expired") ||
		strings.Contains(msg, "run aws sso login") ||
		strings.Contains(msg, "refresh") ||
		strings.Contains(msg, "invalid_grant")
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/identity/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/identity/ go.mod go.sum
git commit -m "feat: STS identity check and SSO-login error classifier"
```

---

### Task 10: Interactive picker

**Files:**

- Create: `internal/picker/picker.go`
- Test: `internal/picker/picker_test.go`

**Interfaces:**

- Consumes: `profiles.Profile` (Task 3).
- Produces:
  - `picker.Item{ Label, Value string }`.
  - `picker.BuildItems(ps []profiles.Profile, active string) []Item` (pure: label shows the name plus ` (active)` for the current profile; value is the bare name).
  - `picker.Pick(items []Item) (string, error)` (huh Select, filterable, rendered to stderr).

- [ ] **Step 1: Add the huh dependency**

```bash
go get github.com/charmbracelet/huh
```

Note: if this resolves to huh v2, the import path becomes `charm.land/huh/v2` with the same `NewSelect` API; adjust the import only.

- [ ] **Step 2: Write failing test (BuildItems is pure)**

Create `internal/picker/picker_test.go`:

```go
package picker

import (
	"testing"

	"github.com/payfacto/awsprof-cli/internal/profiles"
)

func TestBuildItems(t *testing.T) {
	ps := []profiles.Profile{{Name: "alpha"}, {Name: "beta"}}
	items := BuildItems(ps, "beta")
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Value != "alpha" || items[0].Label != "alpha" {
		t.Fatalf("alpha item wrong: %+v", items[0])
	}
	if items[1].Value != "beta" || items[1].Label != "beta (active)" {
		t.Fatalf("active label wrong: %+v", items[1])
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/picker/`
Expected: FAIL (undefined).

- [ ] **Step 4: Write the implementation**

Create `internal/picker/picker.go`:

```go
// Package picker provides the interactive profile selector.
package picker

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/payfacto/awsprof-cli/internal/profiles"
)

// Item is one selectable profile row.
type Item struct {
	Label string
	Value string
}

// BuildItems turns profiles into picker items, marking the active one.
func BuildItems(ps []profiles.Profile, active string) []Item {
	items := make([]Item, 0, len(ps))
	for _, p := range ps {
		label := p.Name
		if p.Name == active {
			label = p.Name + " (active)"
		}
		items = append(items, Item{Label: label, Value: p.Name})
	}
	return items
}

// Pick shows a filterable single-select and returns the chosen profile name.
// The UI renders to stderr so stdout stays reserved for the export line.
func Pick(items []Item) (string, error) {
	opts := make([]huh.Option[string], len(items))
	for i, it := range items {
		opts[i] = huh.NewOption(it.Label, it.Value)
	}
	var selected string
	field := huh.NewSelect[string]().
		Title("Select an AWS profile").
		Options(opts...).
		Filtering(true).
		Value(&selected)
	err := huh.NewForm(huh.NewGroup(field)).WithOutput(os.Stderr).Run()
	return selected, err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/picker/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w . && go vet ./...
git add internal/picker/ go.mod go.sum
git commit -m "feat: filterable huh-based profile picker (renders to stderr)"
```

---

### Task 11: Activation orchestration, root positional, and hidden `use`

**Files:**

- Create: `cmd/activate.go`
- Create: `cmd/use.go`
- Modify: `cmd/root.go` (fill `RunE`, add persistent `--shell` flag)
- Test: `cmd/activate_test.go`

**Interfaces:**

- Consumes: `config.Load`/`DefaultPath`, `profiles.List`/`Resolve`/`SSOConfig`/`TypeSSO`, `identity.Check`/`NeedsLogin`, `sso.*` (login + cache), `picker.BuildItems`/`Pick`, `shell.Parse`/`ExportLine`.
- Produces: `cmd.activate(ctx context.Context, name string, sh shell.Shell) error` (resolves, logs in if an SSO profile needs it, verifies, prints the export to stdout and identity to stderr); root `RunE` calls the picker when no args else `activate`; a hidden `use` command aliases `activate`; a persistent `--shell` string flag (default `bash`).

- [ ] **Step 1: Add SDK deps used to build the OIDC client**

```bash
go get github.com/aws/aws-sdk-go-v2/service/ssooidc
```

(Already present from Task 8; this is a no-op safety check.)

- [ ] **Step 2: Write failing test (unknown profile path, no network)**

Create `cmd/activate_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// activate() must fail fast (before any network) on an unknown profile and
// must not print an export to stdout.
func TestActivate_UnknownProfileNoExport(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	if err := os.WriteFile(cfgPath, []byte("[profile payfacto-real]\nsso_start_url=https://x/start\nsso_region=us-east-1\nsso_account_id=1\nsso_role_name=r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", cfgPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "creds"))

	out := captureStdout(t, func() {
		if err := resolveTargetForTest("does-not-exist"); err == nil {
			t.Fatal("expected error for unknown profile")
		}
	})
	if out != "" {
		t.Fatalf("expected no stdout on failure, got %q", out)
	}
}
```

Add a small stdout capture helper and a resolution seam in `cmd/activate_test.go`:

```go
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	_ = w.Close()
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	if n < 0 {
		n = 0
	}
	return string(buf[:n])
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cmd/ -run TestActivate_UnknownProfileNoExport`
Expected: FAIL (`resolveTargetForTest` undefined).

- [ ] **Step 4: Write the implementation**

Create `cmd/activate.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/payfacto/awsprof-cli/internal/config"
	"github.com/payfacto/awsprof-cli/internal/identity"
	"github.com/payfacto/awsprof-cli/internal/profiles"
	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/payfacto/awsprof-cli/internal/sso"
	"github.com/pkg/browser"
	"time"
)

// resolveTarget maps a typed name to an existing profile using configured prefixes.
func resolveTarget(name string) (profiles.Profile, error) {
	ps, err := profiles.List()
	if err != nil {
		return profiles.Profile{}, err
	}
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return profiles.Profile{}, err
	}
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name
	}
	resolved, err := profiles.Resolve(name, cfg.Prefixes, names)
	if err != nil {
		printProfiles(ps)
		return profiles.Profile{}, err
	}
	for _, p := range ps {
		if p.Name == resolved {
			return p, nil
		}
	}
	return profiles.Profile{}, fmt.Errorf("unknown profile %q", name)
}

// resolveTargetForTest is a thin seam used by tests to exercise resolution.
func resolveTargetForTest(name string) error {
	_, err := resolveTarget(name)
	return err
}

func printProfiles(ps []profiles.Profile) {
	fmt.Fprintln(os.Stderr, "Available profiles:")
	for _, p := range ps {
		fmt.Fprintf(os.Stderr, "  %s\n", p.Name)
	}
}

// activate resolves, logs in if needed, verifies, and emits the export line.
func activate(ctx context.Context, name string, sh shell.Shell) error {
	prof, err := resolveTarget(name)
	if err != nil {
		return err
	}

	id, err := identity.Check(ctx, prof.Name)
	if err != nil {
		if prof.Type == profiles.TypeSSO && identity.NeedsLogin(err) {
			if err := ssoLogin(ctx, *prof.SSO); err != nil {
				return err
			}
			id, err = identity.Check(ctx, prof.Name)
		}
		if err != nil {
			return err
		}
	}

	// stdout: only the export line (for the shell wrapper to eval).
	fmt.Fprintln(os.Stdout, sh.ExportLine(prof.Name))
	// stderr: human confirmation.
	fmt.Fprintf(os.Stderr, "AWS_PROFILE=%s\nAccount %s  %s\n", prof.Name, id.Account, id.Arn)
	return nil
}

func ssoLogin(ctx context.Context, sc profiles.SSOConfig) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(sc.Region))
	if err != nil {
		return err
	}
	client := ssooidc.NewFromConfig(cfg)
	tok, err := sso.Login(ctx, client, browser.OpenURL, sc, time.Now)
	if err != nil {
		return err
	}
	path, err := sso.CacheFilePath(sc.SessionName, sc.StartURL)
	if err != nil {
		return err
	}
	return sso.WriteToken(path, tok)
}
```

Create `cmd/use.go`:

```go
package cmd

import "github.com/spf13/cobra"

var useCmd = &cobra.Command{
	Use:    "use <profile>",
	Short:  "Activate a profile by name (hidden alias of `awsprof <profile>`)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := resolveShell()
		return activate(cmd.Context(), args[0], sh)
	},
}

func init() { rootCmd.AddCommand(useCmd) }
```

Modify `cmd/root.go` to add the `--shell` flag and fill `RunE`. Replace the `rootCmd` var block and add helpers:

```go
var shellFlag string

func init() {
	rootCmd.PersistentFlags().StringVar(&shellFlag, "shell", "bash", "target shell for export syntax (bash|zsh|fish|powershell)")
}

func resolveShell() shell.Shell {
	sh, err := shell.Parse(shellFlag)
	if err != nil {
		return shell.Bash
	}
	return sh
}
```

Update the existing `rootCmd.RunE` to:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := resolveShell()
		if len(args) == 1 {
			return activate(cmd.Context(), args[0], sh)
		}
		ps, err := profiles.List()
		if err != nil {
			return err
		}
		choice, err := picker.Pick(picker.BuildItems(ps, os.Getenv("AWS_PROFILE")))
		if err != nil {
			return err
		}
		if choice == "" {
			return nil
		}
		return activate(cmd.Context(), choice, sh)
	},
```

Add the needed imports to `cmd/root.go`: `os`, `github.com/payfacto/awsprof-cli/internal/picker`, `github.com/payfacto/awsprof-cli/internal/profiles`, `github.com/payfacto/awsprof-cli/internal/shell`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./cmd/ -run TestActivate_UnknownProfileNoExport`
Expected: PASS.

- [ ] **Step 6: Build and manual smoke (needs real SSO)**

```bash
go build -o awsprof .
./awsprof --shell bash payfacto-<real-profile>   # prints export to stdout, identity to stderr
./awsprof                                         # picker
```

Expected: for a valid cached session, prints `export AWS_PROFILE=...` on stdout and identity on stderr; for an expired SSO session, opens the browser, then activates. (Manual: real SSO required.)

- [ ] **Step 7: Commit**

```bash
gofmt -w . && go vet ./...
git add cmd/root.go cmd/activate.go cmd/use.go cmd/activate_test.go go.mod go.sum
git commit -m "feat: activation flow, root positional profile, picker wiring, hidden 'use'"
```

---

### Task 12: `awsprof whoami`

**Files:**

- Create: `cmd/whoami.go`

**Interfaces:**

- Consumes: `identity.Check`.
- Produces: `whoami` subcommand. Prints `AWS_PROFILE=<value>` plus Account and ARN to stdout (whoami is a passthrough command). Exits non-zero with a stderr message when not authenticated.

- [ ] **Step 1: Write the implementation**

Create `cmd/whoami.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/internal/identity"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current AWS identity without switching",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profile := os.Getenv("AWS_PROFILE")
		display := profile
		if display == "" {
			display = "(unset -> default)"
		}
		id, err := identity.Check(cmd.Context(), profile)
		if err != nil {
			return fmt.Errorf("not authenticated for %s: %w", display, err)
		}
		fmt.Fprintf(os.Stdout, "AWS_PROFILE=%s\nAccount %s\n%s\n", display, id.Account, id.Arn)
		return nil
	},
}

func init() { rootCmd.AddCommand(whoamiCmd) }
```

- [ ] **Step 2: Build and manual check**

```bash
go build -o awsprof .
./awsprof whoami
```

Expected: with a valid session, prints the active profile, account, and ARN; otherwise exits non-zero with a stderr error. (Manual: real credentials required.)

- [ ] **Step 3: Commit**

```bash
gofmt -w . && go vet ./...
git add cmd/whoami.go
git commit -m "feat: add 'awsprof whoami' identity command"
```

---

### Task 13: `awsprof shell-init`

**Files:**

- Create: `cmd/shell_init.go`
- Test: `cmd/shell_init_test.go`

**Interfaces:**

- Consumes: `shell.Parse`, `(Shell).Hook`.
- Produces: `shell-init <shell>` subcommand that prints the hook to stdout.

- [ ] **Step 1: Write failing test**

Create `cmd/shell_init_test.go`:

```go
package cmd

import (
	"strings"
	"testing"
)

func TestShellInitOutput(t *testing.T) {
	out, err := renderShellInit("bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "awsprof()") {
		t.Fatalf("bash hook missing: %q", out)
	}
	if _, err := renderShellInit("tcsh"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestShellInitOutput`
Expected: FAIL (renderShellInit undefined).

- [ ] **Step 3: Write the implementation**

Create `cmd/shell_init.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/internal/shell"
	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init <bash|zsh|fish|powershell>",
	Short: "Print the shell hook to add to your shell profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := renderShellInit(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, out)
		return nil
	},
}

func renderShellInit(name string) (string, error) {
	sh, err := shell.Parse(name)
	if err != nil {
		return "", err
	}
	return sh.Hook(), nil
}

func init() { rootCmd.AddCommand(shellInitCmd) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestShellInitOutput`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w . && go vet ./...
git add cmd/shell_init.go cmd/shell_init_test.go
git commit -m "feat: add 'awsprof shell-init' to emit the shell hook"
```

---

### Task 14: Docs sync and final green

**Files:**

- Modify: `README.md`, `llms.txt`, `CLAUDE.md`, `.context/TECHSTACK.md`

**Interfaces:**

- Consumes: the finished command surface.
- Produces: docs whose command reference matches the implemented CLI; TECHSTACK "to be decided" items replaced with the chosen libraries.

- [ ] **Step 1: Update README usage**

In `README.md`, replace the "Usage (in design)" section with the real command set (bare picker, `<profile>`, `list [--plain]`, `whoami`, `shell-init <shell>`), and add per-shell install lines from the shell matrix. Remove the "Status: early / in design" banner.

- [ ] **Step 2: Update llms.txt**

In `llms.txt`, replace the "in design" command block with the final commands and note the shell-hook install requirement and the stdout/stderr contract.

- [ ] **Step 3: Update CLAUDE.md and TECHSTACK.md**

In `CLAUDE.md`, replace the "Intended architecture (to be confirmed in design)" and "Current state" sections with the real package layout and a "code present" state. In `.context/TECHSTACK.md`, move the chosen libraries (cobra, huh, aws-sdk-go-v2 subpackages, pkg/browser, yaml.v3, ini.v1) from "to be decided" into the confirmed sections with versions from `go.mod`.

- [ ] **Step 4: Full suite green**

```bash
gofmt -l .
go vet ./...
go test ./...
go build -o awsprof .
```

Expected: `gofmt -l .` prints nothing; vet clean; all tests PASS; binary builds.

- [ ] **Step 5: Commit**

```bash
git add README.md llms.txt CLAUDE.md .context/TECHSTACK.md
git commit -m "docs: sync README/llms.txt/CLAUDE.md/TECHSTACK to the implemented CLI"
```

---

## Post-implementation manual verification (real SSO required)

These cannot be automated; run them once against a real payfacto SSO profile:

1. `eval "$(./awsprof shell-init bash)"` then `awsprof <short-name>` sets `AWS_PROFILE` in the current shell (`echo $AWS_PROFILE`).
2. With an expired SSO session, `awsprof <name>` opens the browser, completes login, writes `~/.aws/sso/cache/*.json`, and the subsequent `aws sts get-caller-identity` (aws CLI) reuses the same cached token (confirms cache interop and the cache-key hashing).
3. `awsprof list --plain | sort` works (passthrough stdout not eval'd).
4. `awsprof` with no args shows the filterable picker on the terminal while stdout stays clean.
5. Windows PowerShell: `awsprof shell-init powershell | Out-String | Invoke-Expression`, then `awsprof <name>` sets `$env:AWS_PROFILE`.
