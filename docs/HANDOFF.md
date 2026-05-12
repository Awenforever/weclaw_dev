# WeClaw Dev Handoff

## v0.1.5-alpha current handoff

Repository: `Awenforever/weclaw_dev`
Local path: `~/projects/weclaw-streaming`
Main branch after release should point to the same commit as `v0.1.5-alpha` and `v0.1p5a16-release-v0.1.5-alpha`.

Release purpose:
- WeClaw can resume a Codex-backed session after restart or upgrade with:
  `weclaw start deepseek-thinking resume <session-id>`
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
