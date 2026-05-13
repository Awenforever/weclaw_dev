# WeClaw Dev Handoff

## v0.1.5-alpha current handoff

Repository: `Awenforever/weclaw_dev`
Local path: `~/projects/weclaw-streaming`
Main branch after release should point to the same commit as `v0.1.5-alpha` and `v0.1p5a16-release-v0.1.5-alpha`.

Release purpose:
- WeClaw can resume a Codex-backed session after restart or upgrade with:
  `weclaw start deepseek-thinking resume <session-id>
# Or omit the session ID to resume the most recent ACP/Codex session for this profile:
weclaw start deepseek-thinking resume`
- If the old Codex thread cannot be recovered, WeClaw creates a replacement session and continues the user message instead of returning `thread not found`.
- `weclaw upgrade` remains the normal stable/latest upgrade path.
- `weclaw upgrade --alpha` is the explicit alpha/pre-release channel.

Validation required before or during release:
- `git diff --check`
- focused resume tests
- broader package tests
- full `go test ./...`
- five release assets uploaded:
  `weclaw_linux_amd64`, `weclaw_linux_arm64`, `weclaw_darwin_amd64`, `weclaw_darwin_arm64`, `weclaw_windows_amd64.exe`

Release note rule:
- Do not start the Release body with `WeClaw Dev v...`.
- Start with `Highlights:` or another short content heading.

## Current handoff after v0.1.6-alpha

Current repository: `Awenforever/weclaw_dev`
Local development path: `~/projects/weclaw-streaming`
Main branch after release: `main=origin/main=bbb2f28`
Public Release: `v0.1.6-alpha` at `bbb2f28`
Internal development tag for the release commit: `v0.1p5a17-codex-bwrap-warning-handling` at `bbb2f28`
Previous public Release: `v0.1.5-alpha` at `de2ba03`

Release status:
- `v0.1.6-alpha` has been published as GitHubLatestRelease.
- Release title is `WeClaw Dev v0.1.6-alpha`.
- Release is not draft and not prerelease.
- Five assets are uploaded:
  - `weclaw_linux_amd64`
  - `weclaw_linux_arm64`
  - `weclaw_darwin_amd64`
  - `weclaw_darwin_arm64`
  - `weclaw_windows_amd64.exe`
- Release notes must continue to start with `Highlights:` and must not repeat the GitHubRelease title.

User-visible changes in `v0.1.6-alpha`:
- WeChat-facing Codex sandbox dependency errors are cleaner.
- ANSI terminal escape sequences are removed from Codex error messages before showing them to users.
- Codex `bubblewrap` sandbox diagnostics are converted into a short install-and-restart suggestion.
- The session resume behavior from `v0.1.5-alpha` remains available with `weclaw start deepseek-thinking resume <session-id>
# Or omit the session ID to resume the most recent ACP/Codex session for this profile:
weclaw start deepseek-thinking resume`.
- `weclaw upgrade` and `weclaw upgrade --alpha` remain supported.

Important boundary:
- `v0.1.6-alpha` points to `bbb2f28` and must not be moved.
- Subsequent documentation-only commits may move `main`, but they must not move `v0.1.6-alpha`.
- If a new public user-visible build is needed, publish a new public tag such as `v0.1.7-alpha`.

Remaining validation:
- Run VM user-path validation for `v0.1.6-alpha`.
- Confirm LatestRelease install or upgrade yields `public version: v0.1.6-alpha | bbb2f28`.
- Confirm Linuxamd64 asset digest matches the installed binary.
- Confirm same-version `weclaw upgrade` reports already up to date.
