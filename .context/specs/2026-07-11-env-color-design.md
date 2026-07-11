# Environment coloring for profile display

Approved 2026-07-11. Design for color-coding AWS profiles by the environment
detected in their name, shown in `list`, the interactive picker, and `whoami`.

## Goal

Give an at-a-glance signal of which environment a profile touches, so the user
can tell prod from dev without reading the whole name. Profile names follow a
regular convention (`payfacto-<app>-<env>-<role>`, e.g.
`payfacto-titan-prod-readonly`), so the environment is a hyphen-delimited
segment we can detect and color.

## Detection

- Split the profile name on `-`. Lowercase each segment. The **first** segment
  that matches a known keyword decides the environment; later segments are
  ignored.
- Whole-segment match only (not substring), so `payfacto-sandbox-admin` matches
  the `sandbox` segment while `payfacto-synapse-readonly` matches nothing (no
  false match on `syn`... etc.).
- No match -> `EnvNone`, and the name renders with no color.

### Environment -> keywords -> color

| Env | Keywords (case-insensitive) | Color | Weight |
| --- | --- | --- | --- |
| prod | `prod`, `production` | red `#ff5c57` | **bold** |
| staging | `staging`, `stage`, `stg` | orange `#ff9f43` | normal |
| uat | `uat` | purple `#c586e0` | normal |
| qa | `qa` | yellow `#f2cc60` | normal |
| dev | `dev`, `development` | green `#57ab5a` | normal |
| sandbox | `sandbox`, `test`, `sbx` | blue `#54aeff` | normal |
| (none) | anything else | none | normal |

`prod` is the only bold entry - it is the "don't fat-finger this" environment
and should be unmistakable.

## Rendering - "Style D"

Only the matched env segment is colored; the rest of the name renders in the
terminal's default foreground. Example (env segment shown in brackets):

```
payfacto-titan-[prod]-readonly        prod segment bold red
payfacto-gateway-[staging]-poweruser  staging segment orange
payfacto-synapse-readonly             unchanged (no env)
```

This keeps the full name readable while drawing the eye to the environment. It
was chosen over whole-name coloring (too loud), a leading badge (extra width),
and a dimmed-remainder variant (unnecessary de-emphasis).

## Color mechanism

Render through `github.com/charmbracelet/lipgloss` (already in the module graph
indirectly via huh; promoted to a direct dependency). lipgloss is chosen over
hand-rolled ANSI because it:

- auto-detects the terminal's color profile (TrueColor / 256 / 16 / none) and
  degrades the hex colors accordingly,
- enables Windows virtual-terminal processing,
- honors `NO_COLOR` and disables styling when the output stream is not a TTY.

A `*lipgloss.Renderer` is bound to the specific output stream:

- `os.Stdout` for `list` and `whoami`,
- `os.Stderr` for the picker and the unknown-profile fallback list (the picker
  UI already renders to stderr).

## Package layout

New pure package **`internal/envcolor`**:

- `type Env` with constants `EnvNone`, `EnvProd`, `EnvStaging`, `EnvUAT`,
  `EnvQA`, `EnvDev`, `EnvSandbox`.
- `Detect(name string) (Env, int)` - returns the env and the index of the
  matched segment (index `-1` for `EnvNone`). Pure; the unit of truth for
  detection, independent of any rendering.
- `Render(name string, r *lipgloss.Renderer) string` - applies Style D using the
  detection result and the palette. `EnvNone` returns `name` unchanged. Whether
  color actually appears is decided by the renderer's detected profile, so this
  is safe to call unconditionally.
- Hardcoded alias map and palette live here. Not configurable in v1 (see
  Non-goals).

## Wiring

- **`cmd/list.go`** `renderList`: color each name via a stdout-bound renderer.
  `--plain` bypasses coloring entirely (and every other decoration) so the
  scripting/pipe contract stays byte-clean.
- **`internal/picker`** `BuildItems`: color the label text. Risk: huh applies
  its own foreground style to the *selected* row and may override the injected
  color. Verify on a live TTY; if huh clobbers it, fall back to a small colored
  leading marker (e.g. a colored dot) that survives huh's row styling. The
  picker renderer is stderr-bound.
- **`cmd/whoami.go`**: color the active profile name in the `AWS_PROFILE=<name>`
  line.
- **`cmd/activate.go`** `printProfiles` (the unknown-profile fallback list on
  stderr): color for consistency with `list`. Minor, in-spirit addition.

The activation export line (`cmd/activate.go`, the single stdout write consumed
by the shell hook) is **never** colored - stdout there must stay a clean
`export AWS_PROFILE=...` for `eval`.

## Terminal safety

Handled by lipgloss profile detection, asserted as defaults (no flag):

- Color only when the target stream is a TTY and `NO_COLOR` is unset.
- `list --plain` and any piped/redirected output render with no escape codes.
- The hook-consumed export line stays uncolored regardless.

## Testing

Stdlib `testing`, table-driven:

- `Detect`: each env and every alias; case-insensitivity; no-env; segment-
  boundary non-match (a keyword embedded in a larger segment must not match);
  first-match-wins when two env-like segments appear.
- `Render`: force a color profile on via an explicit lipgloss renderer/profile so
  output is deterministic in tests; assert the env segment is wrapped and the
  remainder is untouched; `EnvNone` returns the input unchanged.
- `renderList`: `--plain` output contains no escape codes and equals the bare
  names.

## Non-goals (v1)

- YAML-configurable env keywords or colors (`~/.awsprof.yaml`). Defaults are
  hardcoded; revisit only if naming conventions diverge.
- A `--color=always|never` flag. Auto-detection + `NO_COLOR` covers the cases.
- Light-vs-dark adaptive hues. The palette leans readable-on-dark; if a light
  terminal washes out qa-yellow, switch those entries to
  `lipgloss.AdaptiveColor` later.
