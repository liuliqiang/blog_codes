package agentloop

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// useMCP installs a config and captures MCP output for the test.
func useMCP(t *testing.T, cfg MCPConfig) *bytes.Buffer {
	t.Helper()
	origCfg, origOut := MCP, mcpOut
	out := &bytes.Buffer{}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	MCP, mcpOut = cfg, out
	t.Cleanup(func() { MCP, mcpOut = origCfg, origOut })
	return out
}

// fakeTransport answers calls from a table instead of talking to a process.
type fakeTransport struct {
	replies map[string]interface{} // method → result, or an error to return
	calls   []string
	params  []map[string]interface{}
	closed  bool
}

func (f *fakeTransport) Call(_ context.Context, method string, params interface{}) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	raw, _ := json.Marshal(params)
	var asMap map[string]interface{}
	_ = json.Unmarshal(raw, &asMap)
	f.params = append(f.params, asMap)
	reply, ok := f.replies[method]
	if !ok {
		return json.RawMessage(`{}`), nil
	}
	if err, isErr := reply.(error); isErr {
		return nil, err
	}
	out, err := json.Marshal(reply)
	return out, err
}

func (f *fakeTransport) Notify(method string, _ interface{}) error {
	f.calls = append(f.calls, "notify:"+method)
	return nil
}

func (f *fakeTransport) Close() error { f.closed = true; return nil }

func docsTransport() *fakeTransport {
	return &fakeTransport{replies: map[string]interface{}{
		"initialize": map[string]interface{}{"protocolVersion": mcpProtocolVersion},
		"tools/list": map[string]interface{}{"tools": []map[string]interface{}{
			{"name": "search", "description": "Search the docs.", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}}, "required": []string{"query"}}},
			{"name": "get_version", "description": "Current API version."},
		}},
		"tools/call": map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": "hooks are documented in chapter 4"}}},
	}}
}

