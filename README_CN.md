# WeClaw Dev

<p align="center">
  <strong>面向Codex、DeepSeek和长对话工作流优化的微信AI Agent桥接器。</strong>
</p>

<p align="center">
  <a href="README_CN.md">中文文档</a> · <a href="README.md">English</a>
</p>

> `weclaw_dev`是[`fastclaw-ai/weclaw`](https://github.com/fastclaw-ai/weclaw)的开发分支。
> 本分支保留上游“微信接入AI Agent”的核心模式，同时强化Codex、DeepSeek、命令格式化、会话连续性和微信聊天体验。
> 本项目仅限个人学习和研究使用。

---

## WeClaw Dev是什么？

WeClaw Dev用于把微信消息接入本地或远程AI Agent。

微信消息进入WeClaw后，会被路由到已配置的Agent，然后经过格式整理后发回微信。Agent可以是本地ACP进程、CLI命令，也可以是OpenAI兼容HTTP后端。

这个dev分支重点面向实际Agent工作流：

- 通过微信控制Codex会话
- 通过CoDeepSeedeX使用DeepSeek驱动Codex
- 在微信中获得更清晰的命令输出
- 改善当前会话连续性
- 长时间Agent任务期间保持微信typing状态
- 优化`/status`、`/help`、`/profile`和`/balance`等运维命令

---

## 效果预览

| 命令输出 | 长文本回复格式化 |
| --- | --- |
| <img src="assets/readme/weclaw-dev-formatted-status.jpg" width="420" alt="WeClaw Dev格式化状态命令" /> | <img src="assets/readme/weclaw-dev-codex-session.jpg" width="420" alt="Codex长文本回复在微信中的格式化效果" /> |

| typing状态保活 | Profile和Balance命令 |
| --- | --- |
| <img src="assets/readme/weclaw-dev-typing-keepalive.jpg" width="420" alt="长回复期间typing状态保活" /> | <img src="assets/readme/weclaw-dev-profile-balance.jpg" width="420" alt="Profile和Balance命令示例" /> |

---

## 快速开始

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
weclaw start
```

首次启动时，WeClaw会：

1. 显示微信登录二维码
2. 尽可能检测已安装的AI Agent
3. 保存配置到`~/.weclaw/config.json`
4. 开始接收并回复微信消息

常用命令：

```bash
weclaw login
weclaw status
weclaw stop
weclaw start -f
```

---

<!-- WECLAW_DOCS_RESTRUCTURE:BEGIN:user_changes -->
## 用户可感知行为变化

| 版本 | 影响对象 | 此前行为 | 当前行为 |
| --- | --- | --- | --- |
| v0.1.7-alpha | 后台日志 | 很多后台运行场景需要显式配置stdout/stderr保存。 | `weclaw start`默认将后台日志写入`~/.weclaw/weclaw.log`，并带有有限裁剪。 |
| v0.1.7-alpha | 会话恢复 | `resume`通常需要显式填写ACP/Codex session ID。 | `weclaw start deepseek-thinking resume`可以省略session ID，默认使用该profile最近一次ACP/Codex会话。 |
| v0.1.7-alpha | 受管启动 | 已有受管WeClaw进程运行时，再次start容易造成理解混乱。 | `weclaw start ...`保持幂等，报告已有受管进程，不直接替换。 |
| v0.1.7-alpha | alpha升级 | 同一公开tag被重建到新commit时，仅比较公开版本号可能漏升。 | `weclaw upgrade --alpha`会比较构建元数据，并在可恢复场景下迁移运行中的受管进程。 |
| 未发布main | stop命令 | `weclaw stop`只打印笼统的stopped信息，用户难以确认stale运行态或额外受管进程是否已清理。 | `weclaw stop`会报告是否停止了PID或没有发现进程，清理stale pid状态，并确认没有managed WeClaw进程残留。 |

<!-- WECLAW_DOCS_RESTRUCTURE:END:user_changes -->

## 安装说明
<!-- weclaw-install-fallbacks -->

### 安装入口与备用入口

推荐优先使用jsDelivr入口：

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

如果jsDelivr不可用，可改用GitHub raw入口：

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh
```

卸载也可以使用同样的入口：

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh -s -- --uninstall
```

备用入口：

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh -s -- --uninstall
```

本分支安装源为：

```text
Awenforever/weclaw_dev
```

安装脚本优先使用GitHub Release产物。如果当前仓库还没有Release，则自动回退为源码构建。

源码构建回退要求：

- 本地需要有`git`
- `go`不是强制要求。如果本地没有`go`，安装脚本可以自动临时引导Go工具链

其他安装方式：

```bash
go install github.com/Awenforever/weclaw_dev@latest
docker run -it -v ~/.weclaw:/root/.weclaw ghcr.io/fastclaw-ai/weclaw start
```

---

## 更新与卸载

### 旧版本升级路径

如果已经安装过较旧alpha版本，先重复运行一行安装命令，把本机切到当前release通道：

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

安装v0.1.2-alpha或更新版本后，后续可以使用内置更新命令：

```bash
weclaw upgrade
```

自v0.1.4-alpha起，可使用`weclaw upgrade --alpha`显式升级到pre-release/alpha通道；普通`weclaw upgrade`仍默认使用stable latest。

```bash
weclaw upgrade --alpha
```

更新后的会话连续性：

```bash
# 任务运行中，先在微信端查看当前session/thread ID。
/now

# 升级或重启后，用该ID恢复Codex会话。
weclaw start deepseek resume <session-id>
weclaw start deepseek-thinking resume <session-id>
# 或省略session ID，默认使用最近一次ACP/Codex会话ID：
weclaw start deepseek-thinking resume
```

该resume ID会应用到对应profile收到的第一个微信用户回合。只有明确需要新会话时，才使用`/restart`或`/new`。

可以随时重复运行一行安装命令来安装最新GitHub Release：

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
```

安装后推荐使用内置更新命令：

```bash
weclaw upgrade
weclaw update
weclaw version
```

`weclaw start`会定期检查GitHub Release。如果发现新版本，会提示用户运行`weclaw upgrade`。

> 自`v0.1.7-alpha`起，`weclaw upgrade`会在可恢复的情况下保留运行态。它会先下载并安装新版本，再停止旧的受管WeClaw进程，并使用原profile和ACP session自动启动新版本。

仅卸载二进制文件并保留`~/.weclaw`用户数据：

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh -s -- --uninstall
weclaw uninstall
```

同时删除二进制文件和本地用户数据：

```bash
weclaw uninstall --purge
```

---

## 使用DeepSeek启动

WeClaw Dev提供面向DeepSeek的启动模式。

```bash
# 使用普通DeepSeek profile启动
weclaw start deepseek

# 使用DeepSeek thinking profile启动
weclaw start deepseek-thinking
```

这两个是WeClaw启动模式，不是微信聊天里的slash命令。启动后，直接在微信中发送普通消息即可，或使用下方“聊天命令”小节中列出的真实命令。

基本链路是：

```text
微信消息
→ WeClaw Dev
→ 所选DeepSeek-backed Codex runtime
→ 模型回复
→ 格式化后的微信回复
```

普通用户通常不需要手动编辑`~/.weclaw/config.json`来完成这一路径。`deepseek`和`deepseek-thinking`启动模式就是面向直接使用的入口。

如果启动失败，应先检查底层本地runtime是否可用。例如，先确认相关Codex/CoDeepSeedeX profile能在WeClaw之外正常工作，再排查WeClaw本身。

推荐分工：

| 组件 | 职责 |
| --- | --- |
| WeClaw Dev | 微信登录、消息路由、聊天命令和用户侧bot行为 |
| Codex runtime | 通过所选启动模式执行Agent任务 |
| CoDeepSeedeX | 使用时提供DeepSeek-backed Codex runtime/proxy层 |
| DeepSeek | 所选runtime使用的模型后端 |

CoDeepSeedeX：

```text
https://github.com/Awenforever/CoDeepSeedeX
```

---

## 重启或升级后的会话恢复

WeClaw Dev可以在重启或升级后继续使用已有的Codex会话。只要你能从微信端看到当前session ID，就可以在重启时指定它。

常用流程：

```bash
# 先在微信端查看当前session。
/status
/now

# 重启WeClaw，并尝试恢复该session。
weclaw start deepseek-thinking resume <session-id>
# 或省略session ID，默认使用最近一次ACP/Codex会话ID：
weclaw start deepseek-thinking resume
```

如果旧session已经不能继续使用，WeClaw现在会自动开启新session并继续处理当前消息，不会让对话一直卡在失效session上。

升级时：

```bash
weclaw upgrade

# 如需显式进入alpha/pre-release通道：
weclaw upgrade --alpha
```

<!-- WECLAW_DOCS_RESTRUCTURE:BEGIN:cli_commands -->
## WeClaw CLI命令

| 任务 | 命令 | 说明 |
| --- | --- | --- |
| 使用DeepSeek thinking启动 | `weclaw start deepseek-thinking` | thinking profile的日常启动命令。 |
| 恢复最近一次ACP/Codex会话 | `weclaw start deepseek-thinking resume` | 使用该profile最近一次记录的ACP/Codex session。 |
| 恢复指定ACP/Codex会话 | `weclaw start deepseek-thinking resume <session-id>` | 使用`/now`或`/status`显示的session/thread ID。 |
| 停止受管进程 | `weclaw stop` | 停止所有检测到的受管WeClaw进程，清理stale pid状态，并确认没有managed进程残留。 |
| 查看运行状态 | `weclaw status` | 从CLI侧查看受管进程和运行状态。 |
| 普通通道升级 | `weclaw upgrade` | 使用普通Latest Release通道。 |
| alpha通道升级 | `weclaw upgrade --alpha` | 显式进入alpha通道。 |
| 查看构建元数据 | `weclaw version` | 显示公开版本和内部版本元数据。 |

<!-- WECLAW_DOCS_RESTRUCTURE:END:cli_commands -->

## 微信端slash commands

| 命令 | 说明 |
| --- | --- |
| `你好` | 发送给默认Agent |
| `/codex 写一个解析器` | 路由到Codex |
| `/cc 解释这段代码` | 通过别名路由 |
| `/claude` | 切换默认Agent为Claude |
| `/cwd /path/to/project` | 切换工作目录 |
| `/new` | 开始新对话 |
| `/status` | 查看运行状态和Agent状态 |
| `/help` | 查看精简命令帮助 |
| `/profile` | 在支持时查看或复用profile/session上下文 |
| `/balance` | 在后端支持时查看余额 |
| `/info` | 查看当前Agent信息 |

默认别名：

| 别名 | Agent |
| --- | --- |
| `/cc` | Claude |
| `/cx` | Codex |
| `/cs` | Cursor |
| `/km` | Kimi |
| `/gm` | Gemini |
| `/ocd` | OpenCode |
| `/oc` | OpenClaw |

自定义别名：

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

## 配置

配置文件：

```text
~/.weclaw/config.json
```

示例：

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

环境变量：

| 变量 | 说明 |
| --- | --- |
| `WECLAW_DEFAULT_AGENT` | 覆盖默认Agent |
| `OPENCLAW_GATEWAY_URL` | OpenClaw或兼容HTTP端点 |
| `OPENCLAW_GATEWAY_TOKEN` | HTTP网关Token |
| `WECLAW_API_ADDR` | 主动推送API监听地址 |

---

## 后台运行

```bash
weclaw start
weclaw start --stdout
weclaw status
weclaw stop
weclaw start -f
```

`weclaw start --stdout`会把stdout/stderr写入`~/.weclaw/weclaw.log`。

---

## 权限说明

部分CLI Agent需要交互式权限确认，不适合微信场景。

| Agent | 参数 | 含义 |
| --- | --- | --- |
| Claude CLI | `--dangerously-skip-permissions` | 跳过交互式工具权限确认 |
| Codex CLI | `--skip-git-repo-check` | 允许在非git仓库目录运行 |

只有在理解安全影响后，才应启用权限绕过参数。能用ACP时优先使用ACP。

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

ACP和CLI Agent需要容器内存在对应二进制文件。HTTP模式只需要兼容的远程或本地HTTP端点。

---

## 消息格式化

微信不是终端。这个分支更强调手机端可读性：

- 将Markdown回复转换为适合微信阅读的文本
- 在更适合纯文本展示时移除代码块围栏
- 保留链接可读文本
- 将状态类命令输出整理成摘要
- 常规聊天中避免直接倾倒大段原始JSON
- 让`/status`、`/help`、`/profile`和`/balance`更容易阅读

调试大输出时建议查看本地日志。

---

## typing状态保活

长时间Agent任务容易让微信bot看起来像“卡住了”。在支持的场景下，WeClaw Dev会在Agent仍在工作时保持微信typing状态。

适用场景包括：

- Codex正在读取或修改项目
- 模型调用耗时较长
- proxy后端正在等待tool call
- 回复需要分多步生成

typing保活不会让模型更快，它改善的是长任务期间的用户侧反馈。

---

## 富媒体消息

WeClaw支持图片、视频、文件和语音消息。

语音消息可通过微信语音转文字处理，再把转写文本发送给当前Agent。重复语音事件会尽可能去重。

Agent回复中的Markdown图片URL可以被提取、下载并发送回微信。

支持示例：

- 图片：`png`、`jpg`、`gif`、`webp`
- 视频：`mp4`、`mov`
- 文件：`pdf`、`doc`、`zip`

---

## 主动推送消息

CLI：

```bash
weclaw send --to "user_id@im.wechat" --text "Hello from WeClaw"
weclaw send --to "user_id@im.wechat" --media "https://example.com/photo.png"
weclaw send --to "user_id@im.wechat" --text "Check this out" --media "https://example.com/photo.png"
```

`weclaw start`运行时可使用HTTP API：

```bash
curl -X POST http://127.0.0.1:18011/api/send \
  -H "Content-Type: application/json" \
  -d '{"to": "user_id@im.wechat", "text": "Hello from WeClaw"}'
```

修改监听地址：

```bash
export WECLAW_API_ADDR=0.0.0.0:18011
```

---

## 工作原理

<p align="center">
  <img src="assets/readme/weclaw-dev-architecture.jpg" width="720" alt="WeClaw Dev架构图" />
</p>

| 模式 | 工作方式 | 典型Agent |
| --- | --- | --- |
| ACP | 长驻子进程，通过stdio进行JSON-RPC通信。速度最快，因为进程和会话可以复用 | Claude、Codex、Gemini、Kimi、Cursor、OpenCode |
| CLI | 每条消息启动一个新进程。部分Agent支持会话恢复 | Claude CLI、Codex exec |
| HTTP | OpenAI兼容Chat Completions API | OpenClaw、自定义网关、本地proxy |

当同一Agent同时存在ACP和CLI时，WeClaw优先选择ACP。

---

## WeClaw Dev与原始WeClaw的区别

| 维度 | 原始`fastclaw-ai/weclaw` | `Awenforever/weclaw_dev` |
| --- | --- | --- |
| 项目定位 | 通用微信AI Agent桥接器 | 面向Codex、DeepSeek和微信命令工作流的开发分支 |
| 安装来源 | `fastclaw-ai/weclaw` | `Awenforever/weclaw_dev` |
| 安装逻辑 | 标准安装路径 | 优先使用GitHub Release，无Release时回退源码构建 |
| Agent模式 | ACP、CLI和HTTP | 保留ACP、CLI和HTTP，并额外关注Codex ACP运行行为 |
| Codex使用方式 | 基础Codex支持 | ACP模式下直接使用真实`codex`可执行文件，除非明确需要持久NDJSON日志，否则不建议套`tee`或stdout抓取脚本 |
| 对话处理 | 基础路由和`/new`清空会话 | 更强调当前会话复用、Agent/Profile切换和微信侧命令连续性 |
| 命令格式化 | 可用的纯文本命令回复 | 面向微信阅读优化的紧凑命令摘要 |
| 模型与后端 | 本地Agent和HTTP兼容后端 | 更适合Codex profile、DeepSeek-backed Codex和CoDeepSeedeX联动 |
| typing状态 | 不是重点 | 长时间Agent回复期间尽量保持微信typing状态，降低“机器人卡死”的观感 |
| 运维命令 | `/help`、`/info`、`/cwd`、`/new`等基础命令 | 增强或整理`/status`、`/help`、`/profile`和`/balance`等命令输出 |
| 目标用户 | 普通微信接入Agent用户 | 需要从微信管理Agent、代理后端、模型profile和命令式工作流的用户 |
| 稳定性策略 | 上游release线 | dev分支，迭代更快，行为可能更频繁变化 |

---

## 与上游的关系

本分支保留上游WeClaw的核心设计：

- 微信登录
- 消息桥接
- ACP、CLI和HTTP接入
- 聊天命令
- 媒体消息处理
- 后台运行

dev分支增加的是日常从微信使用Agent时更需要的实用行为。

```text
上游项目：https://github.com/fastclaw-ai/weclaw
开发分支：https://github.com/Awenforever/weclaw_dev
```

---

## 开发

```bash
make dev
go build -o weclaw .
./weclaw start
```

推荐检查：

```bash
git status --short
go test ./...
git diff --check
```

---

## Codex sandbox依赖提示

如果WeClaw提示Codex缺少`bubblewrap`，先安装系统包，然后重启WeClaw：

```bash
sudo apt update
sudo apt install -y bubblewrap
weclaw stop
weclaw start deepseek-thinking
```

## 许可证

[MIT](LICENSE)
