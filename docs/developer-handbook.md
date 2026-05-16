# WeClaw Dev Developer Handbook

This is the canonical English handoff for starting a new AI development conversation. Keep it concise and current. Long chronological details belong in `docs/development-log.md`.

## 1. Current trusted state

- Project path: `~/projects/weclaw-streaming`
- GitHub repository: `Awenforever/weclaw_dev`
- Main branch: `main`
- Current public Release: `v0.1.8-alpha`
- Current public Release commit: `05cb93c`
- Current Release internal marker: `p0.1.5a50-outbound-markdown-capture` at `05cb93c`
- Current internal development tag: `p0.1.5a56-mainline-tracker-audit-policy`
- Last audited handoff baseline before this sync: `main=origin/main=p0.1.5a55-cross-project-profile-boundary-docs=cad0437`
- Current active development line: `p0.1.5a56-mainline-tracker-audit-policy`. Resolve its exact commit from Git instead of trusting a copied static hash.
- Previous public Release `v0.1.7-alpha` remains at `31fa432` and must not be moved.
- `v0.1.8-alpha` GitHub Release title is `WeClaw Dev v0.1.8-alpha`, is not draft, is not prerelease, and has five uploaded assets.
- Expected Release assets: Linux amd64, Linux arm64, Darwin amd64, Darwin arm64, and Windows amd64.
- p0.1.5a51 only documented the VM GitHub Release asset download failure and host-proxy fix. It did not move the public Release.
- Start every future task with a read-only audit of `main`, `origin/main`, the active internal tag, public Release tags, clean worktree state, and the handbook/log heads.

## 2. Long-term mainline task tracker

Update this table whenever a cross-conversation task changes scope, status, expected metrics, or ownership. Treat it as the first progress checkpoint for any new development conversation.

Status vocabulary: `planned`, `in_progress`, `verified`, `blocked`, `done`, `superseded`.

| Mainline item | Expected metric or acceptance condition | Current version or source | Current status | Last maintained | Notes |
| --- | --- | --- | --- | --- | --- |
| Full telemetry contract baseline | Local `dsproxy` exposes profile status and WeClaw status JSON for `deepseek` and `deepseek-thinking`, including `model`, `effort`, `context_window`, `tokens`, `pricing`, `cost`, `balance`, and `compaction`. | CoDeepSeedeX `p2.10a48-weclaw-full-telemetry-contract`; WeClaw audit before `p0.1.5a56` | verified | 2026-05-16 | Contract is accepted as the first engineering baseline. WeClaw still needs integration work. |
| WeClaw ownership boundary | WeClaw does not directly edit Codex profile files for normal `/effort`, `/model`, `/status`, or telemetry paths when `dsproxy` provides a structured contract. | WeClaw `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | Existing WeClaw code still contains profile repair logic and must be refactored in the next implementation branch. |
| `/effort` integration | `/effort max` calls the authoritative `dsproxy profile set-effort <profile> max --json` contract and displays `effort.user_facing` or `effort.deepseek_reasoning_effort`. | Pending WeClaw implementation after CoDeepSeedeX `p2.10a48` | planned | 2026-05-16 | Do not widen WeClaw into a global Codex profile repair layer. |
| `/status` contract integration | `/status` consumes `dsproxy status <route> --weclaw-json` or the equivalent HTTP endpoint and renders only returned data. Missing or unavailable fields degrade explicitly. | Pending WeClaw implementation after CoDeepSeedeX `p2.10a48` | planned | 2026-05-16 | Token-level context and char-level runtime compaction must not be merged into one progress bar without unit labels. |
| Telemetry display quality | Mobile WeChat output remains compact and Markdown-first while showing model, effort, context window, token usage, estimated cost, balance, and compaction from `dsproxy`. | Pending WeClaw implementation | planned | 2026-05-16 | Do not maintain model prices, balance logic, or prompt subcategory token estimates in WeClaw. |
| Evidence-first audit discipline | Source and document changes are based on full source files, full canonical documents, or complete function/module blocks rather than isolated grep snippets. | `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | Grep/rg may help locate symbols or verify markers, but it is not sufficient evidence for patch design. |
| Cross-project feedback loop | After each WeClaw integration round, produce a precise CoDeepSeedeX follow-up prompt for missing fields, ambiguous semantics, or unstable contract behavior. | Pending after first WeClaw integration branch | planned | 2026-05-16 | Keep the prompt grounded in actual WeClaw implementation results and runtime logs. |
| Release readiness | README, handbooks, development log, focused tests, full tests, and local runtime version metadata are consistent before any public Release decision. | Public Release remains `v0.1.8-alpha` at `05cb93c` | planned | 2026-05-16 | Internal tracker updates must not move public Release tags or rebuild assets. |

