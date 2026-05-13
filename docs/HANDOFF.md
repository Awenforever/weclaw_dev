# WeClaw Dev Handoff

## Current state after p5a19

Repository: `Awenforever/weclaw_dev`
Local path: `~/projects/weclaw-streaming`
Main branch: `main`

Current source baseline:
- `main=origin/main=5dd76d2`
- Internal tag: `v0.1p5a19-resume-default-stdout-latest-acp-session=5dd76d2`
- Previous public Release: `v0.1.6-alpha=bbb2f28`
- `v0.1.6-alpha` must not be moved.

p5a19 changes:
- `weclaw start deepseek-thinking resume` can omit the session ID.
- Omitted resume ID resolves to the most recent ACP/Codex session for the selected profile.
- Background stdout/stderr are saved to `~/.weclaw/weclaw.log` by default.
- Background log trimming keeps newer complete log lines in old-to-new order.
- Upgrade-driven restart also preserves background logging.

## v0.1.7-alpha pre-release purpose

`v0.1.7-alpha` is intended as a GitHub pre-release for VM validation of p5a19 behavior.

Expected Release properties:
- Release title: `WeClaw Dev v0.1.7-alpha`
- GitHub Release must be `prerelease=true`
- Release notes body must start with `Highlights:`
- Release notes body must not repeat the GitHub Release title.
- Five assets must be uploaded:
  - `weclaw_linux_amd64`
  - `weclaw_linux_arm64`
  - `weclaw_darwin_amd64`
  - `weclaw_darwin_arm64`
  - `weclaw_windows_amd64.exe`

## VM validation reminder

Interactive WeChat login requires foreground mode in the VM terminal:

```bash
weclaw start -f
# or
weclaw start deepseek-thinking -f
```

The QR code is displayed in the VM terminal for scanning. After credentials are saved, subsequent validation can use background mode and logs.

For alpha upgrade validation, use:

```bash
weclaw upgrade --alpha
```

Required evidence:
- Version before upgrade
- Target alpha Release
- Installed binary version metadata
- Linux amd64 asset digest matches installed binary
- `~/.weclaw/weclaw.log` is created by default
- `weclaw start deepseek-thinking resume` without session ID behaves according to runtime state
- WeChat message path works after foreground login


## p5a21 correction before final v0.1.7-alpha validation

The earlier `v0.1.7-alpha` pre-release was created successfully but did not pass resume behavior validation.

p5a21 correction:
- Codex ACP resume must call app-server `thread/resume` before `turn/start`.
- WeClaw must not silently replace a missing resumed thread with a new thread.
- `weclaw start ...` must be idempotent when WeClaw is already running.
- `weclaw start ... resume ...` while WeClaw is already running must report that resume was not applied.

The public `v0.1.7-alpha` pre-release may be deleted and rebuilt at the p5a21 commit for validation. Keep it marked as pre-release until VM validation passes. After validation passes, mark the same Release as Latest instead of publishing another tag.
