# Context Compaction 设计

日期：2026-09-15
章节：`08_context_compact`
参考：[pi agent 主流程与特性分析 — Context Compaction](https://liqiang.dev/post/pi-agent-main-flow-and-features-analysis-en/#Context%20Compaction)

## 目标

对话越跑越长，context window 是固定的。当上下文接近上限时，把较早的一段对话压缩成一份结构化摘要，只保留最近的一段原文，让 agent 可以无限跑下去而不撑爆窗口。

要做到 pi 里的四个要点：

1. **触发**：超过 `window − reserveTokens` 时触发。
2. **切点**：从最新消息向前累加，累到 `keepRecentTokens` 处切；切点绝不能落在 tool result 上。
3. **摘要**：固定 6 段模板（Goal / Constraints & Preferences / Progress / Key Decisions / Next Steps / Critical Context）；再次压缩时传入上一份摘要做增量更新。
4. **文件清单**：`<read-files>` / `<modified-files>` 跨多次压缩累积——对话可以忘，碰过哪些文件不能忘。

## 非目标

- Session tree / fork / branch summary（pi 的树形 session 结构）。我们的 `a.messages` 是线性数组，压缩直接原地替换。
- 手动 `/compact` 命令（没有 REPL）。
- tool output 截断（pi 的另一层保护，属于另一个章节）。
- 在 `trace.html` 里可视化压缩事件——先用 stdout 日志，后面按需加 Recorder 事件。

## 架构

```
RunLoop 每轮开头
  │
  ├─ shouldCompact()?  ── lastUsage.Input + lastUsage.Output > window − reserve
  │        │ yes
  │        ▼
  │   compact(ctx)
  │     1. findCutPoint(messages)         → cut 索引
  │     2. summarize(ctx, messages[:cut]) → 调一次 LLM（无 tools）生成 6 段摘要
  │     3. 合并 read/modified 文件清单
  │     4. a.messages = [summaryMsg] + messages[cut:]
  │
  └─ SendMessages(...)
```

压缩只发生在 `RunLoop` 里两次 LLM 调用之间，不改动 hooks、tools、recorder 的接口。

## 组件

### 1. Usage 透传（`llm.go` / `llm/deepseek`）

触发条件要知道当前上下文有多大。不估算，直接用上一次响应里模型报的用量：

```go
type Usage struct {
    InputTokens  int
    OutputTokens int
}

type SendMessagesResponse struct {
    Content []MessagesBlock
    Usage   Usage
}
```

DeepSeek Responses API 的 `usage.input_tokens` / `usage.output_tokens` 解析进来。`InputTokens + OutputTokens` 就是下一轮请求的近似大小（下一轮 = 这轮的输入 + 这轮的输出 + 新的 tool result）。

### 2. 配置（`compact.go`）

```go
type CompactionConfig struct {
    ContextWindow    int // 模型窗口大小，默认 128_000
    ReserveTokens    int // 安全余量，默认 16_384（同 pi）
    KeepRecentTokens int // 至少保留多少最近原文，默认 20_000（同 pi）
}

var compaction = CompactionConfig{...}
```

沿用本项目"包级变量 + 测试里替换"的风格（同 `skillsDir`、`allowList`），`NewAgent` 签名不变。`ContextWindow <= 0` 表示关闭压缩。

### 3. 触发（`shouldCompact`）

```go
func (a *agent) shouldCompact() bool {
    if compaction.ContextWindow <= 0 || a.lastUsage == (Usage{}) {
        return false
    }
    return a.lastUsage.InputTokens+a.lastUsage.OutputTokens > compaction.ContextWindow-compaction.ReserveTokens
}
```

`a.lastUsage` 在每次 `SendMessages` 成功后更新；第一轮没有用量，不会触发。检查放在每轮循环开头、调 LLM 之前，这样压缩完的消息直接用于下一次请求。

### 4. 切点（`findCutPoint`）

```go
// 返回 cut：messages[:cut] 被摘要，messages[cut:] 原样保留。返回 0 表示不压缩。
func findCutPoint(messages []Message, keepRecentTokens int) int
```

算法：

1. 从最后一条消息向前累加 `estimateTokens(msg)`，累计值第一次 ≥ `keepRecentTokens` 时，那条消息的索引就是候选切点。
2. 如果候选切点落在 tool result 消息上（user 消息且 content 是 `[]MessagesBlock` 且含 `tool_result` block），继续向前移，直到落在一条不是 tool result 的消息上。这样带 tool_use 的 assistant 消息和它的结果一定一起保留，不会出现"有结果没调用"的孤儿。
3. 如果 `a.messages[0]` 是上一次的摘要消息，搜索范围从索引 1 开始（摘要不参与"被摘要"的原文，而是作为 `previousSummary` 单独传入）。
4. 切点 ≤ 搜索起点则返回 0：要保留的已经是全部，没东西可压。

`estimateTokens` 用 `len(文本)/4` 粗估：把 Content 字符串、各 block 的 Text、tool input 的 JSON、tool result 的 Content 全部算进去。它只用于找切点，不用于触发（触发用真实 usage），所以粗一点没关系。

### 5. 摘要（`summarize`）

用同一个 `LLMClient` 再发一次请求，不带 tools，不经过 hooks：

- system prompt：压缩指令 + 6 段模板。
- 输入：`messages[start:cut]` 序列化成纯文本对话记录——`user:` / `assistant:` 前缀，tool 调用写成 `[tool_use run_bash {"command": "..."}]`，tool 结果写成 `[tool_result <id>] ...`（超长结果截到 2000 字符，摘要不需要全文）。
- 有 `previousSummary` 时把它放在最前面，并指示"在这份摘要基础上更新，不要重写"。

固定模板：

```
## Goal
## Constraints & Preferences
## Progress
### Done
### In Progress
### Blocked
## Key Decisions
## Next Steps
## Critical Context
```

### 6. 文件清单（跨压缩累积）

不靠 LLM，程序扫描被压缩的那段消息里的 tool_use：

- `read_file` 的 `path` → `readFiles`
- `write_file` / `edit_file` 的 `path` → `modifiedFiles`

两个集合存在 agent 上，每次压缩取并集，然后拼在摘要末尾：

```
<read-files>
a.go
b.go
</read-files>
<modified-files>
b.go
</modified-files>
```

### 7. 插入位置

摘要作为一条 user 消息放到 `a.messages[0]`：

```go
Message{Role: MessageRoleUser, Content: "<summary>\n" + summary + "\n</summary>"}
```

后面直接跟 `messages[cut:]`。用 user 角色是因为它替代的是"用户到目前为止说了什么 + 发生了什么"，而且保证消息列表仍以 user 开头。下一次压缩时它被识别为 `previousSummary`，替换成新摘要，所以任何时刻列表里最多只有一份摘要。

agent 上新增的状态：

```go
type agent struct {
    ...
    lastUsage     Usage
    summary       string          // 当前摘要，"" 表示还没压缩过
    readFiles     map[string]bool
    modifiedFiles map[string]bool
}
```

`resetLoop` 里连同 `currLoop` 一起清零，子 agent 是新的 agent 实例，天然独立。

### 8. 可观测性与失败处理

- 压缩前后打一条日志到 `compactOut`（默认 stdout，测试可替换）：`[COMPACT] summarized 37 messages (~54k tokens) into 1.2k tokens, kept 12`。
- 摘要请求失败：打 error 日志，`a.messages` 不动，本轮照常继续。下一轮 usage 仍超线会再试；这是接受的行为，不做额外退避。
- `findCutPoint` 返回 0：不压缩，也不再重复尝试直到 usage 变化——实际上只要还在跑，消息一定在增长，下轮自然再判。

## 数据流示例

```
之前（usage 115k > 128k − 16k）：
  [0] user   "重构 X"
  [1] asst   tool_use read_file a.go
  [2] user   tool_result ...
  ...
  [36] asst  tool_use edit_file b.go      ← 从后向前累到 20k 落在 [37]，[37] 是 tool_result，
  [37] user  tool_result ...                 前移到 [36]，cut = 36
  ...
  [48] asst  "接下来..."

之后：
  [0] user   <summary>...6 段 + 文件清单...</summary>
  [1] asst   tool_use edit_file b.go
  [2] user   tool_result ...
  ...
```

## 测试

- `estimateTokens`：字符串 / block / tool input / tool result 都计入。
- `findCutPoint`：正常切；切点落在 tool_result 时前移到对应 assistant；全部都在 keepRecent 内返回 0；有上一份摘要时从 1 开始搜且不切到摘要。
- `shouldCompact`：关闭、无 usage、未超线、超线四种。
- `summarize`：用 `scriptedLLM` 断言发出的请求——无 tools、system prompt 含 6 段模板、输入含序列化对话、有 previousSummary 时带上。
- 文件清单：read / write / edit 归类正确，跨两次压缩累积。
- 端到端：`scriptedLLM` 返回带 `Usage` 的响应，跑到超线后下一次 LLM 调用收到的 messages 是 `[summary] + 保留段`，且列表里只有一份摘要；摘要请求失败时 messages 不变、loop 不中断。
- DeepSeek client：`usage` 解析。

## 实现顺序

1. `Usage` 类型 + deepseek 解析 → 单测。
2. `compact.go`：配置、`estimateTokens`、`findCutPoint` → 单测。
3. `summarize` + 文件清单 + `compact()` → 单测。
4. 接入 `RunLoop` → 端到端测试。
5. `main.go` 无需改动（默认配置即可）。
