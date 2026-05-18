## p0.1.5a80 status Details与Cost格式

p0.1.5a80在微信`/status`中新增`Details`行，只消费`dsproxy status <route> --weclaw-json`返回的`tokens.prompt_subcategory_split.categories`。WeClaw不得本地tokenize，不得按字符数估算，不得读取debug文件，也不得把session累计tokens当成context窗口占用。

显示规则：
- 当`tokens.prompt_subcategory_split.available=true`时，显示`Details  user~...  hist~...  tool~...  sys~...  dev~...  comp~...  other~...`
- 当`tokens.prompt_subcategory_split.reason=profile_tokenizer_available_but_no_observed_prompt`时，显示`Details  n/a · waiting first prompt`
- 当`tokens.profile_tokenizer.available=false`时，显示`Details  n/a · tokenizer unavailable`

费用行标签改为`Cost`，并使用紧凑`label~value`字段：`Cost     session~...  last~...  aux~...`。`last`、`session`、`aux`和费用仍以provider usage为权威来源。Compact和Trim仍是char级runtime payload guard。


# WeClaw Dev开发者交接手册

## p0.1.5a78 raw固定tag pre-release安装入口

p0.1.5a78将公开pre-release安装入口从jsDelivr固定tag URL改为raw GitHub固定tag URL。VM测试显示，即使purge成功，jsDelivr仍可能继续返回旧的`@v0.1.9-alpha/install.sh`内容。

pre-release Release notes使用：

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

jsDelivr仍可用于stable/latest的`@main`便捷入口，但公开pre-release tag被移动刷新后，不应把jsDelivr固定tag URL作为权威pre-release安装路径。


## p0.1.5a77 pre-release固定版本安装

p0.1.5a77修复面向用户的pre-release安装路径。`install.sh`在没有传入`--version`时会默认使用stable/latest GitHub Release通道。因此pre-release的Release notes和文档必须使用固定版本命令：

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

不要在pre-release notes中发布“不带`--version`的tagged installer管道命令”；那会安装stable/latest Release，而不是pre-release资产。


## p0.1.5a76 v0.1.9-alpha发布刷新

p0.1.5a76在a72-a75 `/status`遥测线完成后，刷新文档并将当前`v0.1.9-alpha` pre-release重新发布到最新mainline。该Release包含Context、Pricing、Compact、Trim和Policy展示改进。Release notes只写用户可见变化，不重复GitHub Release标题行。


这是中文维护者镜像。英文版`docs/developer-handbook.md`是新AI对话的主要启动手册。详细时间线放在`docs/development-log.md`。

## 1. 当前可信状态

- 项目路径：`~/projects/weclaw-streaming`
- GitHub仓库：`Awenforever/weclaw_dev`
- 主分支：`main`
- 当前Latest公开Release：`v0.1.8-alpha`
- 当前Latest公开Release commit：`05cb93c`
- 当前公开pre-release：`v0.1.9-alpha`，位于`<to-be-refreshed-by-p79>`。
- 当前Release对应内部标记：`p0.1.5a69-release-v0.1.9-alpha-refresh`，位于`6a5f10f`
- 当前内部开发标签：`p0.1.5a79-prerelease-install-command-curl-prefix`
- 本次同步前最后一次审计基线：`main=origin/main=p0.1.5a59-status-paths-restore=68ca2bb`
- 当前活动开发线：`p0.1.5a79-prerelease-install-command-curl-prefix`
- 旧公开Release `v0.1.7-alpha`仍位于`31fa432`，不得移动。
- 目标`v0.1.9-alpha` GitHub Release标题为`WeClaw Dev v0.1.9-alpha`，必须创建为pre-release，并在使用CoDeepSeedeX集成时要求CoDeepSeedeX `v0.3.9-alpha`或更新版本。
- `v0.1.8-alpha`的GitHub Release标题为`WeClaw Dev v0.1.8-alpha`，不是draft，不是prerelease，并且已有五个平台资产。
- 预期Release资产：Linux amd64、Linux arm64、Darwin amd64、Darwin arm64和Windows amd64。
- p0.1.5a51只记录VM中GitHub Release资产下载失败和宿主机代理修复方案，不移动公开Release。
- 后续每个新任务都先做只读审计，确认`main`、`origin/main`、当前内部tag、公开Release tag、工作区干净状态以及开发手册和开发日志头部。

