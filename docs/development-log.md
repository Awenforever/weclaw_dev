# WeClaw Dev Development Log

## 2026-05-17 - p0.1.5a65-status-round3-compact-consolidation

- Scope: consolidate the temporary a64 debug rendering into a single strengthened compact `/status`.
- Change: removed `/status verbose` and `/status debug` aliases.
- Change: ordinary `/status` now folds compact round3 summaries into the panel: diagnostics counts, token attribution availability, pricing refresh/source state, and semantic compaction readiness.
- Boundary: WeClaw still does not fabricate user/tool/env/history token splits, does not derive context usage from session totals, and does not enable semantic payload compaction.
- Validation: requires `gofmt`, `git diff --check`, `bash -n install.sh`, focused messaging tests, broader package tests, and full `go test ./...`.

## 2026-05-17 - p0.1.5a64-status-debug-diagnostics

- Scope: WeClaw-side optional diagnostic rendering for the CoDeepSeedeX `p2.10a59-weclaw-round3-token-attribution-plan` contract.
- Change: added `/status verbose` and `/status debug` as explicit diagnostic aliases while preserving ordinary compact `/status` output.
- Change: verbose status now consumes dsproxy `--weclaw-json` fields for `diagnostics.degraded_fields`, context used-token unavailable reason/action, token attribution boundaries, pricing refresh/source state, and semantic compaction readiness.
- Boundary: WeClaw still does not parse Codex profile files, does not derive context usage from session totals, does not fabricate user/tool/env/history token splits, and does not enable semantic payload compaction.
- Validation: requires `gofmt`, `git diff --check`, `bash -n install.sh`, focused messaging tests, broader package tests, and full `go test ./...`.

## 2026-05-17 - p0.1.5a62-second-round-contract-acceptance-audit

- Scope: documentation-only WeClaw-side acceptance audit for the original second-round CoDeepSeedeX profile and telemetry contract request.
- Change: recorded the A1-A10 second-round acceptance matrix in both developer handbooks.
- Change: clarified that `p2.10a55-weclaw-runtime-status-contract` is a stage-accepted baseline for current WeClaw operation, not a complete closure of every original second-round requirement.
- Change: documented `/model` and `/balance` as partial boundaries. `/model` may update WeClaw local runtime/config fallback after calling dsproxy, but dsproxy `effective_model` remains authoritative. `/balance` still consumes legacy `dsproxy balance` JSON and should later consider the richer `--weclaw-json` balance diagnostic fields.
- Change: documented third-round CoDeepSeedeX candidates: real `context_window.used_tokens`, prompt-subcategory attribution, official pricing refresh (`pricing.refresh.available=false` / `official_live_pricing_refresh_not_implemented`), model catalog binding, semantic payload compaction readiness, and debug/verbose diagnostics.
- Validation: docs-only branch should pass `git diff --check`, `bash -n install.sh`, focused tests, and full `go test ./...`.
- Notes or lessons: second-round acceptance must be matrix-based. A correct compact `/status` screenshot proves only one output surface, not the whole cross-project ownership and telemetry contract.

## 2026-05-17 - p0.1.5a61-dsproxy-runtime-status-followup

- Scope: WeClaw follow-up integration against CoDeepSeedeX `p2.10a55-weclaw-runtime-status-contract`.
- Change: `/status` no longer uses `tokens.session_total` as token-level Context used tokens. It displays an unavailable marker when `context_window.used_tokens_available=false`.
- Change: `/status` now reads dsproxy `summary.total_tokens` for `tokens.last_turn`, `tokens.session_total`, and `tokens.auxiliary_model_calls`.
- Change: small estimated costs keep enough decimal precision to avoid displaying nonzero usage as `$0`.
- Change: developer handbooks were synchronized with the a61 internal line and the development log H1 was restored to the first line.
- Validation: `gofmt`, `git diff --check`, `bash -n install.sh`, focused messaging tests, broader package tests, and full `go test ./...` are required before merge.
- Notes or lessons: usage ledger totals and context-window used tokens are different metrics. WeClaw must only display context used tokens when dsproxy explicitly marks them available.

