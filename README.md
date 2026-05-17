# WeClaw Dev

## Current pre-release: v0.1.9-alpha

`v0.1.9-alpha` now includes the WeClaw `/status` telemetry refresh from the a72-a75 line, the pinned pre-release installer fix from a77, and the raw GitHub fixed-tag pre-release entry from a78.

Pre-release install must use the raw GitHub fixed-tag installer and pass the Release tag explicitly. Do not use jsDelivr fixed-tag URLs for pre-release installation, because jsDelivr can keep stale content after a public tag is moved during a pre-release refresh.

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

`v0.1.9-alpha` highlights:

- Context shows the dsproxy-reported context window numerator and display limit.
- Pricing shows current per-1M token prices and the pricing update date.
- Compact and Trim show real-time char-level progress from CoDeepSeedeX `runtime_payload_guard`.
- Policy displays the retained-message count as `keep ⤒24 msgs`.

For full `/status` telemetry, use CoDeepSeedeX `v0.3.9-alpha` or newer.

## What is WeClaw Dev?

WeClaw Dev connects WeChat messages to local or remote AI agents.

A WeChat message is received by WeClaw, routed to a configured agent, then normalized and sent back to WeChat. Agents can be local ACP processes, CLI commands or OpenAI-compatible HTTP backends.

This fork focuses on practical agent operation from WeChat:

- Codex sessions controlled from WeChat
- DeepSeek-backed Codex workflows through CoDeepSeedeX
- Cleaner mobile-readable command output
- Better current-session continuity
- WeChat typing-state keepalive during long agent turns
- Refined operational commands such as `/status`, `/help`, `/profile` and `/balance`

---

## Preview

| Command output | Long-form response formatting |
| --- | --- |
| <img src="assets/readme/weclaw-dev-formatted-status.jpg" width="420" alt="WeClaw Dev formatted status command" /> | <img src="assets/readme/weclaw-dev-codex-session.jpg" width="420" alt="Long-form Codex response formatting in WeChat" /> |

| Typing keepalive | Runtime settings |
| --- | --- |
| <img src="assets/readme/weclaw-dev-typing-keepalive.jpg" width="420" alt="Typing keepalive during long replies" /> | <img src="assets/readme/weclaw-dev-profile-balance.jpg" width="420" alt="Runtime settings command examples" /> |

---

## Quick start

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
weclaw start
```

On first start, WeClaw will:

1. Show a QR code for WeChat login
2. Detect installed AI agents where possible
3. Save config to `~/.weclaw/config.json`
4. Start receiving and replying to WeChat messages

Useful commands:

```bash
weclaw login
weclaw status
weclaw stop
weclaw start -f
```

---

<!-- WECLAW_DOCS_RESTRUCTURE:BEGIN:user_changes -->
## User-facing behavior changes

| Version | Affected area | Previous behavior | Current behavior |
| --- | --- | --- | --- |
| v0.1.9-alpha | CoDeepSeedeX telemetry integration | `/effort` and `/status` could only rely on partial local state or placeholders. | `/effort max` uses the dsproxy profile contract. `/status` consumes dsproxy WeClaw telemetry and now shows model, effort, token context, token usage, `EstCost`, balance, pricing source/update state, compaction policy, Compact/Trim, proxy and runtime paths. Requires CoDeepSeedeX >= v0.3.9-alpha when CoDeepSeedeX integration is used. |
| v0.1.7-alpha | Background logs | Background stdout/stderr required explicit setup in many flows. | `weclaw start` keeps a default background log at `~/.weclaw/weclaw.log`, with bounded trimming. |
| v0.1.7-alpha | Session resume | `resume` usually required an explicit ACP/Codex session ID. | `weclaw start deepseek-thinking resume` may omit the session ID and reuse the latest ACP/Codex session for that profile. |
| v0.1.7-alpha | Managed start | Starting an already running managed WeClaw process could be confusing. | `weclaw start ...` is idempotent and reports the existing managed process instead of replacing it. |
| v0.1.7-alpha | Alpha upgrade | Rebuilt assets under the same public tag could be missed by public-version-only checks. | `weclaw upgrade --alpha` compares build metadata and can migrate a running managed process when the same public tag points to a newer commit. |
| Unreleased main | Stop command | `weclaw stop` printed a generic stopped message and could leave users unsure whether stale runtime state or extra managed processes were cleared. | `weclaw stop` reports whether it stopped PIDs or found nothing, clears stale PID state, and verifies that no managed WeClaw process remains. |
| Unreleased main | WeChat slash commands | Command replies mixed languages and some outputs repeated fields or used verbose tables. | Slash command replies are English, compact and Markdown-first. `/status` shows profile, model, session, context, tokens, cost placeholder and current balance. `/balance` uses a fixed-width Total-only panel. `/info` shows WeClaw and dsproxy versions plus uptime. |

<!-- WECLAW_DOCS_RESTRUCTURE:END:user_changes -->

## Installation notes
<!-- weclaw-install-fallbacks -->

### Install endpoints and fallback

Prefer the jsDelivr endpoint:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

If jsDelivr is unavailable, use the GitHub raw endpoint:

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh
```

