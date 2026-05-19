# WeClaw Dev Developer Handbook

## p0.1.5a94 CoDeepSeedeX p2.10a79 Details origin breakdown

p0.1.5a94 consumes `tokens.prompt_reconciliation.details_origin_breakdown` from CoDeepSeedeX p2.10a79.

Rules:

- Details reads origin components before legacy `prompt_subcategory_split`.
- The new Details line is a token-origin breakdown, not a classified-total reconciliation.
- WeClaw must not display `classified~`, `partial~`, or `covered~` when `details_origin_breakdown` is available.
- `tools` is `tool_output + tools_schema`.
- `env` is `environment + runtime_injected + other_prompt`.
- `provider_residual` is never merged into `other_prompt`; it is displayed as `resid~...` only when non-zero and not marked within tolerance.

## p0.1.5a93 CoDeepSeedeX p2.10a76 aux and coverage contract

p0.1.5a93 consumes the CoDeepSeedeX p2.10a76 token coverage contract.

Rules:

- `Tokens aux` displays `0` only when `tokens.auxiliary_model_calls.available=true` and the auxiliary token ledger is current-session scoped.
- Route/profile auxiliary totals must not be displayed as current-session `aux`.
- `Details` appends `partial~<categories_sum>/<provider_reference>` when dsproxy reports incomplete prompt-subcategory coverage.
- `Details` appends `covered~<categories_sum>/<provider_reference>` when dsproxy reports complete prompt-subcategory coverage.
- WeClaw does not assign `delta_tokens` to `other`; the coverage suffix uses dsproxy-provided coverage fields directly.

## p0.1.5a92 CoDeepSeedeX p2.10a75 contract alignment

p0.1.5a92 aligns WeClaw `/status` with the CoDeepSeedeX p2.10a75 contract.

Rules:

- `Cost session` and trailing `Cost total` are displayed only for current-session scoped cost.
- A nested `cost.session.available=true` object is not enough by itself; `cost.scope`, `cost.ledger_scope`, or `cost.session.scope` must indicate `current_session`.
- `Details` is displayed only when `tokens.prompt_subcategory_split.available=true`, `scope=current_session`, and the split session id is compatible with `tokens.session.session_id` when both are present.
- Compact/Trim main progress consumes display/retention/progress fields as information-retention values, not capacity-trigger progress.
- Capacity/trigger progress remains a separate dsproxy concern and is not used for the main WeClaw Compact/Trim row.

## p0.1.5a91 new-session active state

p0.1.5a91 fixes the `/new` active-session state path.

Rules:

- `/new` must clear any pending startup resume for the current profile after a new session/thread is created.
- `/new` must ensure the active agent reports the new session id immediately.
- The next `/status` after `/new` must use the new session id, not the previous resumed thread.
- Runtime-state persistence records the new session id after reset.
- This patch does not implement current-session cost or Compact/Trim information-retention semantics; those remain blocked on the dsproxy contract update.

## p0.1.5a90 new-session status fallback scope

p0.1.5a90 narrows the p89 route fallback scope.

Rules:

- Session-filtered `/status` remains the source of truth for Context, Details, Tokens, Cost, Compact, and Trim.
- No-session route fallback is allowed only for route-level metadata such as Pricing, Balance, Proxy, Paths, health, diagnostics, and a sanitized Policy subset.
- `/new` followed by `/status` must not display the previous session's Context, Details, Compact, or Trim.
- WeClaw must still not synthesize current-session cost or reinterpret Compact/Trim retention semantics. Those remain blocked on the dsproxy contract update.

## p0.1.5a89 dsproxy route startup and safe status fallback

p0.1.5a89 fixes the runtime failure mode found after p88:

- Starting `deepseek` or `deepseek-thinking` ensures the matching dsproxy route first.
- `/status` first queries dsproxy with the active session id.
- If that session-scoped status is degraded before the first model request, WeClaw may fallback to the no-session route status only for non-session fields: Context, Pricing, Balance, Policy, Compact/Trim, proxy/paths diagnostics.
- WeClaw must never fallback `Tokens session`, `Cost session`, or trailing `Cost total` from route/profile/global totals.
- If current-session cost remains unavailable after a real request, that is a dsproxy contract gap, not a WeClaw display-layer calculation target.

