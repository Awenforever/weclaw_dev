# WeClaw Dev Developer Handbook

This is the canonical English handoff for starting a new AI development conversation. Keep it concise and current. Long chronological details belong in `docs/development-log.md`.

## 1. Current trusted state

- Project path: `~/projects/weclaw-streaming`
- GitHub repository: `Awenforever/weclaw_dev`
- Main branch: `main`
- Current public Release: `v0.1.7-alpha`
- Current public Release commit: `31fa432`
- Current internal development tag: `p0.1.5a33-internal-version-format-policy`
- Current `main` and `origin/main`: same commit as `p0.1.5a33-internal-version-format-policy` after merge
- Current Release asset/code line: `v0.1p5a28-stop-semantics` at `31fa432`
- p5a29 synchronized documentation and Release notes, p5a30 removed obsolete documentation entry points, p5a31 fixed final documentation entry references, p5a32 synchronized the handoff with the current `main` line, and p0.1.5a33 establishes the normalized internal version/tag format. None of these documentation-only commits should be confused with the Release asset commit.
- Previous public Release `v0.1.6-alpha` remains at `bbb2f28` and must not be moved.
- `v0.1.7-alpha` has been published as Latest and is no longer marked as pre-release.
- Release assets expected for five platforms: Linux amd64, Linux arm64, Darwin amd64, Darwin arm64, and Windows amd64.

## 2. File map

| Path | Role |
| --- | --- |
| `cmd/root.go` | Root CLI command wiring and version display. |
| `cmd/start.go` | `weclaw start`, profile startup, managed process behavior, and resume entry points. |
| `cmd/update.go` | Upgrade flow, channel selection, Release asset download, runtime migration. |
| `cmd/status.go` | CLI-side status reporting. |
| `messaging/handler.go` | WeChat-side message handling, slash commands, `/now`, `/status`, and session display. |
| `agent/acp_agent.go` | ACP/Codex app-server integration, thread start/resume, and event handling. |
| `agent/agent.go` | Agent abstraction and runtime behavior shared by providers. |
| `runtime_state/` | Persisted runtime state, last profile, last ACP/Codex session, and migration metadata. |
| `config/` | Profile and configuration loading. |
| `install.sh` | User installation script and download fallback behavior. |
| `.github/workflows/` | Build and Release automation. |
| `README.md` | English user-facing documentation. Keep usage first and technical details later. |
| `README_CN.md` | Chinese user-facing documentation. Mirror the practical structure of `README.md`. |
| `docs/developer-handbook.md` | English AI startup handoff. |
| `docs/developer-handbook.zh-CN.md` | Chinese maintainer mirror of the handoff. |
| `docs/development-log.md` | Detailed chronological development log. |

## 3. Development working contract

- The user runs commands locally or in a VM and uploads logs.
- Do not guess source layout, runtime state, logs, or configuration.
- If evidence is missing, ask for a read-only audit command.
- Prefer copyable commands and patch scripts over manual file editing.
- Long output must be written to `/tmp/*.txt`; the terminal should show only `run_ok`, `out`, line count, byte count, and a short tail.
- Every audit or patch log must include `stage`, `prefix`, `branch`, `status_count`, `head`, `run_ok`, and `out`.
- Commands must not contain shell-session terminators such as `exit` or `logout`.
- Complex heredocs must use unique long delimiters and outer Markdown fences longer than any inner fences.
- Before modifying files, check `git status --short`.
- Back up key files or diffs under `/tmp` before patching.
- Use `work/<version-description>` branches.
- Completed changes must be committed.
- When pushing a work branch, push the matching internal tag as well.
- Merge to `main` only by fast-forward after validation.

## 4. Version, tag, and Release rules

