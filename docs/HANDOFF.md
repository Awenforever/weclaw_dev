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


## p5a22 status and context usage consistency

p5a22 changes:
- `/now` and `/status` share a default profile/session resolver.
- `/now` can fall back to runtime-state session hints when the default agent is not ready.
- `/status` reports context window usage numerically instead of `unknown`.
- Configured context window is read from Codex profile `model_context_window` first.
- If no usage event exists, `/status` reports zero token usage and marks source as `waiting_for_codex_usage_event`.
- Token usage turn ID is labeled as `last turn id`, because turn ID is per-message and not the session/thread ID.

After p5a22, rebuild `v0.1.7-alpha` as pre-release again, then VM-test `/status`, `/now`, one natural-language turn, and `/now` again before marking Latest.


## p5a23 upgrade runtime migration

p5a23 changes:
- `weclaw upgrade` no longer treats the same public version string as sufficient freshness evidence.
- It compares installed public commit metadata with the remote public tag commit.
- Rebuilt `v0.1.7-alpha` assets can be pulled by `weclaw upgrade --alpha` when the installed commit differs.
- Runtime migration remains automatic: after the asset is prepared, upgrade stops the old managed process and restarts the new binary with the resolved profile and session when available.
- Download timeout was increased to tolerate slow VM network paths.

After p5a23, rebuild `v0.1.7-alpha` as pre-release again, then verify upgrade from an older `v0.1.7-alpha` commit to the rebuilt same tag.

## p5a24 README upgrade migration note

p5a24 is a documentation-only rebuild target for validating p5a23 upgrade behavior.

Expected validation:
- VM starts from `v0.1.7-alpha | 0fb7b8f` with internal `v0.1p5a23-upgrade-runtime-migration`.
- `v0.1.7-alpha` is rebuilt to p5a24.
- VM runs `weclaw upgrade --alpha`.
- The p5a23 upgrader should detect that the same public tag now points to a different commit.
- It should download the new asset, replace the binary, stop the old managed process and restart with the prior profile/session when available.