## 2026-05-17 - v0.1.9-alpha / p0.1.5a60-release-v0.1.9-alpha

- Scope: public pre-release for the WeClaw / CoDeepSeedeX telemetry integration line.
- Release target: `v0.1.9-alpha`.
- Required companion version: CoDeepSeedeX `v0.3.9-alpha` or newer when CoDeepSeedeX integration is used.
- Change: `/effort max` now uses the authoritative `dsproxy profile set-effort <profile> max --json` path and no longer directly edits Codex profile files from WeClaw.
- Change: `/status` now consumes `dsproxy status <route> --weclaw-json` and displays model, effort, token context, token usage, estimated cost, provider balance, runtime Compact/Trim and paths from the dsproxy contract.
- Change: compact `/status` hides internal model-conflict and missing-reason diagnostics while preserving a single-line `Paths` row for active debugging.
- Change: developer handbooks now record the release state, telemetry contract state, and CoDeepSeedeX version requirement.
- Validation: release readiness audit confirmed `main=origin/main=a02b3c9`, clean worktree, target `v0.1.9-alpha` absent before publication, and CoDeepSeedeX runtime `v0.3.9-alpha` telemetry fields available.
- Notes or lessons: Release notes must highlight the CoDeepSeedeX minimum version because older dsproxy builds lack the complete WeClaw telemetry contract.

## 2026-05-16 - p0.1.5a59-status-paths-restore

- Scope: compact `/status` path visibility after p0.1.5a58 display polish.
- Change: restored a single-line paths row: `Paths    cfg ~/.weclaw/config.json · log ~/.weclaw/weclaw.log`.
- Change: kept the code-block status layout, Context progress bar behavior, and natural-language output compliance behavior unchanged.
- Validation: `gofmt`, `git diff --check`, `bash -n install.sh`, focused package tests, and full `go test ./...` are required before merge.
- Notes or lessons: compact `/status` should hide internal dsproxy diagnostic reasons but keep the user-useful config/log paths during active debugging.

## 2026-05-16 - p0.1.5a58-status-telemetry-polish

- Scope: compact `/status` display quality after the first dsproxy telemetry integration.
- Change: hide context source, model-conflict diagnostic details, local path rows, cost missing reasons, and balance missing reasons from compact `/status`.
- Change: show token-level context as unavailable instead of `0/limit` when dsproxy has no session token usage.
- Change: show runtime compaction and trimming as separate char-level progress bars.
- Change: keep token-level context and char-level compaction/trimming visually separated.
- Validation: `gofmt`, `git diff --check`, `bash -n install.sh`, focused package tests, and full `go test ./...` are required before merge.
- Notes or lessons: missing dsproxy token/cost/balance data must degrade as `n/a`; WeClaw must not fabricate usage, cost, or balance values.

## 2026-05-16 - p0.1.5a57-dsproxy-telemetry-contract

- Scope: first WeClaw integration against the dsproxy full telemetry contract.
- Change: changed `/effort` to call `dsproxy profile set-effort <profile> <effort> --json` instead of the old `dsproxy config set-effort` path.
- Change: removed WeClaw-side Codex profile repair from the `/effort` path. WeClaw no longer directly edits `~/.codex/config.toml` for this runtime-control flow.
- Change: changed `/status` to prefer `dsproxy status <route> --weclaw-json` and render model, effort, token window, token buckets, estimated cost, balance and compaction from the dsproxy contract.
- Change: kept explicit fallback when the dsproxy contract is unavailable, without parsing Codex profile files as a source of truth.
- Validation: `gofmt`, `git diff --check`, `bash -n install.sh`, focused package tests, and full `go test ./...` are required before merge.
- Notes or lessons: WeClaw should format dsproxy-owned state, not duplicate dsproxy config ownership. Token-level context window and char-level compaction are displayed as separate lines.

## 2026-05-16 - p0.1.5a56a1-devlog-heading-order

