# WeClaw Dev Development Log

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