## p0.1.5a88 CoDeepSeedeX p2.10a74 status contract adaptation

p0.1.5a88 adapts WeClaw `/status` to the CoDeepSeedeX p2.10a73/p2.10a74 contract.

Rules:

- Pass the active ACP/Codex session id to `dsproxy status ... --weclaw-json --session-id <id>` when available.
- Use `tokens.latest_primary_turn` for the displayed `last` token field; auxiliary calls must not replace the primary request basis.
- Use `tokens.session` only when `available=true`; do not label profile/route lifetime totals as current-session totals.
- Display `Cost session` and trailing `total` only when dsproxy marks the cost ledger as current-session scoped.
- Use `pricing.prices_display` first, then `pricing.effective_prices`, and only then legacy `pricing.prices`.
- Use `runtime_payload_guard.*.progress_numerator_chars`, `progress_denominator_chars`, and `progress_ratio` for Compact/Trim progress.
- Keep WeClaw display-only: no local pricing lookup, no currency conversion, no retokenization, no debug-file reads, and no recomputing session cost from current model prices.

## Current trusted state after v0.1.9-alpha Latest closeout

Runtime release state:

- Public Release: `v0.1.9-alpha`
- Public Release commit: `82ba8ca`
- Runtime internal version for the public asset: `p0.1.5a86-cumulative-release-notes | 82ba8ca`
- GitHub Release state: `draft=false`, `prerelease=false`, Latest
- Release assets: `weclaw_linux_amd64`, `weclaw_linux_arm64`, `weclaw_darwin_amd64`, `weclaw_darwin_arm64`, `weclaw_windows_amd64.exe`
- `main` and `origin/main` at the time of Release update: `82ba8ca`
- p87 is docs-only and does not move the public Release tag.

VM validation state:

- Latest Release API returned `v0.1.9-alpha` with `prerelease=false`.
- Standard install path from `main/install.sh` resolved to `v0.1.9-alpha`.
- Existing same-version install short-circuited correctly.
- `weclaw upgrade` returned `Already up to date (v0.1.9-alpha)`.
- `weclaw start deepseek-thinking resume` started the runtime successfully.
- The earlier downgrade notice from `v0.1.9-alpha` to `v0.1.8-alpha` was not reproduced after marking `v0.1.9-alpha` as Latest. If it reappears in a future VM/user report, reopen as a source-level update-notice bug.

Closed `/status` line:

- Original `█░` progress bars retained.
- Details row does not display the local-estimate suffix.
- Policy target includes the `chars` unit.
- Cost, Pricing, and Balance display RMB/CNY with `￥`.
- Cost displays `session`, `last`, `aux`, and trailing `total`.
- Pricing displays CNY per-million-token values from dsproxy.
- WeClaw remains a display consumer of dsproxy structured telemetry and must not query prices, convert currencies, split reasoning cost, retokenize prompts, read debug files, or recompute session cost from the current model.

Mainline status:

- The v0.1.9-alpha release line is closed.
- Do not continue speculative WeClaw patching unless a new concrete requirement arrives.

## p0.1.5a86 cumulative v0.1.9-alpha release notes

p0.1.5a86 corrects the release-note process after p85. The v0.1.9-alpha notes must be cumulative from v0.1.8-alpha, not a narrow draft for only the last patch. The release body must be updated from the existing GitHub Release body and must cover all user-visible changes since v0.1.8-alpha, including the a72-a79 telemetry/release work and the a80-a84 Details/Cost/Pricing refinements.

The generated cumulative Release notes are stored at the path recorded in the script log. Public tag `v0.1.9-alpha` is intentionally not moved in p86; the actual Release update remains a separate explicit step.

## p0.1.5a85 docs and v0.1.9-alpha release note preparation

Trusted state before public pre-release update:

- main/origin/main target before p85: `6ebef06`
- latest internal implementation tag: `p0.1.5a84-cny-cost-pricing-total`
- current public pre-release tag before release update: `v0.1.9-alpha` -> `f8ef8b0`
- p84 local runtime validation: `/status` displays CNY Cost/Pricing/Balance rows, original block progress bars, no Details local-estimate suffix, and Policy target with `chars`.