- Scope: development-log structure cleanup after p0.1.5a56.
- Fix: restored the H1 title as the first line of `docs/development-log.md`.
- Change: updated the current active internal development line in both developer handbooks from `p0.1.5a56-mainline-tracker-audit-policy` to `p0.1.5a56a1-devlog-heading-order`.
- Validation: `git diff --check`, `bash -n install.sh`, focused package tests, and full `go test ./...` are required before merge.
- Notes or lessons: marker checks and tests can miss document structure mistakes. Development-log patches must verify that the file starts with the canonical H1.

## 2026-05-16 - p0.1.5a56-mainline-tracker-audit-policy

- Scope: documentation governance for the WeClaw and CoDeepSeedeX full telemetry integration line.
- Change: added a long-term mainline task tracker to both developer handbooks, including expected metrics, current version or source, current status, last maintained date, and notes.
- Change: recorded the audit rule that source and document modifications should be based on complete source files, full canonical documents, or complete function/module blocks. Grep and rg are navigation or verification aids, not sufficient patch evidence.
- Change: updated the active internal development line from `p0.1.5a55-cross-project-profile-boundary-docs` to `p0.1.5a56-mainline-tracker-audit-policy`.
- Validation: documentation marker checks, `git diff --check`, `bash -n install.sh`, focused package tests, and full `go test ./...` are required before merge.
- Notes or lessons: long-running cross-project work needs an explicit checklist to prevent task drift across conversations and inserted side tasks.
- Follow-up: implement the first WeClaw integration branch against CoDeepSeedeX `p2.10a48-weclaw-full-telemetry-contract`, then produce the next CoDeepSeedeX feedback prompt from actual implementation gaps.

## 2026-05-15 - p0.1.5a53-effort-profile-slash-polish

- Scope: WeChat slash command semantics and output polish, excluding context-window display changes.
- Fix: `/effort max` sends the DeepSeek-facing semantic value `max` to `dsproxy config set-effort`, while repairing the active Codex profile to the Codex-compatible `xhigh` value so later natural-language turns do not fail config parsing.
- Change: `/effort` replies no longer expose the internal Codex `xhigh` spelling to WeChat users.
- Change: `/profile deepseek` and `/profile deepseek-thinking` return one compact profile card instead of appending a second session card with repeated session and restart hints.
- Change: `/cwd` replies no longer duplicate the same data as both table rows and bullet rows.
- Validation: focused messaging tests and full `go test ./...` are required before merge.

## 2026-05-15 - p0.1.5a51-vm-proxy-docs

- Scope: VM GitHub Release asset download diagnostics and documentation.
- Finding: the VM could reach GitHub API and codeload directly, but Release asset downloads failed after redirecting to `release-assets.githubusercontent.com`.
- Root cause: `weclaw upgrade` uses Go HTTP proxy environment variables and does not read Git proxy settings. The VM had stale Git-only proxy settings pointing to `192.168.231.1:7896`, while the working Windows host proxy was `http://192.168.231.1:7892`.
- Change: documented the durable VM fix in the English and Chinese developer handbooks.

## 2026-05-15 - v0.1.8-alpha / p0.1.5a50

- Scope: public Release after WeChat ClawBot Markdown and slash-command polish.
- Release: `v0.1.8-alpha` was published at commit `05cb93c` with five uploaded assets.
- Internal marker: `p0.1.5a50-outbound-markdown-capture`.
- Highlights: Markdown-first rendering, compact English slash-command replies, dynamic Markdown fence safety, outbound Markdown capture, refreshed README screenshots, and Runtime settings screenshot wording.
- Validation: `git diff --check`, `bash -n install.sh`, and `go test ./... -count=1` passed before Release.

## 2026-05-15 - p0.1.5a50-outbound-markdown-capture

- Scope: outbound Markdown observability for ClawBot rendering diagnostics.
- Change: added opt-in `WECLAW_CAPTURE_OUTBOUND_MARKDOWN_DIR` capture of the exact Markdown text sent through `TextItem.Text` after `MarkdownForClawBot`.
- Purpose: distinguish WeClaw chunking or normalization bugs from ClawBot renderer limitations when testing nested code fences.
- Validation: focused outbound capture tests, messaging tests and full `go test ./...` are expected before merge.

## 2026-05-15 - p0.1.5a49-markdown-fence-safety