## 3. File map

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

## 4. Development working contract

- The user runs commands locally or in a VM and uploads logs.
- Do not guess source layout, runtime state, logs, or configuration.
- If evidence is missing, ask for a read-only audit command.
- When source or documentation changes require structural judgment, first request complete source files, full canonical documents, or complete function/module blocks from the user. Grep or rg excerpts are navigation aids only and must not be treated as sufficient patch evidence.
- Before patching, record the reviewed files or blocks, expected markers, forbidden markers, validation rules, and test assertions. If the required context is unavailable, do a full-context audit before patching.
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

## 5. Version, tag, and Release rules

- Public Release tags use `v0.1.x-alpha`.
- Internal development and handoff tags must use `p<major>.<minor>.<patch>[aN[aM...]][-topic]`.
- Internal tags must start with `p`, must not start with `v`, and must contain three numeric components before any optional `aN` repair/subversion suffix.
- Valid examples include `p0.1.5-topic`, `p0.1.5a1-topic`, `p0.1.5a1a3-topic`, and `p0.1.5a33-internal-version-format-policy`.
- The pre-normalization internal tag style is deprecated. Historical traceability is maintained through normalized mirror tags and the development log.
- Do not silently move public Release tags.
- Updating an existing public Release tag requires an explicit decision to delete and rebuild that Release and tag.
- Old public Release tags must remain stable unless the user explicitly asks otherwise.
- Internal tags may accumulate but must not be treated as public user install targets.
- Before a Release, the development machine must be synchronized with the current `main` commit.
- Release assets must be checked for all expected platforms.

## 6. Release note rules

- GitHub Release already has a title. The Release body must not repeat a title line such as `WeClaw Dev v0.1.7-alpha`.
- Start the body from sections such as `Highlights`, `Changes`, `Fixes`, `Install`, or `Validation`.
- If only the Release note body is wrong, edit the Release body only. Do not rebuild tags or assets.
- Release notes must describe user-visible behavior and validation status.
- Avoid implementation-only details unless they explain a user-visible change or a known operational risk.

## 7. VM GitHub Release download through host proxy

- Root cause pattern: the VM can reach `github.com`, `api.github.com`, `codeload.github.com`, and jsDelivr, but GitHub Release assets redirect to `release-assets.githubusercontent.com`, which may time out from the VM network.
- `weclaw upgrade` is a Go HTTP client. It reads `HTTP_PROXY`, `HTTPS_PROXY`, `http_proxy`, and `https_proxy`; it does not read `git config http.*.proxy`.
- In the verified VMware NAT setup, the Windows host is reachable at `192.168.231.1` and the working proxy is `http://192.168.231.1:7892`.
- A stale VM Git proxy such as `192.168.231.1:7896` is insufficient and can be misleading, especially because it affects Git only and not `weclaw upgrade`.
- Preferred VM fix: persist a user-level `~/.weclaw/proxy.env`, source it from `~/.profile` and `~/.bashrc`, and update Git proxy settings to the same reachable host proxy.
- Verification command after persistence: `weclaw upgrade` should run without explicit proxy variables and report either `Already up to date` or upgrade to the current public Release.

## 8. Documentation maintenance rules

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

## 9. Recent major-version context

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

## 10. Lessons learned

- Directly overwriting a running binary can trigger `Text file busy`. Use stop, build, install to a new path, then atomic replacement.
- Codex ACP thread identity is not enough for a new app-server process. Real resume requires `thread/resume`.
- `thread/resume.excludeTurns` requires experimental capabilities and must not be used in the stable default path.
- `turn` ID and session/thread ID are different concepts. UI text should say `last turn id` when displaying a turn ID.
- Same public tag with a different commit can happen during alpha rebuilds. Upgrade logic must compare build metadata, not only public version strings.
- Release note cleanup normally does not require rebuilding assets.
- Raw ACP stdout logging is useful for diagnosis but should stay off by default.
- A stop command must be explicit: report what was stopped, clear stale PID state, and verify that follow-up `start` will not immediately see the same managed process.

## 11. Next likely work