User-facing v0.1.9-alpha release note scope:

- compact WeChat `/status` display
- original `█░` progress bars retained after WeChat rendering review
- Details row cleanup
- Policy target unit fix
- CNY/RMB Cost, Pricing, and Balance display
- Cost `session`, `last`, `aux`, trailing `total`
- dsproxy structured telemetry boundary

Compatibility note:

- Requires CoDeepSeedeX v0.3.9-alpha or newer.
- CNY Cost/Pricing rows require the dsproxy CNY pricing telemetry contract from `p2.10a70-pricing-cny-primary-source` or later.

## p0.1.5a84 CNY Cost and Pricing display

p0.1.5a84 consumes dsproxy structured CNY pricing and cost fields. `/status` displays `Cost     session~￥...  last~￥...  aux~￥...  total~￥...`, with `total` shown last as the user-facing summary label for dsproxy's total estimated spend field. `Pricing` uses CNY per-million-token values from dsproxy, and `Balance` renders CNY balances as `￥...`.

WeClaw must not query prices, convert currencies, split reasoning cost, or recompute session cost from the current model. It only formats dsproxy structured fields.

## p0.1.5a83 original progress and policy target unit

p0.1.5a83 keeps the original block progress style `█░` as the active `/status` progress bar. The right-endcap line style remains in code as a candidate formatter, but it is not active. The Policy row now includes the unit for the target value: `target 750k chars`.

## p0.1.5a82 right-endcap progress bar trial

p0.1.5a82 switches the active `/status` progress bar style from the original block style `█░` to a right-endcap line style such as `━╸──────────────────`, while keeping the original block formatter in code as `formatCommandProgressBarOriginal` for quick rollback after real WeChat rendering review.

This is a display-only trial. It must not change Context, Compact, Trim, token, cost, pricing, or dsproxy telemetry semantics.

## p0.1.5a80 status Details and Cost format

p0.1.5a80 adds a WeChat `/status` `Details` row that consumes only `tokens.prompt_subcategory_split.categories` from `dsproxy status <route> --weclaw-json`. WeClaw must not tokenize locally, derive estimates from character counts, read debug files, or treat session totals as context-window usage.

Display rules:
- when `tokens.prompt_subcategory_split.available=true`, show `Details  user~...  hist~...  tool~...  sys~...  dev~...  comp~...  other~...`
- when `tokens.prompt_subcategory_split.reason=profile_tokenizer_available_but_no_observed_prompt`, show `Details  n/a · waiting first prompt`
- when `tokens.profile_tokenizer.available=false`, show `Details  n/a · tokenizer unavailable`

The cost row label is now `Cost`, with compact `label~value` fields: `Cost     session~...  last~...  aux~...`. Provider usage remains authoritative for `last`, `session`, `aux`, and cost. Compact/Trim remain char-level runtime payload guards.

## p0.1.5a78 raw fixed-tag pre-release installer

p0.1.5a78 changes the published pre-release install entry from jsDelivr fixed-tag URLs to raw GitHub fixed-tag URLs because VM testing showed that jsDelivr can continue serving stale `@v0.1.9-alpha/install.sh` content even after a successful purge.

Use this for pre-release install notes:

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

Keep jsDelivr for stable/latest `@main` convenience entrypoints, but do not use jsDelivr fixed-tag URLs as the authoritative pre-release installer path after a moved public pre-release tag.


## p0.1.5a77 pinned pre-release installer

p0.1.5a77 fixes the user-facing pre-release install path. `install.sh` intentionally defaults to the stable/latest GitHub Release when no `--version` is provided. Therefore pre-release Release notes and docs must use a pinned command:

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

Do not publish pre-release notes that pipe a tagged installer without passing `--version`; that installs the stable/latest Release instead of the pre-release asset.


## p0.1.5a76 v0.1.9-alpha release refresh

p0.1.5a76 refreshes documentation and republishes the current `v0.1.9-alpha` pre-release from the latest mainline after the a72-a75 `/status` telemetry work. The release includes Context, Pricing, Compact, Trim and Policy display improvements, and keeps the public release note focused on user-visible changes. Release notes must not duplicate the GitHub Release title line.