## 2. 长期主线任务检查表

任何跨对话任务只要改变范围、状态、预期指标或责任边界，都必须更新此表。每个新开发对话都应先核对本表，再判断当前推进位置。

状态词：`planned`、`in_progress`、`verified`、`blocked`、`done`、`superseded`。

| 主线任务 | 预期指标或验收条件 | 当前版本或来源 | 当前状态 | 最后维护日期 | 备注 |
| --- | --- | --- | --- | --- | --- |
| Full telemetry契约基线 | 本机`dsproxy`已为`deepseek`和`deepseek-thinking`提供profile status和WeClaw status JSON，并包含`model`、`effort`、`context_window`、`tokens`、`pricing`、`cost`、`balance`和`compaction`。 | CoDeepSeedeX `v0.3.9-alpha` / `p2.10a55-weclaw-runtime-status-contract`，WeClaw `p0.1.5a59-status-paths-restore` | verified | 2026-05-17 | 运行时审计显示token usage、estimated cost、balance和model conflict display hint已由dsproxy提供。 |
| WeClaw责任边界 | 当`dsproxy`已提供结构化契约时，WeClaw在常规`/effort`、`/model`、`/status`或telemetry路径中不直接编辑Codex profile文件。 | WeClaw `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | `/effort`路径中的profile修复逻辑已在`p0.1.5a57-dsproxy-telemetry-contract`中移除。 |
| `/effort`集成 | `/effort max`调用权威`dsproxy profile set-effort <profile> max --json`契约，并展示`effort.user_facing`或`effort.deepseek_reasoning_effort`。 | WeClaw `p0.1.5a57-dsproxy-telemetry-contract` | verified | 2026-05-16 | 该路径不再由WeClaw直接编辑Codex profile文件。 |
| `/status`契约集成 | `/status`读取`dsproxy status <route> --weclaw-json`，并对缺失或不可用字段明确降级。 | WeClaw `p0.1.5a61-dsproxy-runtime-status-followup`加CoDeepSeedeX `p2.10a55-weclaw-runtime-status-contract` | verified | 2026-05-17 | 状态输出读取可用的usage/cost/balance字段，保持token级Context与usage ledger累计值分离，并恢复单行Paths。 |
| Telemetry展示质量 | 微信移动端输出保持紧凑、Markdown优先，并展示model、effort、context window、token usage、estimated cost、balance、compaction、proxy和单行paths。 | WeClaw `p0.1.5a61-dsproxy-runtime-status-followup` | verified | 2026-05-17 | WeClaw现在读取dsproxy的`summary.total_tokens`，保留estimated cost语义，并且不把`session_total`当作context used tokens。 |
| 证据优先审计纪律 | 源码和文档改动必须基于完整源码文件、完整主文档或完整函数/模块块级上下文，而不是孤立grep片段。 | `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | grep/rg只能用于定位符号或验证标记，不能作为补丁设计的充分证据。 |
| 跨项目反馈闭环 | 每轮WeClaw集成后，根据缺失字段、语义歧义或契约不稳定点，生成精确的CoDeepSeedeX后续需求prompt。 | WeClaw `p0.1.5a67-status-estcost-label`，CoDeepSeedeX `p2.10a59-weclaw-round3-token-attribution-plan` | in_progress | 2026-05-17 | WeClaw保留`aux`，将行尾`est`标记改为`Cost`标签，并保持pricing、token和compaction语义不变。 |
| Release准备 | README、开发手册、开发日志、focused tests、full tests、Release notes和五个平台资产在公开Release发布前保持一致。 | 公开pre-release `v0.1.9-alpha`位于`6a5f10f`，另有发布后文档收口`p0.1.5a70-post-release-doc-finalize` | done | 2026-05-17 | `v0.1.9-alpha`已在a69刷新并重建五个平台资产。a70只替换手册中的临时refreshing占位，不移动公开Release tag。 |