- After p0.1.5a39, internal tag hygiene is complete: use normalized `p*.*.*` internal tags, CI no longer auto-publishes branch/tag pre-releases, and public Releases must go through the manual `release.yml` workflow. GitHub Release `targetCommitish` may display `main`, so the tag target is the source of truth for Release commit verification. All pre-normalization internal `v`-prefixed refs have been removed after verified normalized mirror coverage.
- Keep README structure user-oriented.
- Keep WeChat ClawBot reply formatting Markdown-first. Plain-text conversion is only for internal classification or conservative fallback logic, not the primary outbound path.
- p0.1.5a42 formats WeChat slash command replies as Markdown cards, tables, headings and context-window progress bars.
- Keep this handoff concise.
- Continue moving detailed history into `docs/development-log.md`.
- If dsproxy later exposes token attribution, WeClaw should consume it in `/status` and degrade gracefully when the endpoint is absent.
- p0.1.5a43 treats Markdown syntax as visual affordances: blockquotes may be callouts, fenced text may be status panels, tables are reserved for compact structured data, and lists are preferred for mobile help.
- p0.1.5a45 keeps WeChat slash command replies English, compact and Markdown-first; `/info` is hidden compatibility and `/status` must not invent session cost without a reliable data source.
- p0.1.5a46 keeps `/balance` as a fixed-width Total-only panel, shows current account balance on `/status` without fabricating session cost, and makes `/info` show runtime versions and uptime.
- p0.1.5a47 removes the `/status` agent-type `@` marker, restores the context bar width, and renders CNY balance compactly as `￥...` on the Cost row.
- Avoid long Python heredoc patchers for small follow-up fixes. Prefer short shell commands, focused one-line scripts, or first request exact source snippets when a patch target is uncertain.
- p0.1.5a49 adds dynamic Markdown fence safety: generated outer fences must be longer than any nested backtick run, and chunk splitting must not close a longer outer fence on a shorter nested fence.
- p0.1.5a50 adds opt-in outbound Markdown capture via `WECLAW_CAPTURE_OUTBOUND_MARKDOWN_DIR`; use it to compare the exact `TextItem.Text` sent to ClawBot against the rendered WeChat result.

## Local runtime rebuild version-metadata rule

When replacing the real local `weclaw` runtime from a source checkout, do not install a plain `go build` artifact. A plain local build reports `dev | unknown`, which makes runtime diagnosis and handoff state ambiguous.

For local runtime replacement, the build must inject version metadata with these ldflags symbols:

```text
github.com/fastclaw-ai/weclaw/cmd.Version
github.com/fastclaw-ai/weclaw/cmd.PublicCommit
github.com/fastclaw-ai/weclaw/cmd.InternalVersion
github.com/fastclaw-ai/weclaw/cmd.InternalCommit
```

The expected local-development version shape is:

```text
weclaw public version: v0.1.8-alpha | 05cb93c
weclaw internal version: p0.1.5a56-mainline-tracker-audit-policy | <current-commit>
```

Before replacing `/usr/local/bin/weclaw` or any other real runtime binary, the generated candidate binary must be checked with `weclaw version`. After replacement, the installed binary must be checked again. A result containing `dev | unknown` is a failed installation, even if the binary itself runs.

Keep the public Release tag and commit separate from the internal development tag and commit. A local development build may intentionally show the latest public Release on the public line and the current internal tag on the internal line.

## Cross-project profile ownership boundary

WeClaw must treat CoDeepSeedeX / `dsproxy` as the authority for Codex profile files and DeepSeek runtime configuration. WeClaw may express user intent, such as `/effort max`, but it should not directly edit `~/.codex/config.toml` to compensate for a `dsproxy` profile-write bug.

The 2026-05-15 effort debugging found this concrete failure mode:

```text
dsproxy config set-effort max
```

In the affected CoDeepSeedeX build, that command wrote the Codex profile field as:

```toml
model_reasoning_effort = "max"
```

Codex rejects that value while parsing the whole `~/.codex/config.toml` file. Codex accepts `none`, `minimal`, `low`, `medium`, `high`, and `xhigh`, while DeepSeek-facing effort semantics are `high` and `max`. Therefore the correct ownership model is:

```text
WeClaw user intent: /effort max
dsproxy DeepSeek/env state: DEEPSEEK_REASONING_EFFORT=max
dsproxy Codex profile state: model_reasoning_effort="xhigh"
```

Do not broaden WeClaw into a generic Codex profile repair layer. If a profile-bound value is wrong, fix the CoDeepSeedeX contract and then update WeClaw to consume that contract. This applies to effort, model, profile status, context-window metadata, token telemetry, cost, pricing, balance, and compaction status.

For future WeClaw work:

- do not parse `~/.codex/config.toml` as the source of truth when `dsproxy` can expose a structured contract
- do not maintain model pricing or balance logic in WeClaw
- do not infer user/tool/environment/history token categories inside WeClaw
- do not read `.debug/` reports as a stable public API
- request a machine-readable `dsproxy` CLI or HTTP JSON interface, then format that result for WeChat
- keep WeClaw responsible for messaging, routing, session UX, and Markdown presentation only