This is the canonical English handoff for starting a new AI development conversation. Keep it concise and current. Long chronological details belong in `docs/development-log.md`.

## 1. Current trusted state

- Project path: `~/projects/weclaw-streaming`
- GitHub repository: `Awenforever/weclaw_dev`
- Main branch: `main`
- Current Latest public Release: `v0.1.8-alpha`
- Current Latest public Release commit: `05cb93c`
- Current public pre-release: `v0.1.9-alpha` at `<to-be-refreshed-by-p79>`.
- Current Release internal marker: `p0.1.5a69-release-v0.1.9-alpha-refresh` at `6a5f10f`
- Current internal development tag: `p0.1.5a80-status-details-cost-format`
- Last audited handoff baseline before this sync: `main=origin/main=p0.1.5a59-status-paths-restore=68ca2bb`
- Current active development line: `p0.1.5a79-prerelease-install-command-curl-prefix`
- Previous public Release `v0.1.7-alpha` remains at `31fa432` and must not be moved.
- Target `v0.1.9-alpha` GitHub Release title is `WeClaw Dev v0.1.9-alpha`, must be created as a pre-release, and must require CoDeepSeedeX `v0.3.9-alpha` or newer when CoDeepSeedeX integration is used.
- `v0.1.8-alpha` GitHub Release title is `WeClaw Dev v0.1.8-alpha`, is not draft, is not prerelease, and has five uploaded assets.
- Expected Release assets: Linux amd64, Linux arm64, Darwin amd64, Darwin arm64, and Windows amd64.
- p0.1.5a51 only documented the VM GitHub Release asset download failure and host-proxy fix. It did not move the public Release.
- Start every future task with a read-only audit of `main`, `origin/main`, the active internal tag, public Release tags, clean worktree state, and the handbook/log heads.

## 2. Long-term mainline task tracker

Update this table whenever a cross-conversation task changes scope, status, expected metrics, or ownership. Treat it as the first progress checkpoint for any new development conversation.

Status vocabulary: `planned`, `in_progress`, `verified`, `blocked`, `done`, `superseded`.

| Mainline item | Expected metric or acceptance condition | Current version or source | Current status | Last maintained | Notes |
| --- | --- | --- | --- | --- | --- |
| Full telemetry contract baseline | Local `dsproxy` exposes profile status and WeClaw status JSON for `deepseek` and `deepseek-thinking`, including `model`, `effort`, `context_window`, `tokens`, `pricing`, `cost`, `balance`, and `compaction`. | CoDeepSeedeX `v0.3.9-alpha` / `p2.10a55-weclaw-runtime-status-contract`; WeClaw `p0.1.5a61-dsproxy-runtime-status-followup` | verified | 2026-05-17 | WeClaw now reads dsproxy `summary.total_tokens`, preserves estimated cost display, and does not treat session_total as context used tokens. |
| WeClaw ownership boundary | WeClaw does not directly edit Codex profile files for normal `/effort`, `/model`, `/status`, or telemetry paths when `dsproxy` provides a structured contract. | WeClaw `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | Profile repair logic is removed from the `/effort` path in `p0.1.5a57-dsproxy-telemetry-contract`. |
| `/effort` integration | `/effort max` calls the authoritative `dsproxy profile set-effort <profile> max --json` contract and displays `effort.user_facing` or `effort.deepseek_reasoning_effort`. | WeClaw `p0.1.5a57-dsproxy-telemetry-contract` | verified | 2026-05-16 | WeClaw no longer edits Codex profile files in this path. |
| `/status` contract integration | `/status` consumes `dsproxy status <route> --weclaw-json` and renders returned data with explicit fallback for unavailable fields. | WeClaw `p0.1.5a61-dsproxy-runtime-status-followup` plus CoDeepSeedeX `p2.10a55-weclaw-runtime-status-contract` | verified | 2026-05-17 | Status output reads available usage/cost/balance fields, keeps token-level Context separate from usage ledger totals, and restores a single-line Paths row. |
| Telemetry display quality | Mobile WeChat output remains compact and Markdown-first while showing model, effort, context window, token usage, estimated cost, balance, compaction, proxy, and a single-line paths row. | WeClaw `p0.1.5a59-status-paths-restore` | verified | 2026-05-17 | Compact `/status` hides internal diagnostics but retains user-useful runtime paths. |
| Evidence-first audit discipline | Source and document changes are based on full source files, full canonical documents, or complete function/module blocks rather than isolated grep snippets. | `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | Grep/rg may help locate symbols or verify markers, but it is not sufficient evidence for patch design. |
| Cross-project feedback loop | After each WeClaw integration round, produce a precise CoDeepSeedeX follow-up prompt for missing fields, ambiguous semantics, or unstable contract behavior. | WeClaw `p0.1.5a67-status-estcost-label`; CoDeepSeedeX `p2.10a59-weclaw-round3-token-attribution-plan` | in_progress | 2026-05-17 | WeClaw keeps `aux`, replaces the trailing `est` marker with an `Cost` label, and leaves pricing/token/compaction semantics unchanged. |
| Release readiness | README, handbooks, development log, focused tests, full tests, Release notes, and five platform assets are consistent before public Release publication. | Public pre-release `v0.1.9-alpha` at `6a5f10f` plus post-release doc finalization `p0.1.5a70-post-release-doc-finalize` | done | 2026-05-17 | `v0.1.9-alpha` was refreshed to a69 with five rebuilt assets. a70 only replaces temporary refreshing placeholders in handbooks and must not move the public Release tag. |

