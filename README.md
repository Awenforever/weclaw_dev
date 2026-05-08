# WeClaw Dev

<p align="center">
  <strong>WeChat bridge for AI agents, optimized for Codex, DeepSeek and long-running chat workflows.</strong>
</p>

<p align="center">
  <a href="README.md">English</a> · <a href="README_CN.md">中文文档</a>
</p>

> `weclaw_dev` is a development fork of [`fastclaw-ai/weclaw`](https://github.com/fastclaw-ai/weclaw).
> It keeps the upstream WeChat AI Agent bridge model, while adding development-oriented behavior for Codex, DeepSeek, command formatting, session continuity and WeChat chat ergonomics.
> Personal learning and research use only.

---

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

## Language

- [English README](README.md)
- [中文文档](README_CN.md)

---

## Screenshots

Put images under `assets/readme/` with the exact filenames shown below. They will render automatically.

| Formatted command output | Long-text formatting |
| --- | --- |
| <img src="assets/readme/weclaw-dev-formatted-status.jpg" width="420" alt="WeClaw Dev formatted status command" /> | <img src="assets/readme/weclaw-dev-codex-session.jpg" width="420" alt="Long text formatting from Codex in WeChat" /> |

| Typing keepalive | Profile and balance |
| --- | --- |
| <img src="assets/readme/weclaw-dev-typing-keepalive.jpg" width="420" alt="Typing keepalive during long replies" /> | <img src="assets/readme/weclaw-dev-profile-balance.jpg" width="420" alt="Profile and balance command examples" /> |

| Codex session reuse | Typing keepalive |
| --- | --- |
| <img src="assets/readme/weclaw-dev-codex-session.jpg" width="420" alt="Codex session reuse from WeChat" /> | <img src="assets/readme/weclaw-dev-typing-keepalive.jpg" width="420" alt="Typing keepalive during long replies" /> |

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
| Formatting | Functional plain-text replies | Mobile-readable command summaries and less raw JSON dumping |
| Model workflow | Local agents and HTTP backends | Designed for Codex profiles, DeepSeek-backed Codex and CoDeepSeedeX |
| Typing state | Not the main focus | Keeps WeChat typing state alive during long-running agent turns where supported |
| Commands | Core commands such as `/help`, `/info`, `/cwd`, `/new` | Refined `/status`, `/help`, `/profile`, `/balance` output |
| Target users | General WeChat-to-agent users | Users operating agents, model profiles, proxies and tool workflows from WeChat |
| Stability | Upstream release line | Dev branch. Faster iteration and more frequent behavior changes |

---

## Quick start

```bash
curl -sSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh
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

## Installation notes

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

## Recommended Codex and DeepSeek workflow

For Codex ACP mode, use the real `codex` binary directly.

Do not wrap Codex with `tee`, stdout-capture scripts or logging shims unless persistent NDJSON logs are explicitly required. Wrapping can change runtime behavior and create unexpected local files.

Example Codex ACP config:

```json
{
  "agents": {
    "codex": {
      "type": "acp",
      "command": "/usr/local/bin/codex",
      "args": ["app-server", "--listen", "stdio://"]
    }
  }
}
```

Recommended separation with CoDeepSeedeX:

| Component | Responsibility |
| --- | --- |
| WeClaw Dev | WeChat login, message routing, chat commands and user-side bot behavior |
| CoDeepSeedeX | DeepSeek/Codex runtime backend, local proxy control, MCP bridge and upgrade path |
| Codex | Agent execution and project work |
| DeepSeek | Model backend through the configured profile/proxy |

CoDeepSeedeX:

```text
https://github.com/Awenforever/CoDeepSeedeX
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

## Chat commands

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
| `/info` | Show current agent information |

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

## Message formatting

WeChat is not a terminal. This fork therefore emphasizes readable mobile output:

- Markdown is converted into WeChat-readable text
- Code fences can be stripped when plain text is more useful
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

## Permission notes

Some CLI agents require interactive permission approval, which does not work well in WeChat.

| Agent | Flag | Meaning |
| --- | --- | --- |
| Claude CLI | `--dangerously-skip-permissions` | Skip interactive tool approval |
| Codex CLI | `--skip-git-repo-check` | Allow running outside a git repository |

Only use permission-bypass flags when you understand the security implications. ACP mode should be preferred when available.

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

## Update

```bash
weclaw update
weclaw version
```

---

## Development

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

## License

[MIT](LICENSE)