- Scope: Markdown fence safety for ClawBot rich rendering plus refreshed README screenshots.
- Change: added dynamic fenced-code delimiter selection so wrapper fences are always longer than the longest backtick run inside the content.
- Change: updated command-card fences, code-fence balancing, inline-code wrapping and Markdown block splitting so nested shorter fences do not close outer longer fences.
- Change: accepted refreshed README image assets and renamed the profile/balance screenshot label to Runtime settings.
- Validation: focused Markdown and messaging tests plus full `go test ./...` are expected before merge.

## 2026-05-15 - p0.1.5a48-docs-sync

- Scope: canonical documentation synchronization after slash command mobile polish.
- Change: synchronized README, README_CN, English handbook, Chinese handbook and development log with the p0.1.5a47 slash command behavior.
- Change: recorded the lesson that small follow-up fixes should avoid long Python heredoc patchers after repeated paste-truncation failures.
- Validation: documentation diff checks, shell syntax check and full test suite are expected before merge.

## 2026-05-15 - p0.1.5a47-slash-status-balance-spacing

- Scope: follow-up slash command visual polish after mobile testing.
- Change: `/status` now renders agent type without the `@` marker, restores the context bar length, and shows CNY balance as `￥...` with normal inline spacing on the Cost row.
- Validation: focused messaging tests, broader package tests and full `go test ./...` passed before merge.

## 2026-05-15 - p0.1.5a46-slash-field-polish

- Scope: follow-up WeChat slash command field polish after mobile testing.
- Change: `/balance` now uses a fixed-width fenced text panel with only Currency and Total, hiding Granted and Topped-up fields to avoid noisy account details and WeChat table misalignment.
- Change: `/status` renders agent type as inline code, shortens the context progress bar, and appends the current account balance summary to the cost line without pretending to know session cost.
- Change: `/info` now displays WeClaw and dsproxy public/internal runtime versions plus each process uptime.
- Validation: focused messaging tests, broader package tests and full `go test ./...` are expected before merge.

## 2026-05-14 - p0.1.5a45-english-compact-slash-renderer

- Scope: WeChat slash command output design.
- Change: made slash command replies English and compact across `/help`, `/status`, `/now`, `/balance`, `/model`, `/effort`, `/profile`, `/cancel`, `/restart`, `/info`, and unknown slash commands.
- Status: `/status` now uses a compact mobile panel for context, token usage, cost placeholder, proxy and paths; session cost remains `n/a` because no reliable session-cost data source is available yet.
- Balance: `/balance` uses a single table and hides empty Granted/Topped-up columns instead of duplicating account rows as bullet lines.
- Compatibility: `/info` remains available as a hidden compatibility reply but points users to `/status`.
- Validation: focused messaging tests, broader package tests and full `go test ./...` are expected before merge.

## 2026-05-14 - p0.1.5a43-slash-markdown-newlines-mobile-preview

- Scope: WeChat slash command Markdown rendering and runtime validation.
- Fix: replaced accidental literal `\n` joins with real newline joins so ClawBot can render headings, lists, fenced panels and tables.
- Change: treats Markdown syntax as visual affordances rather than literal semantic names; blockquotes can be callouts, fenced text can be status panels, tables are kept for compact structured data, and mobile help uses sections and lists.
- Runtime: local development binary should be rebuilt with p0.1.5a43 metadata and restarted without resume to avoid reusing stale invalid Codex threads.
- Validation: focused tests, full tests and active WeChat ClawBot preview messages are required.


## 2026-05-14 - p0.1.5a42-slash-command-markdown-formatting

- Scope: WeChat slash command output formatting.
- Change: formatted slash command replies as Markdown cards with headings, normalized lists, tables, and context-window progress bars.
- Details: improved `/help`, `/status`, `/balance`, `/now`, `/cancel`, `/model`, `/effort`, `/profile`, `/restart`, `/info`, `/cwd`, and unknown slash command presentation through the common command-card path.
- Validation: focused messaging tests and full `go test ./...` are expected before commit.
- Notes or lessons: ClawBot Markdown rendering was verified in the real WeChat ClawBot conversation before broadening command formatting.


