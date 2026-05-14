# WeClaw Dev Development Log

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

## 2026-05-14 · v0.1p5a28-stop-semantics

- Date: 2026-05-14
- Version or tag: `v0.1p5a28-stop-semantics`
- Commit: pending at patch time
- Scope: stop command semantics
- Change: made `weclaw stop` output explicit and verifiable, targeted every detected managed WeClaw process, included live WeClaw pid-file targets that do not match the managed foreground pattern, cleared stale PID state, and rechecked that no managed WeClaw process remains.
- Validation: focused command tests, `git diff --check`, `bash -n install.sh`, and full `go test ./...`.
- Notes or lessons: stop semantics must be user-visible and confirmable. A generic `weclaw stopped` message is insufficient when `start` may later detect residual managed processes.
- Follow-up: VM runtime validation should confirm that repeated `weclaw stop && weclaw start deepseek-thinking` behaves predictably.

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