Uninstall uses the same endpoint pattern:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh -s -- --uninstall
```

Fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh -s -- --uninstall
```

This fork installs from:

```text
Awenforever/weclaw_dev
```

The installer prefers GitHub Release artifacts. If no release exists, it falls back to building from source.

Fallback source build requirements:

- `git` must be available
- `go` is optional. If missing, the installer can bootstrap a temporary Go toolchain

Other install methods:

```bash
go install github.com/Awenforever/weclaw_dev@latest
docker run -it -v ~/.weclaw:/root/.weclaw ghcr.io/fastclaw-ai/weclaw start
```

---

## Update and uninstall

### Upgrade path for older installs

> Since `v0.1.7-alpha`, `weclaw upgrade` preserves a running managed WeClaw session when possible. It downloads and installs the new binary first, then stops the old managed process and restarts the new version with the previous profile and ACP session.

If you installed an older alpha, first repeat the one-line installer to move onto the current release channel:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

After v0.1.2-alpha or newer is installed, use the built-in updater:

```bash
weclaw upgrade

# Alpha/pre-release channel, opt-in only:
weclaw upgrade --alpha

Since v0.1.4-alpha, `weclaw upgrade --alpha` opts into the latest pre-release/alpha channel; plain `weclaw upgrade` remains on the stable latest channel.
```

Session continuity after an update:

```bash
# During a running task, ask WeClaw for the current session/thread ID.
/now

# After upgrading or restarting, resume that Codex session/thread.
weclaw start deepseek resume <session-id>
weclaw start deepseek-thinking resume <session-id>
# Or omit the session ID to resume the most recent ACP/Codex session for this profile:
weclaw start deepseek-thinking resume
```

The resumed ID is applied to the first matching WeChat user turn for that profile. Use `/restart` or `/new` only when you intentionally want a fresh session.

Repeat the one-line installer at any time to install the latest GitHub Release:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

Use the built-in updater after WeClaw is installed:

```bash
weclaw upgrade
weclaw update
weclaw version
```

`weclaw version` and `weclaw --version` always print both public and internal metadata:

```text
weclaw public version: v0.1.4-alpha | <commit> (<os>/<arch>)
weclaw internal version: p0.1.5a33-internal-version-format-policy | <commit> (<os>/<arch>)
```

`weclaw start` checks GitHub Releases periodically and prints a one-time reminder when a newer release is available.

Uninstall only the binary and keep `~/.weclaw` user data:

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh -s -- --uninstall
weclaw uninstall
```

Remove the binary and local user data:

```bash
weclaw uninstall --purge
```

---

## Start with DeepSeek

WeClaw Dev provides DeepSeek-oriented startup modes.

```bash
# Start with the normal DeepSeek profile
weclaw start deepseek