## 2026-05-14 - p0.1.5a41-clawbot-markdown-first-formatting

- Scope: WeChat ClawBot output formatting.
- Change: switched outbound text from upstream-style Markdown-to-plain-text downgrade to Markdown-first normalization for ClawBot rich rendering.
- Details: preserved headings, lists, code fences, tables, blockquotes, links, inline code and emphasis in outbound text, while keeping the plain-text converter for internal classification and progress heuristics.
- Validation: focused messaging tests and full `go test ./...` are expected before commit.
- Notes or lessons: README wording inherited from upstream FastClaw/WeClaw must not be treated as the target behavior when WeClaw Dev is intentionally optimizing for ClawBot rich Markdown rendering.


## 2026-05-14 - p0.1.5a40-delete-legacy-v-internal-tags

- Deleted the remaining legacy `v0.1p...` internal tags after verifying every one had a normalized `p0.1.*` mirror tag at the same commit.
- Preserved public Release tags, GitHub Releases, normalized `p*.*.*` internal tags, and the `p0.1.0a*` archival tags.
- After this change, remote `v*` tags are reserved for public Release tags only.


## 2026-05-14 - p0.1.5a39-final-audit-polish

- Removed literal old alpha-work internal tag tokens from future-facing developer handbooks while keeping precise historical mappings in `docs/development-log.md`.
- Documented that GitHub Release `targetCommitish` may display `main`; Release commit verification should use the immutable Release tag target.
- Added guard coverage so future-facing documentation and CI workflows do not reintroduce alpha/beta pre-release noise tokens.


## 2026-05-14 - p0.1.5a38-special-legacy-tag-archive

- Created normalized archival mirror tags for the four special legacy `v0.1d/e/f...` tags, then deleted the old special legacy refs after verifying identical commit targets.
- `v0.1d-codex-model-provider-thread-fix` -> `p0.1.0a1-codex-model-provider-thread-fix`
- `v0.1e-slash-output-polish` -> `p0.1.0a2-slash-output-polish`
- `v0.1f-acp-raw-stdout-log` -> `p0.1.0a3-acp-raw-stdout-log`
- `v0.1f1-handoff-notes` -> `p0.1.0a4-handoff-notes`
- Preserved all public Release tags, all public GitHub Releases, all legacy `v0.1p...` historical internal tags, and all normalized `p0.1.*` internal tags.
- Temporarily disabled the GitHub Actions CI workflow while pushing archival tags to old commits, then re-enabled CI after the archive tag push.


## 2026-05-14 - p0.1.5a37-ci-prerelease-policy

- Disabled CI auto pre-release publishing. The CI workflow now keeps test/build/artifact behavior but no longer creates `beta-latest`, `alpha-p...`, or `alpha-work-p...` GitHub pre-releases.
- Deleted the remaining CI-generated `alpha-p0...`, `alpha-work-p0...`, and `beta-latest` GitHub pre-releases and matching remote tags.
- Preserved all public Release tags, all public GitHub Releases, all legacy `v0.1p...` historical internal tags, and all normalized `p0.1.*` internal tags.
- Left `v0.1d/e/f...` special legacy tags untouched for separate policy review.


## 2026-05-14 - p0.1.5a36-alpha-prerelease-cleanup

- Deleted the 13 legacy `alpha-work-v0.1p...` GitHub pre-releases and their matching remote tags after verifying each one had a normalized `p0.1.*` mirror tag at the same commit.
- Preserved all public Release tags, all GitHub public Releases, all legacy `v0.1p...` internal tags, and all normalized `p0.1.*` mirror tags.
- Left `alpha-p0...`, `alpha-work-p0...`, and `beta-latest` untouched for separate policy review.


## 2026-05-14 - p0.1.5a35-internal-tag-mirror-migration

- Created normalized mirror tags for the 42 auto-mappable legacy `v0.1p...` internal tags.
- Verified that each new `p0.1.*` mirror tag points to the same commit as its legacy `v0.1p...` source tag.
- Updated the developer handbooks so the current Release asset/code line can be referenced by normalized `p0.1.5a28-stop-semantics` while preserving the historical `v0.1p5a28-stop-semantics` reference.
- Preserved all old `v0.1p...` tags, all `alpha-work-v0.1p...` tags, all public Release tags, and all GitHub Releases. No old tag was deleted or moved.


