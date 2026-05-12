# WeClaw Dev Development Manual

This manual is the working map for maintaining `Awenforever/weclaw_dev`. It records source layout, high-risk functions, Git/version rules, Release rules and verification requirements.

## 1. Repository and branch basics

- Repository: `Awenforever/weclaw_dev`
- Main branch: `main`
- Working branches: `work/<version-description>`
- Internal development tags: `v0.1p...`, for example `v0.1p5a11-version-metadata-dual-output`
- Public Release tags during alpha: `v*.*.*-alpha`, for example `v0.1.4-alpha`
- Do not create GitHub Releases for internal `v0.1p...` tags.
- Do not create public GitHub Releases for bare semantic tags such as `v0.1.3`.
- If a bare tag accidentally gets a GitHub Release, delete that GitHub Release and delete the bare local/remote tag unless it is intentionally retained as non-public history.

## 2. Source map

### `cmd/root.go`

Main Cobra root command and version metadata.

Important items:

- `rootCmd`
- `Version`
- `PublicCommit`
- `InternalVersion`
- `InternalCommit`
- `versionOutput(goos, goarch string)`
- `releaseMetadataLDFlags()`

Rules:

- `weclaw version` and `weclaw --version` must print both public and internal metadata.
- Public Release assets must inject all four metadata fields through `-ldflags`.
- Plain local builds may show `dev | unknown`, but public assets must never ship with only `dev`.

Expected output shape:

```text
weclaw public version: v0.1.4-alpha | <commit> (<os>/<arch>)
weclaw internal version: v0.1p5a11-version-metadata-dual-output | <commit> (<os>/<arch>)
```

### `cmd/update.go`

Update, upgrade, uninstall and update notice logic.

Important items:

- `runUpdate`
- `upgradeCmd`
- `upgradeAlpha`
- `getUpgradeTargetVersion`
- `getLatestVersion`
- `getLatestAlphaVersion`
- `downloadFile`
- `replaceBinary`
- `stageReplacementBinary`
- `managedProcessPIDsForExecutable`
- `stopManagedPIDs`
- `upgradeRestartResumeSelection`
- `maybePrintUpdateNotice`
- `shouldOfferUpdate`

Channel rules:

- `weclaw upgrade` defaults to stable GitHub latest, using `/releases/latest`.
- `weclaw upgrade --alpha` is the explicit alpha/pre-release channel.
- Alpha channel must only select draft=false, prerelease=true and tag matching `v*.*.*-alpha`.
- `maybePrintUpdateNotice` should remain stable-channel only unless explicitly redesigned.

Runtime restart rules:

- When upgrading a running managed process, preserve session when possible.
- `upgradeRestartResumeSelection` should prefer `runtime_state` and only fall back to log-derived hints.
- Restart path should call `runDaemon(false, apiAddr, profile, resumeID)` when resume metadata exists.

### `cmd/start.go`

Startup, daemon mode and profile selection.

Important areas:

- `runStart`
- `runDaemon`
- default agent/profile resolution
- pending resume injection into the message handler
- runtime default profile persistence through `runtime_state.SetDefaultProfile`

Rules:

- Starting with `weclaw start <profile> resume <session-id>` must not create a fresh thread before applying the pending resume.
- Daemon restarts from upgrade must carry profile/session when available.

### `cmd/status.go`

CLI-side status command.

Use this for OS/process level status. Do not confuse it with the WeChat slash command `/status`.

### `messaging/handler.go`

WeChat message handling, slash command dispatch and agent turn tracking.

Important areas:

- `HandleMessage`
- `handleRuntimeControl`
- `/now`
- `/status`
- `/profile`
- `/balance`
- `applyPendingResume`
- `recordRuntimeSession`
- `trackAgentSessionIDDuringTurn`
- `buildContextUsageLines`
- `formatContextWindowLines`
- `unknownTokenUsageLines`

Rules:

- `/now` and `/status` must apply pending resume before any `ensureAgentSession` call.
- Runtime session hints should be persisted when a session ID is known.
- `/status` must separate `Context window` from `Token usage`.
- Do not label cumulative token usage as context-window usage.
- If true context used/left is not available, show `used: unknown` and `left: unknown`.

### `runtime_state/runtime_state.go`

Runtime state used for upgrade/session continuity.

Important items:

- `State`
- `SessionHint`
- `Path`
- `LogPath`
- `SetDefaultProfile`
- `UpsertSession`
- `MostRecentSessionForDefaultProfile`
- `MostRecentLogThreadForPIDs`

Rules:

- Runtime state should be stored under `~/.weclaw/runtime-state.json`.
- It may contain profile, user ID and session/thread ID.
- It must not contain API keys, access tokens or secrets.
- Tests must use temporary `HOME`.

### `agent/acp_agent.go`

ACP agent implementation, Codex app-server interaction and token usage events.

Important areas:

- session/thread creation and reuse
- resume handling
- token usage event parsing
- `CurrentTokenUsage`
- ACP subprocess lifecycle

Rules:

- Resume behavior must be validated with real logs before claiming end-to-end success.
- Token usage is not automatically equal to current context-window occupancy.

### `agent/agent.go`

Shared agent interfaces and types.

Important items:

- `Agent`
- `SessionResumer`
- `TokenUsageInspector`
- `TokenUsageSnapshot`

Rules:

- Add interface methods here only when multiple agent implementations can support the behavior.
- Keep test doubles updated when interfaces change.

### `config/`

Configuration loading and default agent/profile logic.

Important files:

- `config/config.go`
- `config/detect.go`
- `config/detect_profile.go`

Rules:

- Do not assume a profile exists. Detect or surface a clear error.
- Avoid storing secrets in debug output or docs.

### `install.sh`

One-line installer, Release asset install path and source-build fallback.

Important items:

- `fetch_release_version`
- `installed_binary_version`
- `install_release`
- `install_from_source`
- `install_binary_file`
- `curl_fetch`

Rules:

- Default install should resolve GitHub stable latest.
- Source fallback must inject version metadata fields when building.
- Same-version rerun should skip redundant asset download.
- Long network operations should use retry and fallback.
- Do not delete user data during install or ordinary uninstall.

### `.github/workflows/`

CI and GitHub-side build/release workflows.

Important files:

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`

Rules:

- Public Release builds must inject all four version fields.
- Do not rely on internal tags for GitHub public Releases.
- If workflow behavior differs from manual Release procedure, update this manual.

### `README.md` and `README_CN.md`

User-facing documentation.

Rules:

- README should stay concise.
- Detailed development and release procedure belongs in this manual.
- When changing commands, update README and this manual together.

## 3. Git workflow rules

1. Start from clean `main`.
2. Confirm `main == origin/main`.
3. Create `work/<version-description>` branch.
4. Back up important files or save diff to `/tmp`.
5. Apply patch.
6. Run focused tests and full tests.
7. Commit.
8. Create internal tag on the work-branch commit.
9. Push work branch and internal tag.
10. Fast-forward merge to `main`.
11. Push `main`.
12. Verify `main`, `origin/main`, work branch and internal tag all point to the expected commit.

Never commit from a dirty or ambiguous state.

## 4. Internal tag and public Release rules

Internal tags:

- Pattern: `v0.1p...`
- Example: `v0.1p5a11-version-metadata-dual-output`
- Purpose: development checkpoint and troubleshooting
- Must not have a GitHub Release

Public Release tags during alpha:

- Pattern: `v*.*.*-alpha`
- Example: `v0.1.4-alpha`
- Must point to the commit used to build Release assets
- Should be marked as pre-release when intentionally exposing an alpha channel
- Must not be silently moved

Forbidden public Release tags during alpha:

- Bare `v*.*.*`, such as `v0.1.3`
- Internal `v0.1p...` tags

## 5. Upgrade channel rules

Default stable channel:

```bash
weclaw upgrade
```

- Uses GitHub stable latest.
- Should not opt users into alpha/pre-release.

Explicit alpha channel:

```bash
weclaw upgrade --alpha
```

- Uses latest GitHub pre-release matching `v*.*.*-alpha`.
- Intended for testers and controlled verification.

Installer:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

- Should install stable latest by default.
- Explicit version install can use `--version v0.1.4-alpha`.

## 6. Release build metadata

Public Release assets must be built with all four metadata fields:

```bash
-ldflags="-s -w \
  -X github.com/fastclaw-ai/weclaw/cmd.Version=<public-release-tag> \
  -X github.com/fastclaw-ai/weclaw/cmd.PublicCommit=<public-release-commit> \
  -X github.com/fastclaw-ai/weclaw/cmd.InternalVersion=<internal-p-tag> \
  -X github.com/fastclaw-ai/weclaw/cmd.InternalCommit=<internal-p-commit>"