### 第二轮CoDeepSeedeX契约验收

`p0.1.5a62-second-round-contract-acceptance-audit`是WeClaw侧对原第二轮CoDeepSeedeX需求的验收检查点。它不是新的运行功能分支，也不是公开Release。它记录WeClaw可以安全消费CoDeepSeedeX `p2.10a55-weclaw-runtime-status-contract`的哪些内容，以及哪些内容仍属于后续CoDeepSeedeX轮次。

| 编号 | 需求范围 | WeClaw验收结论 | 证据和边界 |
| --- | --- | --- | --- |
| A1 | dsproxy是Codex profile和DeepSeek运行配置的权威方。 | 阶段性接受。 | `dsproxy profile status deepseek-thinking --json`和`dsproxy status thinking --weclaw-json`已经提供结构化`model`、`effort`、`context_window`、`health`、`tokens`、`cost`、`balance`和`compaction`字段。 |
| A2 | WeClaw在普通运行控制路径中不得直接编辑`~/.codex/config.toml`。 | 当前`/effort`和`/status`路径接受。 | `/effort`使用`dsproxy profile set-effort <profile> <effort> --json`。生产路径不得重新加入Codex profile修复逻辑。测试夹具可以创建临时`.codex/config.toml`。 |
| A3 | dsproxy已有契约时，WeClaw不得解析Codex profile来推断profile、model、effort、context、token、cost或compaction状态。 | 阶段性接受，并保留降级纪律。 | `/status`读取`dsproxy status <route> --weclaw-json`。只有契约不可用时才允许显式降级。 |
| A4 | `/effort high|max`只表达用户意图并调用dsproxy。 | 接受。 | WeClaw调用`dsproxy profile set-effort deepseek-thinking max --json`。用户输出展示`max`，隐藏Codex内部`xhigh`。 |
| A5 | `/model`不能成为Codex profile权威源。 | 部分满足，需要记录边界。 | `/model`调用`dsproxy config set-model`，同时更新WeClaw本地agent运行态和配置作为显示或运行fallback。通过telemetry展示时，dsproxy `effective_model`仍是权威值。 |
| A6 | `/status`消费dsproxy telemetry，不伪造context、cost、balance或token值。 | 对当前紧凑`/status`接受。 | `p0.1.5a61`读取`summary.total_tokens`，显示dsproxy提供的cost和balance，并且只有`context_window.used_tokens_available=true`时才显示context used tokens。 |
| A7 | `/balance`必须兼容dsproxy维护的balance。 | 部分满足。 | `/balance`仍消费`dsproxy balance`旧JSON。它不维护价格或余额，但后续分支应考虑消费`--weclaw-json`中的`balance.status/reason/action/display`。 |
| A8 | `/info`只能作为诊断入口，不能成为隐藏profile真相来源。 | 部分满足。 | `/info`可以显示WeClaw和dsproxy版本、uptime。未来profile诊断应来自`dsproxy profile status --json`，且不进入普通`/status`。 |
| A9 | upgrade/start/resume/uninstall必须保留运行态，且不破坏dsproxy或Codex profile所有权。 | 阶段性接受。 | 当前upgrade/start/resume路径处理WeClaw二进制、pid/log文件、runtime state和session hint。不得扩展为修改Codex profile。 |
| A10 | 第三轮CoDeepSeedeX需求必须按优先级拆分。 | 下一轮前必须完成。 | 见下方候选列表。 |

本轮审计得到的第三轮CoDeepSeedeX候选需求：

- `context_window.used_tokens`仍不可用。WeClaw可以显示`—/limit`，但CoDeepSeedeX后续需要定义真实来源，或明确这是长期限制。
- user、assistant history、tool、environment、runtime和compaction summary等prompt子类归因仍不可用。当前taxonomy只能表达provider usage总量和dsproxy调用purpose归因。
- 官方价格刷新仍未实现。当前`pricing.refresh.available=false`，原因为`official_live_pricing_refresh_not_implemented`。
- model catalog context window尚未纳入WeClaw契约。当前契约报告`model_catalog.available=false`。
- semantic payload compaction可观测但尚不能启用。当前blockers包括semantic audit、semantic policy dry-run和semantic payload compaction事件缺失。
- 如果WeClaw需要在普通`/status`之外展示`reason`、`action`、`diagnostic_hint`、model conflict细节或降级字段解释，还需要稳定debug或verbose诊断契约。