## 2026-05-14 - p0.1.5a34-internal-tag-format-guard

- Replaced future-facing old internal tag examples in `README.md` and `cmd/update_test.go` with normalized `p*.*.*` examples.
- Added a focused guard test to reject legacy `v0.1p...` internal tag examples in current documentation and release workflow inputs while preserving historical/deprecated references in the development log and handoff context.
- Preserved all historical tags and GitHub Releases. No old tag was deleted or moved.


## 2026-05-14 - p0.1.5a33-internal-version-format-policy

- Established the normalized internal development tag format: `p<major>.<minor>.<patch>[aN[aM...]][-topic]`.
- Confirmed that internal tags must start with `p`, must not start with `v`, and must keep three numeric components before optional repair/subversion suffixes.
- Recorded the read-only audit result: local and remote internal-like tags are synchronized, with 42 legacy `v0.1p...` tags, 3 suspect `v0.1d/e/f...` legacy tags, and 13 remote `alpha-work-v0.1p...` pre-release tags that do not match the new internal tag format.
- Preserved all historical tags and GitHub Releases in this change. No old tag was deleted or moved.
- Updated `.github/workflows/release.yml` so the internal tag input example uses the normalized `p0.1.5a33-...` style.


## 2026-05-14 - v0.1p5a32-doc-current-state-sync

- Synchronized `docs/developer-handbook.md` and `docs/developer-handbook.zh-CN.md` with the actual current `main` line after p5a31.
- Preserved the public Release state: `v0.1.7-alpha` remains the Latest public Release and continues to point to the Release asset/code line at `31fa432`.
- Preserved the old public Release state: `v0.1.6-alpha` remains at `bbb2f28`.
- Clarified that p5a29, p5a30, p5a31, and p5a32 are documentation-only or handoff-synchronization commits and must not be confused with the Release asset commit.


This file keeps detailed, chronological, and reviewable development history. Keep entries uniform. Keep the AI startup context in `docs/developer-handbook.md`.

## Entry format

Each entry should use:

- Date
- Version or tag
- Commit
- Scope
- Change
- Validation
- Notes or lessons
- Follow-up

## 2026-05-14 · v0.1p5a31-doc-entry-final-fix

- Date: 2026-05-14
- Version or tag: `v0.1p5a31-doc-entry-final-fix`
- Commit: pending at patch time
- Scope: final documentation entry cleanup
- Change: removed the stale README link to the deleted `docs/DEVELOPMENT_MANUAL.md` file and updated the English/Chinese developer handbooks so their current state reflects p5a30 after merge.
- Validation: local link checks for all canonical markdown documents, stale removed-document checks, `git diff --check`, `bash -n install.sh`, and full `go test ./...`.
- Notes or lessons: deleting obsolete entry points must include a reference sweep across user-facing documents. A removed file must not remain as a README link.
- Follow-up: final read-only audit should pass with `run_ok=1` and no tracked or linked references to removed document entry points outside the development log history.

## 2026-05-14 · v0.1p5a30-doc-entry-cleanup

- Date: 2026-05-14
- Version or tag: `v0.1p5a30-doc-entry-cleanup`
- Commit: pending at patch time
- Scope: documentation entry cleanup
- Change: removed obsolete pointer/duplicate documents `HANDOFF.md`, `docs/HANDOFF.md`, and `docs/DEVELOPMENT_MANUAL.md`. Kept the canonical document set limited to `README.md`, `README_CN.md`, `docs/developer-handbook.md`, `docs/developer-handbook.zh-CN.md`, and `docs/development-log.md`.
- Validation: document inventory checks, stale-entry checks, `git diff --check`, `bash -n install.sh`, and full `go test ./...`.
- Notes or lessons: do not keep meaningless compatibility pointer documents just to satisfy old tests or ghost references. Tests and references must follow the canonical documentation layout.
- Follow-up: future document additions must justify their audience and maintenance owner before being added.