# Start with the DeepSeek thinking profile
weclaw start deepseek-thinking
```

These are WeClaw startup modes, not WeChat slash commands. After startup, send normal messages in WeChat, or use the documented chat commands in the section below.

The intended flow is:

```text
WeChat message
→ WeClaw Dev
→ selected DeepSeek-backed Codex runtime
→ model response
→ formatted WeChat reply
```

Users normally do not need to hand-edit `~/.weclaw/config.json` for this path. The `deepseek` and `deepseek-thinking` startup modes are the intended entry points.

If startup fails, verify the underlying local runtime first. For example, check that the related Codex/CoDeepSeedeX profile works outside WeClaw before debugging WeClaw itself.

Recommended separation:

| Component | Responsibility |
| --- | --- |
| WeClaw Dev | WeChat login, message routing, chat commands and user-side bot behavior |
| Codex runtime | Agent execution through the selected startup mode |
| CoDeepSeedeX | DeepSeek-backed Codex runtime/proxy layer when used |
| DeepSeek | Model backend used by the selected runtime |

CoDeepSeedeX:

```text
https://github.com/Awenforever/CoDeepSeedeX
```

Compatibility note: WeClaw Dev `v0.1.9-alpha` requires CoDeepSeedeX `v0.3.9-alpha` or newer when CoDeepSeedeX integration is used.

CoDeepSeedeX:

```text
https://github.com/Awenforever/CoDeepSeedeX
```

---

## Session resume after restart or upgrade

WeClaw Dev can continue a Codex-backed conversation after a restart or upgrade when a session ID is available.

Common flow:

```bash
# Check the current session from WeChat.
/status
/now

# Restart WeClaw and ask it to resume that session.
weclaw start deepseek-thinking resume <session-id>
# Or omit the session ID to resume the most recent ACP/Codex session for this profile:
weclaw start deepseek-thinking resume
```

If the saved session cannot be reused, WeClaw now starts a fresh session automatically and continues handling the current message instead of leaving the chat stuck on a stale session.

For upgrades:

```bash
weclaw upgrade