操作规则：不要因为紧凑`/status`当前正确就认为第三轮没有必要。只有当下一个WeClaw任务需要上述deferred字段，或长会话/运行时测试证明当前降级行为不足时，才启动第三轮CoDeepSeedeX需求。

## 3. 文件地图

| 路径 | 作用 |
| --- | --- |
| `cmd/root.go` | CLI根命令和版本显示。 |
| `cmd/start.go` | `weclaw start`、profile启动、受管进程行为和resume入口。 |
| `cmd/update.go` | 升级流程、通道选择、Release资产下载和运行态迁移。 |
| `cmd/status.go` | CLI侧状态显示。 |
| `messaging/handler.go` | 微信侧消息处理、slash命令、`/now`、`/status`和session显示。 |
| `agent/acp_agent.go` | ACP/Codex app-server集成、thread启动和恢复、事件处理。 |
| `agent/agent.go` | Agent抽象和供应商共享运行逻辑。 |
| `runtime_state/` | 运行态持久化、最近profile、最近ACP/Codex session和迁移元数据。 |
| `config/` | profile和配置读取。 |
| `install.sh` | 用户安装脚本和下载fallback。 |
| `.github/workflows/` | 构建和Release自动化。 |
| `README.md` | 英文用户文档，使用优先，技术细节后置。 |
| `README_CN.md` | 中文用户文档，与英文README保持实用结构一致。 |
| `docs/developer-handbook.md` | 英文AI启动交接手册。 |
| `docs/developer-handbook.zh-CN.md` | 中文维护者镜像。 |
| `docs/development-log.md` | 可回溯详细开发日志。 |

## 4. 开发协作契约

- 用户在本机或VM执行命令并上传日志。
- 不猜源码结构、运行态、日志或配置。
- 证据不足时先给只读审计命令。
- 当源码或文档改动需要结构判断时，应优先要求用户直接上传完整源码文件、完整主文档或完整函数/模块块级上下文。grep或rg片段只能作为定位辅助，不能作为补丁设计的充分证据。
- 补丁前必须记录已审阅文件或块、expected markers、forbidden markers、验证规则和测试断言。若上下文不足，应先做全文审计，再补丁。
- 默认给可复制命令或补丁脚本，不要求手动打开文件替换。
- 长输出写入`/tmp/*.txt`，终端只显示`run_ok`、`out`、行数、字节数和简短tail。
- 每个审计或补丁日志必须包含`stage`、`prefix`、`branch`、`status_count`、`head`、`run_ok`和`out`。
- 命令中不得包含`exit`、`logout`这类会结束shell会话的操作。
- 复杂heredoc必须使用足够长且唯一的分隔符，外层Markdown代码块必须长于内部代码块。
- 修改前必须检查`git status --short`。
- 关键文件补丁前先在`/tmp`备份或保存diff。
- 使用`work/<version-description>`分支。
- 完成修改必须commit。
- 推送工作分支时同步推送内部tag。
- 合并到`main`必须在验证后fast-forward。

## 5. 版本、tag和Release规则

- 公开Release tag使用`v0.1.x-alpha`。
- 内部开发和handoff tag必须使用`p<major>.<minor>.<patch>[aN[aM...]][-topic]`。
- 内部tag必须以`p`开头，不得以`v`开头，并且可选`aN`修复/子版本后缀之前必须包含三段数字版本号。
- 合规示例包括`p0.1.5-topic`、`p0.1.5a1-topic`、`p0.1.5a1a3-topic`和`p0.1.5a33-internal-version-format-policy`。
- 旧的预规范化内部tag格式已经废弃。历史追溯通过规范镜像tag和开发日志维护。
- 不要静默移动公开Release tag。
- 更新已有公开Release tag必须明确决定删除并重建该Release和tag。
- 除非用户明确要求，否则旧公开Release tag必须保持稳定。
- 内部tag可以累计，但不能作为公开用户安装目标。
- 发布Release前，开发机必须与当前`main`提交同步。
- Release资产必须覆盖全部预期平台。