## 2026-05-14 · v0.1p5a29-release-note-devlog-sync

- Date: 2026-05-14
- Version or tag: `v0.1p5a29-release-note-devlog-sync`
- Commit: pending at patch time
- Scope: release note and development log synchronization
- Change: updated the `v0.1.7-alpha` GitHub Release body to include p5a28 stop semantics, commit `31fa432`, and VM stop-semantics validation. Updated this development log so p5a28 no longer has a pending commit marker.
- Validation: `git diff --check`, `bash -n install.sh`, GitHub Release body verification after edit, and release tag verification.
- Notes or lessons: if a Release tag is rebuilt after a feature-line extension, the Release body must be explicitly audited and updated. Asset rebuild alone is not enough.
- Follow-up: final read-only audit should verify that the Release body mentions `weclaw stop`, `Stopped PIDs`, `Verified: no managed weclaw process remains`, `p5a28`, and `31fa432`.
## 2026-05-14 · v0.1p5a28-stop-semantics

- Date: 2026-05-14
- Version or tag: `v0.1p5a28-stop-semantics`
- Commit: `31fa432`
- Scope: stop command semantics
- Change: made `weclaw stop` output explicit and verifiable, targeted every detected managed WeClaw process, included live WeClaw pid-file targets that do not match the managed foreground pattern, cleared stale PID state, and rechecked that no managed WeClaw process remains.
- Validation: focused command tests, `git diff --check`, `bash -n install.sh`, `go test ./cmd`, and full `go test ./...`.
- VM validation: after installing old `v0.1.6-alpha` in an isolated VM path and intentionally leaving a managed `start -f` process alive, the new `v0.1.7-alpha | 31fa432` asset stopped the old PID, printed `Stopped PIDs: ...`, verified `no managed weclaw process remains`, and then reported `weclaw was not running` on a second stop.
- Notes or lessons: stop semantics must be user-visible and confirmable. A generic `weclaw stopped` message is insufficient when `start` may later detect residual managed processes.
- Follow-up: keep future stop/start/status runtime-state changes covered by both unit tests and VM process-level validation.
## 2026-05-14 · v0.1p5a27-readme-cn-upgrade-polish

- Date: 2026-05-14
- Version or tag: `v0.1p5a27-readme-cn-upgrade-polish`
- Commit: pending at patch time
- Scope: Chinese README cleanup
- Change: removed the trailing duplicate `## 升级` section from `README_CN.md`, merged the v0.1.7-alpha runtime-migration note into `## 更新与卸载`, and corrected the `weclaw upgrade` code block so prose is no longer inside a bash fence.
- Validation: documentation structure checks, `git diff --check`, `bash -n install.sh`, and full `go test ./...`.
- Notes or lessons: bilingual README structure must stay aligned. Upgrade behavior notes should live in the update section, not after the license.
- Follow-up: keep future milestone behavior changes in the README behavior-change table.

## 2026-05-14 · v0.1p5a26-docs-restructure

- Date: 2026-05-14
- Version or tag: `v0.1p5a26-docs-restructure`
- Commit: pending at patch time
- Scope: documentation structure
- Change: reorganized README order, introduced user-facing behavior-change tables, split developer handoff from chronological development log, and added English and Chinese developer handbooks.
- Validation: documentation checks, `git diff --check`, shell syntax check for `install.sh`, and full Go tests are expected before commit.
- Notes or lessons: README is for users. Handoff is for AI and maintainer startup context. Long history belongs in this file.
- Follow-up: keep milestone behavior changes in README tables.

## 2026-05-14 · v0.1.7-alpha / p5a25

- Date: 2026-05-14
- Version or tag: `v0.1.7-alpha`, `v0.1p5a25-codex-resume-stable-api`
- Commit: `25d36b7`
- Scope: Codex resume stable API and final v0.1.7-alpha release state
- Change: removed default `excludeTurns` from `thread/resume`, used stable Codex resume behavior, and confirmed natural-language resume no longer fails with the experimental capability error.
- Validation: focused, broader, and full `go test ./...` passed. VM upgrade from p5a24 to p5a25 passed. WeChat `/now`, `/status`, natural language, and resumed thread reuse were validated.
- Notes or lessons: stable resume must not send `excludeTurns` by default.
- Follow-up: token attribution is a dsproxy-side requirement, not a WeClaw-side guess.