### Second-round CoDeepSeedeX contract acceptance

`p0.1.5a62-second-round-contract-acceptance-audit` is a WeClaw-side acceptance checkpoint for the original second-round CoDeepSeedeX request. It must not be interpreted as a new runtime feature branch or as a public Release. It records what WeClaw can safely consume from CoDeepSeedeX `p2.10a55-weclaw-runtime-status-contract` and what still belongs in a later CoDeepSeedeX round.

| ID | Requirement area | WeClaw acceptance | Evidence and boundary |
| --- | --- | --- | --- |
| A1 | dsproxy is authoritative for Codex profiles and DeepSeek runtime configuration. | Stage-accepted. | `dsproxy profile status deepseek-thinking --json` and `dsproxy status thinking --weclaw-json` provide structured `model`, `effort`, `context_window`, `health`, `tokens`, `cost`, `balance`, and `compaction` fields. |
| A2 | WeClaw must not directly edit `~/.codex/config.toml` in normal runtime-control paths. | Accepted for current `/effort` and `/status` paths. | `/effort` uses `dsproxy profile set-effort <profile> <effort> --json`; production paths must not add Codex profile repair logic. Test fixtures may still create temporary `.codex/config.toml` files. |
| A3 | WeClaw must not parse Codex profiles to infer profile, model, effort, context, token, cost, or compaction state when dsproxy provides a contract. | Stage-accepted with fallback discipline. | `/status` consumes `dsproxy status <route> --weclaw-json`; fallbacks are allowed only when the contract is unavailable and must be visibly degraded. |
| A4 | `/effort high|max` should express user intent and call dsproxy. | Accepted. | WeClaw calls `dsproxy profile set-effort deepseek-thinking max --json`; user output shows `max` and hides Codex-internal `xhigh`. |
| A5 | `/model` must not become a Codex profile authority. | Partial, documented boundary. | `/model` calls `dsproxy config set-model` and updates WeClaw local agent runtime/config as a display and runtime fallback. The dsproxy `effective_model` remains authoritative when shown through telemetry. |
| A6 | `/status` consumes dsproxy telemetry and does not fabricate context, cost, balance, or token values. | Accepted for current compact status. | `p0.1.5a61` reads `summary.total_tokens`, displays cost and balance from dsproxy, and only shows context used tokens when `context_window.used_tokens_available=true`. |
| A7 | `/balance` must remain compatible with dsproxy-owned balance. | Partial. | `/balance` still consumes `dsproxy balance` legacy JSON. It does not maintain pricing or balance itself, but a later branch should consider the richer `balance.status/reason/action/display` contract from `--weclaw-json`. |
| A8 | `/info` must remain diagnostics, not a hidden source of profile truth. | Partial. | `/info` can show WeClaw and dsproxy versions and uptime. Any future profile diagnostics should come from `dsproxy profile status --json` and should stay out of ordinary `/status`. |
| A9 | upgrade/start/resume/uninstall must preserve runtime state without corrupting dsproxy or Codex profile ownership. | Stage-accepted. | Current upgrade/start/resume paths operate on WeClaw binaries, pid/log files, runtime state, and session hints. They must not be expanded to mutate Codex profiles. |
| A10 | Third-round CoDeepSeedeX needs must be separated by priority. | Required before the next CoDeepSeedeX round. | See the candidate list below. |