## 6. Release note规则

- GitHub Release页面已有标题，Release正文不要再写`WeClaw Dev v0.1.7-alpha`这类重复标题行。
- 正文从`Highlights`、`Changes`、`Fixes`、`Install`或`Validation`等部分开始。
- 如果只是Release note正文有问题，只编辑Release正文，不重建tag和资产。
- Release note应描述用户可见行为和验证状态。
- 除非解释用户可见变化或已知风险，否则不要堆实现细节。

## 7. VM中通过宿主机代理下载GitHub Release资产

- 根因模式：VM可以访问`github.com`、`api.github.com`、`codeload.github.com`和jsDelivr，但GitHub Release资产会跳转到`release-assets.githubusercontent.com`，该链路在VM网络中可能超时。
- `weclaw upgrade`是Go HTTP客户端。它读取`HTTP_PROXY`、`HTTPS_PROXY`、`http_proxy`和`https_proxy`，不会读取`git config http.*.proxy`。
- 已验证的VMware NAT环境中，Windows宿主机可通过`192.168.231.1`访问，正确代理为`http://192.168.231.1:7892`。
- VM中残留的`192.168.231.1:7896`Git代理配置是不充分且容易误导的，因为它只影响Git，不影响`weclaw upgrade`。
- 推荐VM修复方式：持久化用户级`~/.weclaw/proxy.env`，从`~/.profile`和`~/.bashrc`加载，并将Git代理同步更新到同一个可达宿主机代理。
- 持久化后验证命令：直接运行`weclaw upgrade`，应能正常报告`Already up to date`或升级到当前公开Release。

## 8. 文档维护规则

- README面向用户，应把安装、升级、启动、恢复、CLI命令和微信端slash命令放在技术细节之前。
- WeClaw CLI命令集中放置。
- 微信端slash commands集中放置。
- 内部设计、开发流程和Release流程不要放入README，除非直接影响用户使用。
- 英文handoff作为AI新对话启动手册。
- 中文handoff作为维护者阅读版本。
- 长时间线放入`docs/development-log.md`。
- 不为过期入口、幽灵引用或旧测试保留无意义指针文档。删除废弃入口，并让测试或文档引用适配真实主文档体系。
- 主文档限定为`README.md`、`README_CN.md`、`docs/developer-handbook.md`、`docs/developer-handbook.zh-CN.md`和`docs/development-log.md`。
- 每当里程碑永久改变用户工作流或已有CLI行为时，在README表格新增一行，交代版本、影响对象、此前行为和当前行为。
- 每次Release前确认开发机和当前`main` commit保持同步。

- 保持微信ClawBot回复Markdown优先。纯文本转换只用于内部分类或保守fallback逻辑，不作为主要出站路径。
- p0.1.5a42将微信端slash command统一改为Markdown精排版，包括表格、章节标题和context window进度条。

## 9. 最近一个大版本上下文

`v0.1.7-alpha`收束p5a20到p5a25主线。

用户可见变化：

- 后台模式默认保存日志到`~/.weclaw/weclaw.log`。
- `weclaw start deepseek-thinking resume`可以省略session ID，默认使用该profile最近一次ACP/Codex会话。
- 已有受管进程运行时，`weclaw start ...`保持幂等。
- Codex恢复使用稳定`thread/resume`，随后调用`turn/start`。
- 默认`thread/resume`不再发送`excludeTurns`。
- `/now`和`/status`共享profile和session解析逻辑。
- `/status`应尽量避免裸显示unknown上下文窗口值。
- `weclaw upgrade --alpha`可以识别同公开tag但commit更新的升级，并迁移受管进程。
- `weclaw stop`现在会报告停止了哪些PID或当前未运行，清理stale pid状态，并确认没有managed进程残留。

