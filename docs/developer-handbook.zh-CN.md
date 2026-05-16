# WeClaw Dev开发者交接手册

这是中文维护者镜像。英文版`docs/developer-handbook.md`是新AI对话的主要启动手册。详细时间线放在`docs/development-log.md`。

## 1. 当前可信状态

- 项目路径：`~/projects/weclaw-streaming`
- GitHub仓库：`Awenforever/weclaw_dev`
- 主分支：`main`
- 当前公开Release：`v0.1.8-alpha`
- 当前公开Release commit：`05cb93c`
- 当前Release对应内部标记：`p0.1.5a50-outbound-markdown-capture`，位于`05cb93c`
- 当前内部开发标签：`p0.1.5a56-mainline-tracker-audit-policy`
- 本次同步前最后一次审计基线：`main=origin/main=p0.1.5a55-cross-project-profile-boundary-docs=cad0437`
- 当前活动开发线：`p0.1.5a56-mainline-tracker-audit-policy`。精确commit必须用Git解析，不要相信复制到静态文档中的旧hash。
- 旧公开Release `v0.1.7-alpha`仍位于`31fa432`，不得移动。
- `v0.1.8-alpha`的GitHub Release标题为`WeClaw Dev v0.1.8-alpha`，不是draft，不是prerelease，并且已有五个平台资产。
- 预期Release资产：Linux amd64、Linux arm64、Darwin amd64、Darwin arm64和Windows amd64。
- p0.1.5a51只记录VM中GitHub Release资产下载失败和宿主机代理修复方案，不移动公开Release。
- 后续每个新任务都先做只读审计，确认`main`、`origin/main`、当前内部tag、公开Release tag、工作区干净状态以及开发手册和开发日志头部。

## 2. 长期主线任务检查表

任何跨对话任务只要改变范围、状态、预期指标或责任边界，都必须更新此表。每个新开发对话都应先核对本表，再判断当前推进位置。

状态词：`planned`、`in_progress`、`verified`、`blocked`、`done`、`superseded`。

| 主线任务 | 预期指标或验收条件 | 当前版本或来源 | 当前状态 | 最后维护日期 | 备注 |
| --- | --- | --- | --- | --- | --- |
| Full telemetry契约基线 | 本机`dsproxy`已为`deepseek`和`deepseek-thinking`提供profile status和WeClaw status JSON，并包含`model`、`effort`、`context_window`、`tokens`、`pricing`、`cost`、`balance`和`compaction`。 | CoDeepSeedeX `p2.10a48-weclaw-full-telemetry-contract`，WeClaw `p0.1.5a56`前审计 | verified | 2026-05-16 | 接受为第一版工程基线。WeClaw侧仍需集成。 |
| WeClaw责任边界 | 当`dsproxy`已提供结构化契约时，WeClaw在常规`/effort`、`/model`、`/status`或telemetry路径中不直接编辑Codex profile文件。 | WeClaw `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | 当前WeClaw代码仍存在profile修复逻辑，下一轮实现分支需要重构。 |
| `/effort`集成 | `/effort max`调用权威`dsproxy profile set-effort <profile> max --json`契约，并展示`effort.user_facing`或`effort.deepseek_reasoning_effort`。 | CoDeepSeedeX `p2.10a48`之后的WeClaw待实现项 | planned | 2026-05-16 | 不要把WeClaw扩展成全局Codex profile修复层。 |
| `/status`契约集成 | `/status`读取`dsproxy status <route> --weclaw-json`或等价HTTP接口，只展示返回字段。缺失或不可用字段必须明确降级。 | CoDeepSeedeX `p2.10a48`之后的WeClaw待实现项 | planned | 2026-05-16 | token级context和char级运行时compaction不能混合到同一个进度条，除非明确标注单位。 |
| Telemetry展示质量 | 微信移动端输出保持紧凑、Markdown优先，并从`dsproxy`展示model、effort、context window、token usage、estimated cost、balance和compaction。 | WeClaw待实现项 | planned | 2026-05-16 | WeClaw不维护模型价格、余额逻辑，也不估算prompt内部子类token。 |
| 证据优先审计纪律 | 源码和文档改动必须基于完整源码文件、完整主文档或完整函数/模块块级上下文，而不是孤立grep片段。 | `p0.1.5a56-mainline-tracker-audit-policy` | in_progress | 2026-05-16 | grep/rg只能用于定位符号或验证标记，不能作为补丁设计的充分证据。 |
| 跨项目反馈闭环 | 每轮WeClaw集成后，根据缺失字段、语义歧义或契约不稳定点，生成精确的CoDeepSeedeX后续需求prompt。 | 第一轮WeClaw集成后执行 | planned | 2026-05-16 | prompt必须基于实际WeClaw实现结果和运行日志。 |
| Release准备 | README、开发手册、开发日志、focused tests、full tests和本地运行时版本元数据在公开Release决策前保持一致。 | 当前公开Release仍为`v0.1.8-alpha`，位于`05cb93c` | planned | 2026-05-16 | 内部追踪表更新不得移动公开Release tag，不重建Release资产。 |

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
weclaw public version: v0.1.8-alpha | 05cb93c
weclaw internal version: p0.1.5a56-mainline-tracker-audit-policy | <current-commit>
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
