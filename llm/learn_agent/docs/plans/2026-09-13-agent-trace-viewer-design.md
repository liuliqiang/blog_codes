# Agent Trace Viewer 设计

日期：2026-09-13

## 目标

agent 跑完后生成一个自包含的 `trace.html`，双击打开即可以完整时间线的形式回看整个运行过程（用户输入、每轮的 reasoning / 回复 / tool 调用 / tool 结果、错误）。零依赖，可直接发给别人或放进博客。

## 非目标

- 实时推送（SSE）——留给以后另写一个 Recorder 实现。
- 逐步播放器。
- markdown 渲染。

## 架构

```
agentloop.RunLoop ──回调──▶ Recorder 接口 ──┬─▶ stdoutRecorder（原 showResponse）
                                            └─▶ trace.Recorder ──▶ Trace ──WriteHTML──▶ trace.html
```

核心 loop 只认识 `Recorder` 接口，不认识 HTML。

## 事件模型（`01_agent_loop/trace`）

```go
type Event struct {
    Seq        int                    `json:"seq"`
    Time       time.Time              `json:"time"`
    Turn       int                    `json:"turn"`        // 第几轮 LLM 调用，从 1 开始；用户输入为 0
    Kind       string                 `json:"kind"`        // user | reasoning | text | tool_use | tool_result | error
    Text       string                 `json:"text,omitempty"`
    ToolName   string                 `json:"tool_name,omitempty"`
    ToolID     string                 `json:"tool_id,omitempty"`
    ToolInput  map[string]interface{} `json:"tool_input,omitempty"`
    DurationMS int64                  `json:"duration_ms,omitempty"` // tool_result 专用
}

type Trace struct {
    Model        string
    SystemPrompt string
    StartedAt    time.Time
    EndedAt      time.Time
    Events       []Event
}
```

## Recorder 接口（`agentloop`）

```go
type Recorder interface {
    OnStart(model Model, systemPrompt string, userMessages []Message)
    OnResponse(turn int, resp SendMessagesResponse)
    OnToolResult(turn int, toolUse MessagesBlock, output string, took time.Duration)
    OnEnd(err error)
}
```

- `NewAgent(llmClient, recorders ...Recorder)`，0 到多个。
- `RunLoop` 用 `defer` 保证 `OnEnd(err)` 一定被调。
- `showResponse` 原样搬进 `stdoutRecorder`。
- `trace.Recorder` 只攒数据，不自动写文件；何时写、写到哪由 main 决定。

## HTML 生成

- `trace/viewer.html` 通过 `//go:embed` 内嵌，占位符 `/*__TRACE_JSON__*/`。
- `WriteHTML` 把 `json.Marshal(trace)` 替换进 `<script type="application/json">`，`</` 转义为 `<\/` 防止截断。
- `main.go` 跑完写 `trace.html` 到 cwd，路径加入 `.gitignore`。

## 页面

- 顶部摘要：模型、开始时间、总耗时、轮次数、tool 调用次数。
- 竖向时间线按 turn 分组；reasoning 默认折叠；tool_result 紧跟 tool_use 并显示耗时，超过 20 行折叠，`error:` 开头标红；error 事件红色卡片。
- 纯 HTML/CSS/JS，跟随 `prefers-color-scheme`。

## 测试

- `trace`：回调序列 → Trace 内容；`WriteHTML` 的 JSON 注入与 `</script>` 转义；空 trace。
- `agentloop`：fake Recorder 断言回调顺序与参数；LLM 出错时 `OnEnd(err)` 仍被调用。
- 页面：生成样例 `trace.html` 人工/截图检查。