## 10. 经验总结

- 直接覆盖运行中的二进制可能触发`Text file busy`，应采用停止、构建、写入新路径、原子替换。
- 只有thread id不足以让新的Codex app-server进程真正恢复会话，必须调用`thread/resume`。
- `thread/resume.excludeTurns`需要实验能力，稳定默认路径不能使用。
- `turn` ID和session/thread ID不是同一概念，UI显示turn时应写成`last turn id`。
- alpha阶段可能出现同一公开tag指向不同commit的重建，升级逻辑不能只比较公开版本号。
- Release note清理通常不需要重建资产。
- ACP raw stdout诊断有价值，但默认应关闭。
- stop命令必须显式：说明停止了什么，清理stale pid状态，并确认后续`start`不会立刻看到同一个managed进程。

- p0.1.5a39之后，内部tag治理完成：内部开发使用规范化`p*.*.*`tag，CI不再自动发布分支/tag pre-release，公开Release必须通过手动`release.yml`流程发布。GitHub Release `targetCommitish`可能显示`main`，因此Release commit校验以tag目标为准。所有预规范化内部`v`前缀ref已在确认规范镜像覆盖后删除。

## 11. 后续方向

- 持续保持README面向用户。
- 保持handoff简洁。
- 详细历史继续进入`docs/development-log.md`。
- 如果dsproxy后续开放token attribution接口，WeClaw应在`/status`中读取，并在接口不存在时优雅降级。
- p0.1.5a43将Markdown语法作为视觉表达工具使用：引用块可作为提示条，代码围栏可作为状态面板，表格用于紧凑结构化数据，移动端帮助优先使用列表。

- p0.1.5a45保持微信端slash command英文、紧凑、Markdown优先；`/info`作为隐藏兼容入口，`/status`没有可靠数据源时不得伪造session费用。
- p0.1.5a46将`/balance`保持为固定宽度且仅显示Total的面板，在`/status`中显示当前账户余额但不伪造session cost，并让`/info`显示运行时版本和uptime。
- p0.1.5a47移除`/status`中agent type前的`@`标记，恢复context进度条宽度，并将CNY余额在Cost行紧凑显示为`￥...`。
- 小型后续修复避免继续使用长Python heredoc补丁。优先使用短shell命令、单行脚本，或在补丁目标不确定时先索取精确源码片段。
- p0.1.5a49增加动态Markdown围栏安全：生成的外层围栏必须长于内容中任意连续反引号，分块时不能让较短的内部围栏关闭较长的外部围栏。
- p0.1.5a50增加通过`WECLAW_CAPTURE_OUTBOUND_MARKDOWN_DIR`启用的outbound Markdown捕获，用于对比实际发给ClawBot的`TextItem.Text`和微信端渲染结果。

## 本地运行时重构建版本元数据规则

从源码checkout替换真实本机`weclaw`运行时时，不要安装普通`go build`产物。普通本地构建会显示`dev | unknown`，这会让运行时诊断和开发移交状态变得不可信。

本地运行时替换必须通过ldflags注入版本元数据，当前使用以下符号：

```text
github.com/fastclaw-ai/weclaw/cmd.Version
github.com/fastclaw-ai/weclaw/cmd.PublicCommit
github.com/fastclaw-ai/weclaw/cmd.InternalVersion
github.com/fastclaw-ai/weclaw/cmd.InternalCommit
```

本地开发构建的期望版本格式为：

```text
weclaw public version: v0.1.9-alpha | <release-commit>
weclaw internal version: p0.1.5a60-release-v0.1.9-alpha | <release-commit>
```

替换`/usr/local/bin/weclaw`或其他真实运行时二进制之前，必须先对候选二进制执行`weclaw version`。替换后必须再次检查已安装二进制的版本输出。只要出现`dev | unknown`，就视为安装失败，即使该二进制本身可以运行。

公开Release tag和commit必须与内部开发tag和commit分离。本地开发构建可以在public行显示当前公开Release，同时在internal行显示当前内部tag。

