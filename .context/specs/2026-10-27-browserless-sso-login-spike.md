# Browserless SSO login spike (webframe / glaze)

Status: spike only, nothing merged. Date: 2026-10-27. Platform tested: macOS (arm64).

## Question

Can `awsprof` complete the IAM Identity Center device-authorization login without
the user going to a browser: password from the macOS Keychain, TOTP code
prompted at the CLI?

## Answer

Feasible on macOS with TOTP. Reached the MFA screen with a scripted username +
password login. The MFA submit, the consent ("Allow") screen and the real
device-flow URL were NOT tested.

## Findings

- **Directory:** start URL `d-xxxxxxxxxx.awsapps.com/start` redirects to
  `ca-central-1.signin.aws/platform/d-xxxxxxxxxx/login`, the built-in Identity
  Center sign-in (not an external IdP). Three screens: username + Next, password
  + Sign in, MFA code + Sign in.
- **webframe does not work as-is.** Its `Eval` runs `eval()` in the page and the
  AWS CSP rejects it (`unsafe-eval` not allowed). Use the underlying
  `github.com/crgimenes/glaze` (resolved via `progrium/glaze`) directly:
  `WebView.Eval(js)` is not blocked by the CSP. It is fire-and-forget, so
  results come back through `Bind("name", func(...))`.
- **Module quirks:** import path is `tractor.dev/webframe` (not the GitHub path),
  and its `go.mod` has a `replace github.com/crgimenes/glaze =>
  github.com/progrium/glaze <version>` that consumers must repeat. The project is
  very young (4 commits at time of writing).
- **React inputs:** set the value with the native
  `HTMLInputElement.prototype.value` setter, then dispatch `input` and `change`
  events, then click the visible `button[type=submit]` (after ~300 ms).
- **Fixed sleeps are fragile:** poll for the expected field (for example
  `input[type=password]`) before filling, and abort if it is missing. During the
  first run an early fill landed in the wrong field.
- **Bot detection:** page loads `fwcim_signin` and `threat-mitigation` scripts.
  No CAPTCHA or block page appeared in the runs.
- **Unexplained:** one run showed "We couldn't verify your sign-in credentials"
  on the password screen, then later showed the MFA screen without another click.
  Suspected race with the fraud-detection script; unconfirmed. Needs repeat runs
  (3+) to see whether it is deterministic.
- **Keychain:** `security add-generic-password` must run in a real terminal. Via
  Claude Code's `!` prefix there is no tty, so it stores an empty password
  silently. Reading it with `security find-generic-password -s <svc> -a <acct> -w`
  triggers a one-time macOS "Allow" dialog.

## Risks and open items

- Selectors and flow are tied to AWS sign-in page markup; any change breaks it.
  A browser fallback (current flow) must remain the default.
- Passkey or security-key MFA cannot be scripted; only TOTP fits.
- Check PayFacto security policy on scripted credential entry before shipping.
- Webview has no persistent session, so every login is a full login.

## If resumed: step 4 plan (about 1 hour)

1. Run the real device flow: `StartDeviceAuthorization`, open
   `verificationUriComplete` in the glaze webview.
2. Same login script, then prompt `MFA code:` on the terminal and fill it in.
3. Click Allow, let the existing token polling finish.
4. Run 3 times and record how often the "couldn't verify" banner appears.

Then, if it holds: macOS-only build tags, opt-in flag or config (for example
`--no-browser`), password in Keychain only, never on disk or in argv.

## Spike code

Throwaway, not in the repo: `/tmp/webframe-spike/main.go` (may be gone). Shape:
`glaze.NewWithOptions`, `Bind("__log", ...)`, `Navigate`, then timed `Dispatch(Eval(js))`
calls with a guarded fill helper and a probe that logs url, visible inputs,
buttons and body text. Test Keychain item: service `awsprof-spike`, account
`<username>`; remove with `security delete-generic-password -s awsprof-spike -a <username>`.