# To explicitly follow the alpha/pre-release channel:
weclaw upgrade --alpha
```

<!-- WECLAW_DOCS_RESTRUCTURE:BEGIN:cli_commands -->
## WeClaw CLI commands

| Task | Command | Notes |
| --- | --- | --- |
| Start with DeepSeek thinking | `weclaw start deepseek-thinking` | Default daily-use start command for the thinking profile. |
| Resume the latest ACP/Codex session | `weclaw start deepseek-thinking resume` | Uses the most recent ACP/Codex session recorded for this profile. |
| Resume a specific ACP/Codex session | `weclaw start deepseek-thinking resume <session-id>` | Use the session/thread ID shown by `/now` or `/status`. |
| Stop managed processes | `weclaw stop` | Stops all detected managed WeClaw processes, clears stale PID state, and verifies no managed process remains. |
| Check runtime status | `weclaw status` | Shows managed process and runtime status from the CLI side. |
| Upgrade stable/latest channel | `weclaw upgrade` | Uses the normal latest Release channel. |
| Upgrade alpha channel | `weclaw upgrade --alpha` | Explicitly follows the alpha channel. |
| Show build metadata | `weclaw version` | Shows public and internal version metadata. |

<!-- WECLAW_DOCS_RESTRUCTURE:END:cli_commands -->

## WeChat slash commands

| Command | Description |
| --- | --- |
| `hello` | Send to the default agent |
| `/codex write a parser` | Route to Codex |
| `/cc explain this code` | Route through an alias |
| `/claude` | Switch default agent to Claude |
| `/cwd /path/to/project` | Change working directory |
| `/new` | Start a new conversation |
| `/status` | Show runtime and agent status |
| `/help` | Show concise command help |
| `/profile` | Show or reuse profile/session context where supported |
| `/balance` | Show backend balance where supported |

Default aliases:

| Alias | Agent |
| --- | --- |
| `/cc` | Claude |
| `/cx` | Codex |
| `/cs` | Cursor |
| `/km` | Kimi |
| `/gm` | Gemini |
| `/ocd` | OpenCode |
| `/oc` | OpenClaw |

Custom aliases:

```json
{
  "agents": {
    "claude": {
      "type": "acp",
      "aliases": ["ai", "c"]
    }
  }
}
```

---

## Configuration

Config file:

```text
~/.weclaw/config.json
```

Example:

```json
{
  "default_agent": "codex",
  "agents": {
    "codex": {
      "type": "acp",
      "command": "/usr/local/bin/codex",
      "args": ["app-server", "--listen", "stdio://"],
      "cwd": "/home/user/project"
    },
    "openclaw": {
      "type": "http",
      "endpoint": "https://api.example.com/v1/chat/completions",
      "api_key": "sk-xxx",
      "model": "openclaw:main"
    }
  }
}
```

Environment variables:

| Variable | Description |
| --- | --- |
| `WECLAW_DEFAULT_AGENT` | Override default agent |
| `OPENCLAW_GATEWAY_URL` | OpenClaw or compatible HTTP endpoint |
| `OPENCLAW_GATEWAY_TOKEN` | HTTP gateway token |
| `WECLAW_API_ADDR` | Proactive messaging API listen address |

---

## Background mode

```bash
weclaw start
weclaw start --stdout
weclaw status
weclaw stop
weclaw start -f
```

`weclaw start --stdout` writes stdout/stderr to `~/.weclaw/weclaw.log`.

---

## Permission notes

Some CLI agents require interactive permission approval, which does not work well in WeChat.

| Agent | Flag | Meaning |
| --- | --- | --- |
| Claude CLI | `--dangerously-skip-permissions` | Skip interactive tool approval |
| Codex CLI | `--skip-git-repo-check` | Allow running outside a git repository |

Only use permission-bypass flags when you understand the security implications. ACP mode should be preferred when available.

---

## Docker

```bash
docker build -t weclaw .
docker run -it -v ~/.weclaw:/root/.weclaw weclaw login
docker run -d --name weclaw \
  -v ~/.weclaw:/root/.weclaw \
  -e OPENCLAW_GATEWAY_URL=https://api.example.com \
  -e OPENCLAW_GATEWAY_TOKEN=sk-xxx \
  weclaw
docker logs -f weclaw
```

ACP and CLI agents require their binaries inside the container. HTTP mode works with a compatible remote or local HTTP endpoint.

---

## Message formatting

- Slash command replies use Markdown cards, tables and progress bars where the data benefits from structure.

WeChat is not a terminal. This fork therefore emphasizes readable mobile output:

- Markdown is preserved and normalized for WeChat ClawBot rich rendering
- Code fences, tables, headings, quotes, links, inline code and emphasis are kept where possible
- Links remain readable
- Status-like command output is summarized
- Large raw JSON payloads are avoided in normal chat replies
- `/status`, `/help`, `/profile` and `/balance` are formatted for quick reading

Use local logs for debugging large raw outputs.

---

## Typing-state keepalive

Long agent turns can make a WeChat bot look inactive. Where supported, WeClaw Dev keeps the WeChat typing state alive while the agent is still working.

Useful for:

- Codex reading or editing a project
- Long model calls
- Proxy backends waiting on tool calls
- Multi-step replies

This improves user feedback. It does not make the model faster.

---

## Media messages

WeClaw supports images, videos, files and voice messages.

Voice messages can be transcribed through WeChat speech-to-text and forwarded to the selected agent. Duplicate voice events are deduplicated where possible.

Agent replies containing Markdown image URLs can be extracted, downloaded and sent back to WeChat.

Supported examples:

- Images: `png`, `jpg`, `gif`, `webp`
- Videos: `mp4`, `mov`
- Files: `pdf`, `doc`, `zip`

---

## Proactive messaging

CLI:

```bash
weclaw send --to "user_id@im.wechat" --text "Hello from WeClaw"
weclaw send --to "user_id@im.wechat" --media "https://example.com/photo.png"
weclaw send --to "user_id@im.wechat" --text "Check this out" --media "https://example.com/photo.png"
```

HTTP API while `weclaw start` is running:

```bash
curl -X POST http://127.0.0.1:18011/api/send \
  -H "Content-Type: application/json" \
  -d '{"to": "user_id@im.wechat", "text": "Hello from WeClaw"}'