func TestNormalizeMCPName(t *testing.T) {
	for in, want := range map[string]string{
		"docs":            "docs",
		"docs.one":        "docs_one",
		"get.version":     "get_version",
		"a b/c":           "a_b_c",
		"--weird--":       "weird",
		"keep-dashes_ok1": "keep-dashes_ok1",
	} {
		if got := normalizeMCPName(in); got != want {
			t.Errorf("normalizeMCPName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := mcpToolName("docs.one", "get.version"); got != "mcp__docs_one__get_version" {
		t.Errorf("mcpToolName = %q", got)
	}
}

func TestMCPClient_ConnectAndCall(t *testing.T) {
	useMCP(t, MCPConfig{})
	tr := docsTransport()
	client, err := connectMCP(context.Background(), "docs", tr)
	if err != nil {
		t.Fatal(err)
	}
	// the handshake is initialize, the initialized notification, then discovery
	if strings.Join(tr.calls, ",") != "initialize,notify:notifications/initialized,tools/list" {
		t.Errorf("handshake = %v", tr.calls)
	}
	if v := tr.params[0]["protocolVersion"]; v != mcpProtocolVersion {
		t.Errorf("protocolVersion = %v", v)
	}
	defs := client.Tools()
	if len(defs) != 2 || defs[0].Name != "search" || defs[1].Name != "get_version" {
		t.Fatalf("tools = %+v", defs)
	}

	out, err := client.CallTool(context.Background(), "search", map[string]interface{}{"query": "hooks"})
	if err != nil || out != "hooks are documented in chapter 4" {
		t.Errorf("CallTool = %q, %v", out, err)
	}
	last := tr.params[len(tr.params)-1]
	if last["name"] != "search" {
		t.Errorf("tools/call params = %+v", last)
	}
	if args, _ := last["arguments"].(map[string]interface{}); args["query"] != "hooks" {
		t.Errorf("arguments = %+v", last["arguments"])
	}

	// a tool the server reports as failing is an error, not a result
	tr.replies["tools/call"] = map[string]interface{}{"isError": true, "content": []map[string]interface{}{{"type": "text", "text": "query is required"}}}
	if _, err := client.CallTool(context.Background(), "search", nil); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Errorf("isError = %v", err)
	}
	// so is a transport failure
	tr.replies["tools/call"] = &rpcError{Code: -32601, Message: "method not found"}
	if _, err := client.CallTool(context.Background(), "search", nil); err == nil || !strings.Contains(err.Error(), "method not found") {
		t.Errorf("rpc error = %v", err)
	}
	// empty content is reported as such rather than as an empty result
	tr.replies["tools/call"] = map[string]interface{}{"content": []map[string]interface{}{}}
	if out, err := client.CallTool(context.Background(), "search", nil); err != nil || out != "(no content)" {
		t.Errorf("empty content = %q, %v", out, err)
	}
}

func TestMCPClient_HandshakeFailures(t *testing.T) {
	useMCP(t, MCPConfig{})
	failing := &fakeTransport{replies: map[string]interface{}{"initialize": &rpcError{Code: -1, Message: "nope"}}}
	if _, err := connectMCP(context.Background(), "docs", failing); err == nil || !strings.Contains(err.Error(), "initialize docs") {
		t.Errorf("err = %v", err)
	}
	badList := &fakeTransport{replies: map[string]interface{}{
		"initialize": map[string]interface{}{},
		"tools/list": "not an object",
	}}
	if _, err := connectMCP(context.Background(), "docs", badList); err == nil || !strings.Contains(err.Error(), "tools/list docs") {
		t.Errorf("err = %v", err)
	}
}

func TestMCPRegistry_ToolsAndPrefixes(t *testing.T) {
	out := useMCP(t, MCPConfig{})
	r := NewMCPRegistry()
	docs, err := connectMCP(context.Background(), "docs", docsTransport())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.add("docs", docs); err != nil {
		t.Fatal(err)
	}
	if _, err := r.add("docs", docs); err == nil {
		t.Error("connecting the same server twice should fail")
	}

	// a second server offering the same tool name stays separate
	deploy, err := connectMCP(context.Background(), "deploy", &fakeTransport{replies: map[string]interface{}{
		"initialize": map[string]interface{}{},
		"tools/list": map[string]interface{}{"tools": []map[string]interface{}{{"name": "search"}, {"name": "trigger"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.add("deploy", deploy); err != nil {
		t.Fatal(err)
	}

	tools := r.Tools()
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	want := "mcp__docs__search,mcp__docs__get_version,mcp__deploy__search,mcp__deploy__trigger"
	if strings.Join(names, ",") != want {
		t.Errorf("tools = %v, want %s", names, want)
	}
	// a tool the server described without a schema still gets a usable one
	for _, tool := range tools {
		if tool.Name == "mcp__docs__get_version" {
			if tool.InputSchema["type"] != "object" || tool.Description == "" {
				t.Errorf("get_version = %+v", tool)
			}
		}
	}
	// each handler is bound to its own tool
	for _, tool := range tools {
		if tool.Name != "mcp__docs__search" {
			continue
		}
		if got, err := tool.Handler(context.Background(), map[string]interface{}{"query": "x"}); err != nil || !strings.Contains(got, "chapter 4") {
			t.Errorf("handler = %q, %v", got, err)
		}
	}
	if !strings.Contains(out.String(), "[mcp] connected docs with 2 tool(s)") {
		t.Errorf("out = %q", out.String())
	}
	if got := r.Names(); strings.Join(got, ",") != "docs,deploy" {
		t.Errorf("Names = %v", got)
	}
	r.Close()
	if len(r.Names()) != 0 {
		t.Error("Close should forget the servers")
	}
}

func TestMCPRegistry_CollisionsAndLongNames(t *testing.T) {
	out := useMCP(t, MCPConfig{})
	r := NewMCPRegistry()
	// two server-side names that normalize to the same model-facing name, plus one that is too long
	clashing, err := connectMCP(context.Background(), "docs", &fakeTransport{replies: map[string]interface{}{
		"initialize": map[string]interface{}{},
		"tools/list": map[string]interface{}{"tools": []map[string]interface{}{
			{"name": "get.version"},
			{"name": "get_version"},
			{"name": strings.Repeat("x", 70)},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.add("docs", clashing); err != nil {
		t.Fatal(err)
	}
	tools := r.Tools()
	if len(tools) != 1 || tools[0].Name != "mcp__docs__get_version" {
		t.Fatalf("tools = %+v, want only the first of the clashing pair", tools)
	}
	if !strings.Contains(out.String(), "normalized name mcp__docs__get_version already used by docs/get.version") {
		t.Errorf("collision not reported: %q", out.String())
	}
	if !strings.Contains(out.String(), "longer than 64 characters") {
		t.Errorf("long name not reported: %q", out.String())
	}
}

func TestMCPPolicy(t *testing.T) {
	useMCP(t, MCPConfig{Policy: map[string]string{
		"mcp__docs__search":    mcpPolicyAllow,
		"mcp__deploy__trigger": mcpPolicyConfirm,
		"mcp__deploy__wipe":    mcpPolicyDeny,
	}})
	for name, want := range map[string]string{
		"mcp__docs__search":    mcpPolicyAllow,
		"mcp__deploy__trigger": mcpPolicyConfirm,
		"mcp__deploy__wipe":    mcpPolicyDeny,
		"mcp__deploy__status":  mcpPolicyConfirm, // unconfigured external tools need confirmation
		"mcp__other__anything": mcpPolicyConfirm,
	} {
		if got := mcpPolicyFor(name); got != want {
			t.Errorf("mcpPolicyFor(%q) = %q, want %q", name, got, want)
		}
	}
	if isMCPTool("run_bash") || !isMCPTool("mcp__docs__search") {
		t.Error("isMCPTool")
	}
}

func TestMCPPermission(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	useMCP(t, MCPConfig{Policy: map[string]string{
		"mcp__docs__search": mcpPolicyAllow,
		"mcp__deploy__wipe": mcpPolicyDeny,
	}})
	origIn, origOut := promptIn, promptOut
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })
	var prompts bytes.Buffer
	promptIn, promptOut = yesReader{}, &prompts

	allowed, _ := checkToolPermission(context.Background(), MessagesBlock{Name: "mcp__docs__search", Input: map[string]interface{}{"query": "x"}})
	if !allowed {
		t.Error("an allowed tool should not prompt")
	}
	if strings.Contains(prompts.String(), "mcp__docs__search") {
		t.Error("an allowed tool must not reach the prompt")
	}
	if allowed, msg := checkToolPermission(context.Background(), MessagesBlock{Name: "mcp__deploy__wipe"}); allowed || !strings.Contains(msg, "host policy") {
		t.Errorf("denied tool = %v, %q", allowed, msg)
	}
	// unconfigured: the user is asked, and yesReader says yes
	if allowed, _ := checkToolPermission(context.Background(), MessagesBlock{Name: "mcp__deploy__status"}); !allowed {
		t.Error("unconfigured tool should be allowed after the user agrees")
	}
	if !strings.Contains(prompts.String(), "mcp__deploy__status") {
		t.Errorf("unconfigured tool did not prompt: %q", prompts.String())
	}
	// a scheduled turn has nobody to ask
	if allowed, _ := checkToolPermission(withNonInteractive(context.Background()), MessagesBlock{Name: "mcp__deploy__status"}); allowed {
		t.Error("a non-interactive turn must not allow an unconfigured external tool")
	}
	// the server cannot vouch for itself: a description claiming read-only changes nothing
	if allowed, _ := checkToolPermission(withNonInteractive(context.Background()), MessagesBlock{Name: "mcp__deploy__readOnlyHint"}); allowed {
		t.Error("server hints must not grant access")
	}
}

func TestMCPTools_Registration(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	useMCP(t, MCPConfig{Servers: []MCPServerConfig{{Name: "docs", Command: "true"}}})
	lead := NewAgent(newTeamLLM(), nil).(*agent)
	defer lead.team.Shutdown()

	for _, name := range []string{"connect_mcp", "list_mcp"} {
		if _, ok := lead.toolIndex[name]; !ok {
			t.Errorf("%s not registered", name)
		}
	}
	if !isAllowListed(MessagesBlock{Name: "list_mcp"}) {
		t.Error("list_mcp only reads the registry and should be allow-listed")
	}
	if isAllowListed(MessagesBlock{Name: "connect_mcp"}) {
		t.Error("connect_mcp starts a process and should need approval")
	}
	if !strings.Contains(lead.systemPrompt, "connect_mcp") {
		t.Error("system prompt lacks the MCP guidance")
	}

	ctx := withAgentName(context.Background(), "main")
	out, err := lead.toolIndex["list_mcp"].Handler(ctx, nil)
	if err != nil || !strings.Contains(out, "- docs [not connected]") {
		t.Errorf("list_mcp = %q, %v", out, err)
	}
	if _, err := lead.toolIndex["connect_mcp"].Handler(ctx, map[string]interface{}{"name": "nope"}); err == nil || !strings.Contains(err.Error(), "unknown mcp server") {
		t.Errorf("unknown server = %v", err)
	}

	// the registry is shared: a connection belongs to the process, not to one conversation
	if sub := lead.newSubagent(); sub.mcp != lead.mcp {
		t.Error("subagent should share the registry")
	}
	mate, err := lead.team.Spawn("alice", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if mate.agent.mcp != lead.mcp {
		t.Error("teammate should share the registry")
	}
}

func TestAssembleTools_GrowsAfterConnect(t *testing.T) {
	useTasksDir(t)
	useMCP(t, MCPConfig{})
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("first"), text("second")}}
	a := NewAgent(llm, nil).(*agent)

	if err := a.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	before := len(llm.tools[0])
	for _, tool := range llm.tools[0] {
		if isMCPTool(tool.Name) {
			t.Fatalf("no server is connected yet, but the model was offered %s", tool.Name)
		}
	}

	// connect between turns, as connect_mcp does
	docs, err := connectMCP(context.Background(), "docs", docsTransport())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.mcp.add("docs", docs); err != nil {
		t.Fatal(err)
	}
	if err := a.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	after := llm.tools[1]
	if len(after) != before+2 {
		t.Errorf("tool pool = %d, want %d", len(after), before+2)
	}
	found := map[string]bool{}
	for _, tool := range after {
		found[tool.Name] = true
	}
	if !found["mcp__docs__search"] || !found["mcp__docs__get_version"] {
		t.Errorf("external tools missing from the pool: %v", after)
	}
	// and the dispatch index knows them, so a call reaches the server
	if _, ok := a.toolIndex["mcp__docs__search"]; !ok {
		t.Error("toolIndex was not refreshed")
	}
	if out := a.runTool(withAgentName(context.Background(), "main"), MessagesBlock{Name: "mcp__docs__search", Input: map[string]interface{}{"query": "hooks"}}); !strings.Contains(out, "chapter 4") {
		t.Errorf("dispatch = %q", out)
	}
	// a failing call comes back as a tool result, not an aborted loop
	docs.transport.(*fakeTransport).replies["tools/call"] = &rpcError{Code: -32602, Message: "missing query"}
	if out := a.runTool(context.Background(), MessagesBlock{Name: "mcp__docs__search"}); !strings.HasPrefix(out, "error: mcp docs/search:") {
		t.Errorf("failed call = %q", out)
	}
}

// writeStdioServer writes a tiny newline-delimited JSON-RPC server as a shell script.
func writeStdioServer(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.sh")
	script := `#!/bin/bash
while IFS= read -r line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
  case "$line" in
    *'"initialize"'*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"echo"}}}\n' "$id" ;;
    *'"tools/list"'*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"echo","description":"Echo a message.","inputSchema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]}}\n' "$id" ;;
    *'"tools/call"'*)
      msg=$(printf '%s' "$line" | sed -n 's/.*"message":"\([^"]*\)".*/\1/p')
      printf '{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"echo: %s"}]}}\n' "$id" "$msg" ;;
    *'"notifications/initialized"'*) ;;
    *) printf '{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"method not found"}}\n' "$id" ;;
  esac
done
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMCP_StdioServerEndToEnd(t *testing.T) {
	server := writeStdioServer(t)
	useMCP(t, MCPConfig{
		Servers: []MCPServerConfig{{Name: "echo", Command: "bash", Args: []string{server}}},
		Policy:  map[string]string{"mcp__echo__echo": mcpPolicyAllow},
	})
	r := NewMCPRegistry()
	t.Cleanup(r.Close)

	client, err := r.Connect(context.Background(), "echo")
	if err != nil {
		t.Fatal(err)
	}
	defs := client.Tools()
	if len(defs) != 1 || defs[0].Name != "echo" || defs[0].Description != "Echo a message." {
		t.Fatalf("discovered = %+v", defs)
	}
	if _, err := r.Connect(context.Background(), "echo"); err == nil {
		t.Error("connecting twice should fail")
	}

	tools := r.Tools()
	if len(tools) != 1 || tools[0].Name != "mcp__echo__echo" {
		t.Fatalf("tools = %+v", tools)
	}
	out, err := tools[0].Handler(context.Background(), map[string]interface{}{"message": "hello"})
	if err != nil || out != "echo: hello" {
		t.Errorf("call = %q, %v", out, err)
	}
	// two calls in a row exercise id correlation
	if out, err := tools[0].Handler(context.Background(), map[string]interface{}{"message": "again"}); err != nil || out != "echo: again" {
		t.Errorf("second call = %q, %v", out, err)
	}
	// an unknown method comes back as an error
	if _, err := client.transport.Call(context.Background(), "resources/list", map[string]interface{}{}); err == nil || !strings.Contains(err.Error(), "method not found") {
		t.Errorf("unknown method = %v", err)
	}

	// once closed, further calls fail instead of hanging
	r.Close()
	if _, err := client.CallTool(context.Background(), "echo", map[string]interface{}{"message": "x"}); err == nil {
		t.Error("call after close should fail")
	}
}

func TestMCP_StdioServerStartFailures(t *testing.T) {
	useMCP(t, MCPConfig{
		Servers: []MCPServerConfig{
			{Name: "missing", Command: "definitely-not-a-real-binary-xyz"},
			{Name: "empty", Command: "  "},
			{Name: "silent", Command: "bash", Args: []string{"-c", "sleep 10"}},
		},
		Timeout: 150 * time.Millisecond,
	})
	r := NewMCPRegistry()
	t.Cleanup(r.Close)

	if _, err := r.Connect(context.Background(), "missing"); err == nil {
		t.Error("a missing binary should fail")
	}
	if _, err := r.Connect(context.Background(), "empty"); err == nil || !strings.Contains(err.Error(), "no command") {
		t.Errorf("empty command = %v", err)
	}
	// a server that never answers is bounded by the timeout rather than hanging the agent
	start := time.Now()
	if _, err := r.Connect(context.Background(), "silent"); err == nil {
		t.Error("a silent server should fail")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("Connect took %s, expected the timeout to bound it", took)
	}
	if len(r.Names()) != 0 {
		t.Errorf("failed connections must not be registered: %v", r.Names())
	}
}

func TestStdioTransport_OverPipes(t *testing.T) {
	useMCP(t, MCPConfig{})
	// drive the transport directly over pipes: no process, so the framing itself is under test
	clientReads, serverWrites := io.Pipe()
	serverReads, clientWrites := io.Pipe()
	tr := newStdioTransport(clientWrites, clientReads, nil)
	t.Cleanup(func() { _ = tr.Close() })

	// the server answers everything except "hang", which it never replies to
	go func() {
		dec := json.NewDecoder(serverReads)
		enc := json.NewEncoder(serverWrites)
		for {
			var req map[string]interface{}
			if err := dec.Decode(&req); err != nil {
				return
			}
			id, hasID := req["id"]
			if !hasID || req["method"] == "hang" {
				continue // a notification needs no reply, and "hang" deliberately gets none
			}
			_ = enc.Encode(map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": map[string]interface{}{"method": req["method"]}})
		}
	}()

	raw, err := tr.Call(context.Background(), "ping", map[string]interface{}{})
	if err != nil || !strings.Contains(string(raw), `"method":"ping"`) {
		t.Fatalf("Call = %s, %v", raw, err)
	}
	if err := tr.Notify("notifications/initialized", map[string]interface{}{}); err != nil {
		t.Errorf("Notify = %v", err)
	}
	// a call whose ctx is cancelled while the server stays silent gives up, and does not wedge the transport
	cancelled, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := tr.Call(cancelled, "hang", nil); err == nil {
		t.Error("cancelled call should fail")
	}
	if raw, err := tr.Call(context.Background(), "ping2", nil); err != nil || !strings.Contains(string(raw), "ping2") {
		t.Errorf("transport unusable after a cancelled call: %s, %v", raw, err)
	}
}