## 2026-05-14 · p5a24

- Date: 2026-05-14
- Version or tag: `v0.1p5a24-readme-upgrade-runtime-migration-note`
- Commit: `d25b79d`
- Scope: README upgrade behavior
- Change: documented that starting from `v0.1.7-alpha`, upgrade can preserve runtime state when recoverable.
- Validation: VM upgrade path later confirmed p5a24 to p5a25 migration.
- Notes or lessons: user-facing behavior changes must be documented where users can find them.
- Follow-up: README should keep such behavior changes in a table.

## 2026-05-14 · p5a23

- Date: 2026-05-14
- Version or tag: p5a23
- Commit: not recorded in current handoff
- Scope: upgrade runtime migration
- Change: fixed upgrade comparison so the same public tag rebuilt to a different commit can still be detected and upgraded.
- Validation: VM later confirmed same-tag different-commit upgrade from p5a23 to p5a24 and p5a24 to p5a25.
- Notes or lessons: public version alone is not enough during alpha rebuilds.
- Follow-up: keep public version and internal build metadata separate.

## 2026-05-14 · p5a22

- Date: 2026-05-14
- Version or tag: p5a22
- Commit: not recorded in current handoff
- Scope: `/now`, `/status`, and context usage consistency
- Change: aligned profile and session resolution between `/now` and `/status`, improved context-window display, and clarified turn ID versus session/thread ID semantics.
- Validation: WeChat-side `/now` and `/status` checks passed after p5a25.
- Notes or lessons: UI labels must not imply that a turn ID is the session ID.
- Follow-up: when dsproxy token attribution exists, display input attribution in `/status`.

## 2026-05-14 · p5a21

- Date: 2026-05-14
- Version or tag: p5a21
- Commit: not recorded in current handoff
- Scope: Codex resume and start idempotence
- Change: introduced real Codex `thread/resume` and made managed `weclaw start` idempotent when an existing process is already running.
- Validation: subsequent p5a25 validation confirmed the corrected stable resume path.
- Notes or lessons: resume requests must be applied before `turn/start`, but unsupported experimental fields must not be sent by default.
- Follow-up: keep start behavior explicit when a managed process already exists.

## 2026-05-14 · p5a20 to v0.1.7-alpha preparation

- Date: 2026-05-14
- Version or tag: p5a20 and v0.1.7-alpha preparation
- Commit: not recorded in current handoff
- Scope: v0.1.7-alpha release line
- Change: prepared the release line covering default background logs, session resume defaults, start idempotence, Codex resume correction, status consistency, and upgrade migration.
- Validation: final validation occurred at p5a25.
- Notes or lessons: Release notes should cover the whole release line, not only the final patch.
- Follow-up: Release note body can be edited manually without rebuilding assets.

## Historical note · ACP raw stdout logging

- Date: 2026-05-07
- Version or tag: `v0.1f-acp-raw-stdout-log`
- Commit: not recorded in current handoff
- Scope: diagnostic logging
- Change: added optional `WECLAW_ACP_RAW_LOG=1` support to inspect raw Codex app-server NDJSON.
- Validation: raw logs confirmed DeepSeek through Codex app-server returns the same general event structure as native Codex.
- Notes or lessons: raw stdout logging is diagnostic only and should stay off during normal use.
- Follow-up: if WeChat output is compressed into one line, first inspect raw ACP events before changing formatting logic.

## Historical note · running binary replacement

- Date: 2026-05-07
- Version or tag: legacy operational lesson
- Commit: not recorded in current handoff
- Scope: installation and binary replacement
- Change: documented that direct overwrite of a running `/usr/local/bin/weclaw` can fail with `Text file busy`.
- Validation: operational issue reproduced historically.
- Notes or lessons: stop managed process, build new binary, install to a new path, then atomically replace.
- Follow-up: avoid direct copy-overwrite of active binaries.