```

Example for an alpha Release:

```text
public version:  v0.1.4-alpha | <public commit>
internal version: v0.1p5a11-version-metadata-dual-output | <internal commit>
```

## 7. Minimum test matrix

For code changes:

```bash
git diff --check
go test ./cmd ./config ./agent ./messaging ./runtime_state
go test ./...
```

For shell installer changes:

```bash
bash -n install.sh
```

For version metadata changes:

```bash
weclaw version
weclaw --version
```

Expected checks:

- plain local build prints `dev | unknown`
- Release-style build prints public Release tag and internal tag
- both `version` and `--version` use the same two-line metadata format

For upgrade/session changes:

- unit tests are not enough
- run VM test with a running process
- verify old process stops
- verify new process starts
- verify restart uses profile/session from runtime state
- verify logs show the expected resume marker

## 8. Release procedure

1. Confirm `main == origin/main`.
2. Confirm working tree is clean.
3. Confirm no forbidden public Release tags exist.
4. Confirm public Release tag target, for example `v0.1.4-alpha`.
5. Confirm internal tag target, for example `v0.1p5a11-version-metadata-dual-output`.
6. Build all platform assets with full metadata.
7. Create annotated public tag.
8. Create GitHub Release.
9. Mark alpha Releases as pre-release unless intentionally promoting to stable.
10. Upload five platform assets:
   - `weclaw_linux_amd64`
   - `weclaw_linux_arm64`
   - `weclaw_darwin_amd64`
   - `weclaw_darwin_arm64`
   - `weclaw_windows_amd64.exe`
11. Verify Release metadata and asset states.
12. Run VM user-path install or upgrade validation.
13. Record the final state in handoff notes.

## 9. VM verification rules

Use VM for user-path verification when any of these change:

- installer
- upgrade
- Release assets
- runtime restart/session behavior
- version metadata
- public/internal tag mapping

Do not claim end-to-end behavior unless the VM log proves it.

For alpha upgrade validation:

```bash
weclaw upgrade --alpha
```

Required evidence:

- old version before upgrade
- target alpha Release
- binary replacement succeeded
- old PID stopped
- new PID started
- version metadata matches Release
- session resume marker exists
- `/now` or `/status` reports expected session behavior when applicable

## 10. Logging and command discipline

- Long output goes to `/tmp/*.txt`.
- Terminal output should show only `run_ok`, `out`, line count, byte count and tail.
- Logs must include:
  - `stage`
  - `prefix`
  - `branch`
  - `status_count`
  - `head`
  - `out`
  - `run_ok`
- Do not print secrets, full environment dumps or full source trees.
- Do not use commands that end the shell session.
- Avoid dangerous deletion outside the intended repository or `/tmp` scope.

## 11. Current high-risk areas

- Running-process upgrade and session restoration.
- Public vs internal version metadata drift.
- GitHub Release tag hygiene.
- jsDelivr cache delay after `main` changes.
- Network instability for GitHub raw, Release assets, codeload and Go proxy.
- Confusing cumulative token usage with context-window occupancy.