## 跨项目profile所有权边界

WeClaw必须把CoDeepSeedeX / `dsproxy`视为Codex profile文件和DeepSeek运行配置的权威维护者。WeClaw可以表达用户意图，例如`/effort max`，但不应直接编辑`~/.codex/config.toml`来补偿`dsproxy`的profile写入缺陷。

2026-05-15的effort调试确认了一个具体失败模式：

```text
dsproxy config set-effort max
```

在受影响的CoDeepSeedeX版本中，该命令会把Codex profile字段写成：

```toml
model_reasoning_effort = "max"
```

Codex在解析整个`~/.codex/config.toml`文件时会拒绝该值。Codex接受`none`、`minimal`、`low`、`medium`、`high`和`xhigh`，而DeepSeek侧effort语义是`high`和`max`。因此正确的所有权模型是：

```text
WeClaw用户意图：/effort max
dsproxy DeepSeek/env状态：DEEPSEEK_REASONING_EFFORT=max
dsproxy Codex profile状态：model_reasoning_effort="xhigh"
```

不要把WeClaw扩展成通用Codex profile修复层。如果某个profile绑定值错误，应修复CoDeepSeedeX契约，然后让WeClaw消费该契约。该原则适用于effort、model、profile状态、context窗口元数据、token遥测、cost、pricing、balance和compaction状态。

后续WeClaw开发应遵循：

- 不把`~/.codex/config.toml`作为权威数据源来解析，除非`dsproxy`尚未提供契约且任务明确要求临时审计
- 不在WeClaw维护模型价格或余额逻辑
- 不在WeClaw内部推断user/tool/environment/history token分类
- 不把`.debug/`报告当成稳定公共API读取
- 要求`dsproxy`提供机器可读的CLI或HTTP JSON接口，然后由WeClaw负责微信端展示
- WeClaw只负责消息入口、路由、会话UX和Markdown排版


## p0.1.5a72 status契约展示适配

p0.1.5a72适配CoDeepSeedeX `p2.10a61`的WeClaw契约。普通`/status`只在`context_window.used_tokens_available=true`时展示Context numerator，并用`est`标明估算值。Context分母来自`context_window.display_limit_tokens`或`context_window.limit_explanation.display_limit_tokens`。Pricing行展示dsproxy返回的价格来源、每100万tokens单价和更新时间。`bundled_official_docs_snapshot`必须显示为随包官方快照，不能说成实时官网缓存。Tokens、`aux`、`Cost`、Policy、Compact、Trim、Proxy和Paths继续保持分离。


## p0.1.5a73 status展示精简

p0.1.5a73精简普通`/status`中的低价值来源和估算后缀：Pricing行不再显示`bundled_official_docs_snapshot`标签，但继续显示每100万tokens单价和更新时间；Context行不再显示`est`后缀，但仍只消费dsproxy明确可用的numerator。Compact和Trim在报告文件不存在时回退到dsproxy运行时配置分母，并显示`no report`，避免继续出现无效的`0/-- chars`。


## p0.1.5a74 runtime payload guard展示适配

p0.1.5a74适配CoDeepSeedeX `p2.10a62`新增的`runtime_payload_guard`契约。普通`/status`中的Compact和Trim优先使用`runtime_payload_guard.compaction.current_chars`和`runtime_payload_guard.trimming.current_chars`作为实时char级numerator，并分别使用`trigger_chars`和`max_context_chars`作为分母。旧的config/report回退只保留给尚未提供新契约的运行时。


## p0.1.5a75 Policy保留数量标签精简

p0.1.5a75将普通`/status`中的Policy行从`keep 24`改为`keep ⤒24 msgs`，在不拉长布局的前提下明确这是保留消息数量。


## p0.1.5a79 pre-release安装命令前缀修正

p0.1.5a79修正公开pre-release安装命令格式。raw固定tag URL必须通过`curl -fsSL`调用：

```bash
curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/v0.1.9-alpha/install.sh | sh -s -- --version v0.1.9-alpha
```

不要发布裸URL直接管道给`sh`的命令；那不是可执行shell命令。