```

Change listen address:

```bash
export WECLAW_API_ADDR=0.0.0.0:18011
```

---

## How it works

<p align="center">
  <img src="assets/readme/weclaw-dev-architecture.jpg" width="720" alt="WeClaw Dev architecture" />
</p>

| Mode | How it works | Typical agents |
| --- | --- | --- |
| ACP | Long-running subprocess. JSON-RPC over stdio. Fastest because process and session can be reused | Claude, Codex, Gemini, Kimi, Cursor, OpenCode |
| CLI | Starts a new process per message. Some agents support session resume | Claude CLI, Codex exec |
| HTTP | OpenAI-compatible Chat Completions API | OpenClaw, custom gateways, local proxies |

When ACP and CLI are both available, WeClaw prefers ACP.

---

## WeClaw Dev vs upstream WeClaw

| Area | Upstream `fastclaw-ai/weclaw` | `Awenforever/weclaw_dev` |
| --- | --- | --- |
| Project role | General WeChat AI Agent bridge | Development fork for Codex, DeepSeek and command-driven WeChat workflows |
| Install source | `fastclaw-ai/weclaw` | `Awenforever/weclaw_dev` |
| Installer behavior | Standard install path | Release-first install, with source-build fallback |
| Agent modes | ACP, CLI and HTTP | Keeps ACP, CLI and HTTP, with additional Codex ACP attention |
| Codex usage | Basic Codex support | Use the real `codex` binary directly for ACP mode. Avoid stdout wrappers unless persistent NDJSON logging is explicitly required |
| Conversation | Basic routing and `/new` reset | Current-session reuse, agent/profile switching and WeChat-side continuity |
| Formatting | Functional plain-text replies | ClawBot Markdown-first replies with graceful plain-text readability |
| Model workflow | Local agents and HTTP backends | Designed for Codex profiles, DeepSeek-backed Codex and CoDeepSeedeX |
| Typing state | Not the main focus | Keeps WeChat typing state alive during long-running agent turns where supported |
| Commands | Core commands such as `/help`, `/info`, `/cwd`, `/new` | Refined `/status`, `/help`, `/profile`, `/balance` output |
| Target users | General WeChat-to-agent users | Users operating agents, model profiles, proxies and tool workflows from WeChat |
| Stability | Upstream release line | Dev branch. Faster iteration and more frequent behavior changes |

---

## Relationship to upstream

This fork keeps the core design of upstream WeClaw:

- WeChat login
- Message bridge
- ACP, CLI and HTTP access
- Chat commands
- Media handling
- Background runtime

The dev branch adds practical behavior for daily agent operation from WeChat.

```text
Upstream: https://github.com/fastclaw-ai/weclaw
Dev fork: https://github.com/Awenforever/weclaw_dev
```

---

## Development

Detailed maintainer notes are kept in [`docs/developer-handbook.md`](docs/developer-handbook.md).

```bash
make dev
go build -o weclaw .
./weclaw start
```

Recommended checks:

```bash
git status --short
go test ./...
git diff --check
```

---

## Codex sandbox dependency warning

If WeClaw reports a Codex sandbox dependency warning about `bubblewrap`, install the OS package and restart WeClaw:

```bash
sudo apt update
sudo apt install -y bubblewrap
weclaw stop
weclaw start deepseek-thinking
```

## License

[MIT](LICENSE)

---
- Slash command output uses Markdown as a visual design system: quotes as callouts, code fences as status panels, tables for compact structured data, and lists for mobile readability.