- Public Release tags use `v0.1.x-alpha`.
- Internal development and handoff tags must use `p<major>.<minor>.<patch>[aN[aM...]][-topic]`.
- Internal tags must start with `p`, must not start with `v`, and must contain three numeric components before any optional `aN` repair/subversion suffix.
- Valid examples include `p0.1.5-topic`, `p0.1.5a1-topic`, `p0.1.5a1a3-topic`, and `p0.1.5a33-internal-version-format-policy`.
- The old `v0.1p...` internal tag style is deprecated. Historical tags may remain for traceability until an explicit tag migration is planned and verified.
- Do not silently move public Release tags.
- Updating an existing public Release tag requires an explicit decision to delete and rebuild that Release and tag.
- Old public Release tags must remain stable unless the user explicitly asks otherwise.
- Internal tags may accumulate but must not be treated as public user install targets.
- Before a Release, the development machine must be synchronized with the current `main` commit.
- Release assets must be checked for all expected platforms.

## 5. Release note rules

- GitHub Release already has a title. The Release body must not repeat a title line such as `WeClaw Dev v0.1.7-alpha`.
- Start the body from sections such as `Highlights`, `Changes`, `Fixes`, `Install`, or `Validation`.
- If only the Release note body is wrong, edit the Release body only. Do not rebuild tags or assets.
- Release notes must describe user-visible behavior and validation status.
- Avoid implementation-only details unless they explain a user-visible change or a known operational risk.

## 6. Documentation maintenance rules

- README files are user-facing. Put install, update, start, resume, CLI commands, and WeChat slash commands before technical details.
- Group WeClaw CLI commands together.
- Group WeChat-side slash commands together.
- Keep internal design, development workflow, and release process out of the README unless directly relevant to users.
- Keep this English handbook as the AI startup handoff.
- Keep the Chinese handbook as the human maintainer mirror.
- Keep long chronological history in `docs/development-log.md`.
- Do not keep obsolete pointer documents just to satisfy stale tests or ghost references. Delete dead entry points and update tests or documentation references to the canonical files.
- Canonical documents are limited to `README.md`, `README_CN.md`, `docs/developer-handbook.md`, `docs/developer-handbook.zh-CN.md`, and `docs/development-log.md`.
- Whenever a milestone permanently changes user workflow or existing CLI behavior, add a README table row with version, affected object, previous behavior, and current behavior.
- Before any Release, confirm the development machine and current `main` commit are synchronized.

## 7. Recent major-version context

`v0.1.7-alpha` closed the p5a20 to p5a25 line.

Key user-visible changes:

- Background mode now keeps default logs at `~/.weclaw/weclaw.log`.
- `weclaw start deepseek-thinking resume` can omit a session ID and reuse the latest ACP/Codex session for that profile.
- `weclaw start ...` is idempotent when a managed process is already running.
- Codex resume uses stable `thread/resume` followed by `turn/start`.
- Default `thread/resume` no longer sends `excludeTurns`.
- `/now` and `/status` share profile and session resolution.
- `/status` should avoid bare unknown context-window values when better source data is available.
- `weclaw upgrade --alpha` can detect same-public-tag but newer-commit upgrades and migrate a managed process.
- `weclaw stop` now reports stopped PIDs or not-running state, clears stale PID state, and verifies that no managed process remains.

## 8. Lessons learned

- Directly overwriting a running binary can trigger `Text file busy`. Use stop, build, install to a new path, then atomic replacement.
- Codex ACP thread identity is not enough for a new app-server process. Real resume requires `thread/resume`.
- `thread/resume.excludeTurns` requires experimental capabilities and must not be used in the stable default path.
- `turn` ID and session/thread ID are different concepts. UI text should say `last turn id` when displaying a turn ID.
- Same public tag with a different commit can happen during alpha rebuilds. Upgrade logic must compare build metadata, not only public version strings.
- Release note cleanup normally does not require rebuilding assets.
- Raw ACP stdout logging is useful for diagnosis but should stay off by default.
- A stop command must be explicit: report what was stopped, clear stale PID state, and verify that follow-up `start` will not immediately see the same managed process.

## 9. Next likely work

- After p0.1.5a33, create new internal development tags only with the normalized `p*.*.*` format.
- Keep README structure user-oriented.
- Keep this handoff concise.
- Continue moving detailed history into `docs/development-log.md`.
- If dsproxy later exposes token attribution, WeClaw should consume it in `/status` and degrade gracefully when the endpoint is absent.