Third-round CoDeepSeedeX candidates from this audit:

- `context_window.used_tokens` remains unavailable. WeClaw can display `—/limit`, but CoDeepSeedeX must later define a real source or declare this a long-term limitation.
- Prompt-subcategory attribution such as user, assistant history, tool, environment, runtime, and compaction summary is not available. Current taxonomy supports provider usage totals and dsproxy call-purpose attribution.
- Official pricing refresh is not implemented. `pricing.refresh.available=false` currently reports `official_live_pricing_refresh_not_implemented`.
- Model catalog context-window binding is not part of the WeClaw contract yet. The contract currently reports `model_catalog.available=false`.
- Semantic payload compaction is observable but not safe to enable. Current blockers include missing semantic audit, semantic policy dry-run, and semantic payload compaction events.
- A stable debug or verbose diagnostics contract is still needed if WeClaw should expose `reason`, `action`, `diagnostic_hint`, model conflict details, or degraded-field explanations outside ordinary `/status`.

Operational rule: do not start a third CoDeepSeedeX round merely because compact `/status` is correct. Start it only when the next WeClaw task requires one of the deferred fields above, or when a long-session/runtime test shows the current degraded behavior is insufficient.

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
weclaw public version: v0.1.9-alpha | <release-commit>
weclaw internal version: p0.1.5a60-release-v0.1.9-alpha | <release-commit>
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


## p0.1.5a72 status contract display adaptation

p0.1.5a72 adapts ordinary `/status` to the CoDeepSeedeX `p2.10a61` WeClaw contract. Context now displays a numerator only when `context_window.used_tokens_available=true`, marks estimated numerators with `est`, and uses `context_window.display_limit_tokens` or `context_window.limit_explanation.display_limit_tokens` as the denominator. Pricing now displays the dsproxy pricing source label, per-1M token prices, and update date. `bundled_official_docs_snapshot` must be shown as a bundled snapshot, not as a live official cache. Tokens, `aux`, `Cost`, Policy, Compact, Trim, Proxy, and Paths remain separate.


## p0.1.5a73 status display polish

p0.1.5a73 removes low-value source and estimate suffixes from ordinary `/status`: the Pricing line omits the `bundled_official_docs_snapshot` label while still showing per-1M prices and update date, and the Context line omits the visible `est` suffix while still consuming dsproxy's available numerator only. Compact and Trim now fall back to dsproxy runtime configuration denominators when report files are absent, and display `no report` instead of the invalid `0/-- chars`.


## p0.1.5a74 runtime payload guard display

p0.1.5a74 adapts WeChat `/status` to the CoDeepSeedeX `runtime_payload_guard` contract from `p2.10a62`. Compact and Trim now prefer real-time char counters from `runtime_payload_guard.compaction.current_chars` and `runtime_payload_guard.trimming.current_chars`, using `trigger_chars` and `max_context_chars` as denominators. The older config/report fallback remains only for runtimes that do not expose the new contract.


## p0.1.5a75 Policy keep label polish

p0.1.5a75 changes the ordinary `/status` Policy row from `keep 24` to `keep ⤒24 msgs`, making the retained-message count explicit while keeping the status layout compact.


## p0.1.5a79 pre-release install command prefix

p0.1.5a79 fixes the published pre-release install command format. The raw fixed-tag URL must be invoked through `curl -fsSL`:

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

Do not publish a bare URL piped to `sh`; it is not executable as a shell command.
