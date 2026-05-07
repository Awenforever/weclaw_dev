# WeClaw / CoDeepSeedeX hand-off notes

更新时间：2026-05-07

## 1. 当前可信稳定点

当前主线已合并并推送：

- `v0.1d-codex-model-provider-thread-fix`
- `v0.1e-slash-output-polish`
- `v0.1f-acp-raw-stdout-log`

`v0.1f-acp-raw-stdout-log`新增了可选ACP raw stdout诊断功能，用于查看Codex app-server给WeClaw返回的原始NDJSON事件。

## 2. DeepSeek输出结构结论

已通过ACP raw日志确认，`deepseek`经Codex app-server返回的事件结构与原生Codex基本一致。

典型事件链如下：

````text
item/started
item/agentMessage/delta
item/completed
mcpToolCall started/completed
turn/completed
````

最终回答位于：

````text
item.type = "agentMessage"
item.text = "..."
````

`item.text`会保留：

````text
\n\n
Markdown表格
列表
代码片段
````

因此当前不需要为DeepSeek单独实现一套特殊格式化逻辑。现有WeClaw基于`agentMessage`的段落分割和微信输出格式化方案原则上可以继续复用。

如果微信端再次出现“全部挤成一行”的问题，应优先排查：

1. 是否运行了旧WeClaw二进制
2. 是否没有走stream/completed assistant message路径
3. 是否是slash命令直接回显JSON，而不是模型输出格式问题
4. 模型本身是否没有输出段落换行
5. 微信日志摘要是否被截断导致误判

## 3. ACP raw stdout诊断功能

可选环境变量：

````bash
WECLAW_ACP_RAW_LOG=1
````

启用后，`ACPAgent.readLoop()`会把从Codex app-server stdout读取到的原始NDJSON逐行打印为：

````text
[acp-raw] stdout line: ...
````

该功能默认关闭，不影响正常运行。

需要亲眼查看Codex到底给WeClaw返回什么时，使用：

````bash
weclaw stop || true
WECLAW_ACP_RAW_LOG=1 weclaw start --stdout deepseek
tail -f ~/.weclaw/weclaw.log | grep --line-buffered "\[acp-raw\]"
````

说明：

- `--stdout`负责把后台WeClaw的stdout/stderr写入`~/.weclaw/weclaw.log`
- `WECLAW_ACP_RAW_LOG=1`负责让ACP readLoop打印Codex app-server原始NDJSON
- 两者缺一不可
- 正常使用时不要设置`WECLAW_ACP_RAW_LOG=1`

## 4. 正式安装标准操作

此前发生过：

````text
cp: cannot create regular file '/usr/local/bin/weclaw': Text file busy
````

该错误会导致新构建的二进制没有真正替换`/usr/local/bin/weclaw`，运行中的仍是旧二进制。

后续正式恢复或安装时，必须采用“停进程→从main构建→原子替换”的流程：

````bash
cd ~/projects/weclaw-streaming

weclaw stop || true
pkill -f "/tmp/weclaw-rawlog start" || true
pkill -f "codex --profile deepseek app-server" || true
sleep 2

git switch main
git pull --ff-only
./.tools/go/bin/go build -o /tmp/weclaw-main .

mkdir -p /tmp/weclaw-backup
cp /usr/local/bin/weclaw "/tmp/weclaw-backup/weclaw.before-install.$(date +%Y%m%d_%H%M%S).bak"
install -m 755 /tmp/weclaw-main /usr/local/bin/weclaw.new
mv -f /usr/local/bin/weclaw.new /usr/local/bin/weclaw

weclaw start --stdout deepseek
````

不要直接用：

````bash
cp /tmp/weclaw-main /usr/local/bin/weclaw
````

因为目标二进制可能正在运行，容易触发`Text file busy`。

## 5. 确认当前运行的是正确二进制

如果怀疑仍在运行旧版，核验三处：

````bash
sha256sum /tmp/weclaw-main 2>/dev/null || true
sha256sum /usr/local/bin/weclaw

PID="$(cat ~/.weclaw/weclaw.pid 2>/dev/null || true)"
echo "PID=$PID"

sha256sum "/proc/$PID/exe" 2>/dev/null || true
readlink -f "/proc/$PID/exe" 2>/dev/null || true
strings "/proc/$PID/exe" | grep -E "WECLAW_ACP_RAW_LOG|\[acp-raw\]" || true
tr '\0' '\n' < "/proc/$PID/environ" | grep WECLAW_ACP_RAW_LOG || true
````

判断标准：

- `/usr/local/bin/weclaw`与`/proc/$PID/exe`哈希一致，说明当前进程来自已安装二进制
- `strings /proc/$PID/exe`能看到`WECLAW_ACP_RAW_LOG`和`[acp-raw]`，说明运行中二进制包含raw诊断功能
- `/proc/$PID/environ`中有`WECLAW_ACP_RAW_LOG=1`，说明raw开关已进入daemon环境

## 6. 当前任务结论

本轮排查结论：

1. DeepSeek输出结构与原生Codex基本一致
2. 当前不需要修改DeepSeek模型输出格式化逻辑
3. 已新增可选raw stdout诊断开关
4. 后续安装必须使用原子替换，避免`Text file busy`
5. 若再出现微信输出挤压，应先使用raw诊断确认Codex原始NDJSON，再决定是否修改格式化层
